package main_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/kenee101/go-test/internal/app"
	"github.com/kenee101/go-test/internal/config"
)

const integrationDB = "taskdb_integration_test"

var integrationServer *httptest.Server

func TestMain(m *testing.M) {
	_ = godotenv.Load(".env")

	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}

	client := connectWithRetry(uri, 5, 2*time.Second)
	if client == nil {
		if os.Getenv("CI") != "" {
			log.Fatal("integration tests: could not reach MongoDB")
		}
		log.Println("integration tests skipped: could not reach MongoDB")
		os.Exit(0)
	}

	db := client.Database(integrationDB)

	cfg := &config.Config{JWTSecret: "integration-test-secret"}
	integrationServer = httptest.NewServer(app.NewRouter(cfg, db))

	code := m.Run()

	integrationServer.Close()
	_ = db.Drop(context.Background())
	_ = client.Disconnect(context.Background())

	os.Exit(code)
}

func connectWithRetry(uri string, maxAttempts int, delay time.Duration) *mongo.Client {
	for i := range maxAttempts {
		client, err := mongo.Connect(options.Client().ApplyURI(uri))
		if err != nil {
			log.Printf("db connect attempt %d/%d failed: %v", i+1, maxAttempts, err)
			time.Sleep(delay)
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		pingErr := client.Ping(ctx, nil)
		cancel()
		if pingErr == nil {
			return client
		}
		log.Printf("db ping attempt %d/%d failed: %v", i+1, maxAttempts, pingErr)
		_ = client.Disconnect(context.Background())
		time.Sleep(delay)
	}
	return nil
}

// do sends a JSON request to the integration server.
func do(t *testing.T, method, path string, body any, token string) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req, err := http.NewRequest(method, integrationServer.URL+path, &buf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func decode(t *testing.T, resp *http.Response, dst any) {
	t.Helper()
	defer resp.Body.Close()
	require.NoError(t, json.NewDecoder(resp.Body).Decode(dst))
}

// TestFullAuthAndTaskFlow exercises the complete lifecycle.
func TestFullAuthAndTaskFlow(t *testing.T) {
	// 1. Register a regular user.
	resp := do(t, http.MethodPost, "/register", map[string]string{
		"username": "integration_user",
		"email":    "integration@example.com",
		"password": "pass1234",
	}, "")
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// 2. Login by username.
	resp = do(t, http.MethodPost, "/login", map[string]string{
		"username": "integration_user",
		"password": "pass1234",
	}, "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var loginResp map[string]string
	decode(t, resp, &loginResp)
	token := loginResp["token"]
	assert.NotEmpty(t, token)

	// 3. Login by email — same credentials, different identifier.
	resp = do(t, http.MethodPost, "/login", map[string]string{
		"email":    "integration@example.com",
		"password": "pass1234",
	}, "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// 4. Create a task.
	resp = do(t, http.MethodPost, "/tasks", map[string]any{
		"title":       "Integration task",
		"description": "Created during integration test",
		"completed":   false,
	}, token)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	var createdTask map[string]any
	decode(t, resp, &createdTask)
	taskID, _ := createdTask["id"].(string)
	assert.NotEmpty(t, taskID)

	// 5. List tasks — should contain the one just created.
	resp = do(t, http.MethodGet, "/tasks", nil, token)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var tasks []map[string]any
	decode(t, resp, &tasks)
	assert.NotEmpty(t, tasks)

	// 6. Get single task.
	resp = do(t, http.MethodGet, "/tasks/"+taskID, nil, token)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// 7. Partial update — only change the title, leave description and completed untouched.
	resp = do(t, http.MethodPut, "/tasks/"+taskID, map[string]any{
		"title": "Updated title only",
	}, token)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// 8. Delete task.
	resp = do(t, http.MethodDelete, "/tasks/"+taskID, nil, token)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	// 9. Confirm deleted task returns 404.
	resp = do(t, http.MethodGet, "/tasks/"+taskID, nil, token)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}

func TestAuthEndpoints(t *testing.T) {
	t.Run("duplicate username rejected", func(t *testing.T) {
		payload := map[string]string{
			"username": "dup_user",
			"email":    "dup@example.com",
			"password": "pass",
		}
		do(t, http.MethodPost, "/register", payload, "").Body.Close()
		resp := do(t, http.MethodPost, "/register", payload, "")
		defer resp.Body.Close()
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("duplicate email rejected", func(t *testing.T) {
		do(t, http.MethodPost, "/register", map[string]string{
			"username": "email_user1",
			"email":    "shared@example.com",
			"password": "pass",
		}, "").Body.Close()
		resp := do(t, http.MethodPost, "/register", map[string]string{
			"username": "email_user2",
			"email":    "shared@example.com",
			"password": "pass",
		}, "")
		defer resp.Body.Close()
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("login with wrong password", func(t *testing.T) {
		do(t, http.MethodPost, "/register", map[string]string{
			"username": "wrongpass_user",
			"email":    "wrongpass@example.com",
			"password": "correct",
		}, "").Body.Close()

		resp := do(t, http.MethodPost, "/login", map[string]string{
			"username": "wrongpass_user",
			"password": "wrong",
		}, "")
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("missing fields returns 400", func(t *testing.T) {
		resp := do(t, http.MethodPost, "/register", map[string]string{
			"username": "nopass",
		}, "")
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestProtectedRoutes(t *testing.T) {
	t.Run("no token returns 401", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/tasks", nil, "")
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("invalid token returns 401", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/tasks", nil, "not-a-valid-jwt")
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("non-admin cannot access /admin/tasks", func(t *testing.T) {
		do(t, http.MethodPost, "/register", map[string]string{
			"username": "regular_user2",
			"email":    "regular2@example.com",
			"password": "pass",
		}, "").Body.Close()
		resp := do(t, http.MethodPost, "/login", map[string]string{
			"username": "regular_user2",
			"password": "pass",
		}, "")
		var lr map[string]string
		decode(t, resp, &lr)

		resp = do(t, http.MethodGet, "/admin/tasks", nil, lr["token"])
		defer resp.Body.Close()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}

func TestUpdateTask_PartialFields(t *testing.T) {
	// Register + login.
	do(t, http.MethodPost, "/register", map[string]string{
		"username": "partial_update_user",
		"email":    "partial@example.com",
		"password": "pass",
	}, "").Body.Close()
	resp := do(t, http.MethodPost, "/login", map[string]string{
		"username": "partial_update_user",
		"password": "pass",
	}, "")
	var lr map[string]string
	decode(t, resp, &lr)
	token := lr["token"]

	// Create task.
	resp = do(t, http.MethodPost, "/tasks", map[string]any{
		"title":       "Original title",
		"description": "Original description",
		"completed":   false,
	}, token)
	var created map[string]any
	decode(t, resp, &created)
	taskID := created["id"].(string)

	// Update only the completed field.
	resp = do(t, http.MethodPut, "/tasks/"+taskID, map[string]any{
		"completed": true,
	}, token)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Verify title and description are unchanged.
	resp = do(t, http.MethodGet, "/tasks/"+taskID, nil, token)
	var task map[string]any
	decode(t, resp, &task)
	assert.Equal(t, "Original title", task["title"])
	assert.Equal(t, "Original description", task["description"])
	assert.Equal(t, true, task["completed"])

	t.Run("empty body returns 400", func(t *testing.T) {
		resp := do(t, http.MethodPut, "/tasks/"+taskID, map[string]any{}, token)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
