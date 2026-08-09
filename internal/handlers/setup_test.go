package handlers_test

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/kenee101/go-test/internal/handlers"
	"github.com/kenee101/go-test/internal/middleware"
)

const testSecret = "test-secret"
const testDBName = "taskdb_test"

var testDB *mongo.Database

// TestMain connects to MongoDB once, runs all handler tests, then drops the test DB.
func TestMain(m *testing.M) {
	_ = godotenv.Load("../../.env")

	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		// No MongoDB available — skip all handler tests.
		os.Exit(0)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx, nil); err != nil {
		os.Exit(0)
	}

	testDB = client.Database(testDBName)

	code := m.Run()

	_ = testDB.Drop(context.Background())
	_ = client.Disconnect(context.Background())

	os.Exit(code)
}

// newHandler returns a Handler wired to the test database.
func newHandler() *handlers.Handler {
	return handlers.New(testDB, testSecret)
}

// makeToken creates a signed JWT for use in tests.
func makeToken(t *testing.T, userID, role string, ttl time.Duration) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":  userID,
		"role": role,
		"exp":  time.Now().Add(ttl).Unix(),
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("makeToken: %v", err)
	}
	return tok
}

// withUserCtx injects user_id and role into the request context (simulates AuthMiddleware).
func withUserCtx(r *http.Request, userID, role string) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.CtxUserID, userID)
	ctx = context.WithValue(ctx, middleware.CtxRole, role)
	return r.WithContext(ctx)
}

// withChiParam injects a chi URL parameter into the request context.
func withChiParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}
