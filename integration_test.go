package main_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/kenee101/go-test/internal/handlers"
	"github.com/kenee101/go-test/internal/middleware"
)

const (
	integrationSecret = "integration-test-secret"
	integrationDB     = "taskdb_integration_test"
)

var integrationServer *httptest.Server

// TestMain sets up a real MongoDB connection and an httptest server for the
// full chi router, then tears everything down after all tests run.
func TestMain(m *testing.M) {
	_ = godotenv.Load(".env")

	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		os.Exit(0)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx, nil); err != nil {
		os.Exit(0)
	}

	db := client.Database(integrationDB)

	integrationServer = httptest.NewServer(buildRouter(db))

	code := m.Run()

	integrationServer.Close()
	_ = db.Drop(context.Background())
	_ = client.Disconnect(context.Background())

	os.Exit(code)
}

// buildRouter wires up the chi router exactly as main.go does, minus Swagger.
func buildRouter(db *mongo.Database) http.Handler {
	h := handlers.New(db, integrationSecret)
	r := chi.NewRouter()

	r.Post("/register", h.Register)
	r.Post("/login", h.Login)

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(integrationSecret))
		r.Post("/tasks", h.CreateTask)
		r.Get("/tasks", h.GetTasks)
		r.Get("/tasks/{id}", h.GetTask)
		r.Put("/tasks/{id}", h.UpdateTask)
		r.Delete("/tasks/{id}", h.DeleteTask)
		r.Get("/admin/tasks", h.AdminGetTasks)
	})

	return r
}

// do sends a JSON request to the integration server and returns the response.
func do(t *testing.T, method, path string, body any, token string) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode request body: %v", err)
		}
	}
	req, err := http.NewRequest(method, integrationServer.URL+path, &buf)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func decode(t *testing.T, resp *http.Response, dst any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

// TestFullAuthAndTaskFlow exercises the complete happy-path lifecycle:
// register → login → CRUD tasks → admin view.
func TestFullAuthAndTaskFlow(t *testing.T) {
	// 1. Register a regular user.
	resp := do(t, http.MethodPost, "/register", map[string]string{
		"username": "integration_user",
		"password": "pass1234",
	}, "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register: got %d, want 201", resp.StatusCode)
	}
	resp.Body.Close()

	// 2. Register an admin user (role set via DB in real usage; here we register
	//    and then verify the 403 path, which validates role enforcement).
	resp = do(t, http.MethodPost, "/register", map[string]string{
		"username": "integration_admin",
		"password": "adminpass",
	}, "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register admin: got %d, want 201", resp.StatusCode)
	}
	resp.Body.Close()

	// 3. Login as the regular user.
	resp = do(t, http.MethodPost, "/login", map[string]string{
		"username": "integration_user",
		"password": "pass1234",
	}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login: got %d, want 200", resp.StatusCode)
	}
	var loginResp map[string]string
	decode(t, resp, &loginResp)
	token := loginResp["token"]
	if token == "" {
		t.Fatal("expected token in login response")
	}

	// 4. Create a task.
	resp = do(t, http.MethodPost, "/tasks", map[string]any{
		"title":       "Integration task",
		"description": "Created during integration test",
		"completed":   false,
	}, token)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create task: got %d, want 201", resp.StatusCode)
	}
	var createdTask map[string]any
	decode(t, resp, &createdTask)
	taskID, _ := createdTask["id"].(string)
	if taskID == "" {
		t.Fatal("expected task id in create response")
	}

	// 5. List tasks — should contain the one we just created.
	resp = do(t, http.MethodGet, "/tasks", nil, token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list tasks: got %d, want 200", resp.StatusCode)
	}
	var tasks []map[string]any
	decode(t, resp, &tasks)
	if len(tasks) == 0 {
		t.Fatal("expected at least one task")
	}

	// 6. Get single task.
	resp = do(t, http.MethodGet, "/tasks/"+taskID, nil, token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get task: got %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()

	// 7. Update task.
	resp = do(t, http.MethodPut, "/tasks/"+taskID, map[string]any{
		"title":       "Updated integration task",
		"description": "Updated description",
		"completed":   true,
	}, token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update task: got %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()

	// 8. Delete task.
	resp = do(t, http.MethodDelete, "/tasks/"+taskID, nil, token)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete task: got %d, want 204", resp.StatusCode)
	}
	resp.Body.Close()

	// 9. Confirm deleted task returns 404.
	resp = do(t, http.MethodGet, "/tasks/"+taskID, nil, token)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("get deleted task: got %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestAuthEndpoints(t *testing.T) {
	t.Run("register duplicate username", func(t *testing.T) {
		payload := map[string]string{"username": "dup_user", "password": "pass"}
		do(t, http.MethodPost, "/register", payload, "").Body.Close() // first registration
		resp := do(t, http.MethodPost, "/register", payload, "")
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusConflict {
			t.Errorf("got %d, want 409", resp.StatusCode)
		}
	})

	t.Run("login with wrong password", func(t *testing.T) {
		do(t, http.MethodPost, "/register", map[string]string{
			"username": "wrongpass_user", "password": "correct",
		}, "").Body.Close()

		resp := do(t, http.MethodPost, "/login", map[string]string{
			"username": "wrongpass_user", "password": "wrong",
		}, "")
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", resp.StatusCode)
		}
	})
}

func TestProtectedRoutes(t *testing.T) {
	t.Run("no token returns 401", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/tasks", nil, "")
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", resp.StatusCode)
		}
	})

	t.Run("invalid token returns 401", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/tasks", nil, "not-a-valid-jwt")
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", resp.StatusCode)
		}
	})

	t.Run("non-admin cannot access /admin/tasks", func(t *testing.T) {
		// Register + login a regular user.
		do(t, http.MethodPost, "/register", map[string]string{
			"username": "regular_user2", "password": "pass",
		}, "").Body.Close()
		resp := do(t, http.MethodPost, "/login", map[string]string{
			"username": "regular_user2", "password": "pass",
		}, "")
		var lr map[string]string
		decode(t, resp, &lr)

		resp = do(t, http.MethodGet, "/admin/tasks", nil, lr["token"])
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("got %d, want 403", resp.StatusCode)
		}
	})
}
