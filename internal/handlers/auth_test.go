package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegister(t *testing.T) {
	h := newHandler()
	// Start each test group with a clean users collection.
	_ = testDB.Collection("users").Drop(context.Background())

	t.Run("success", func(t *testing.T) {
		body := `{"username":"alice","password":"password123"}`
		req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		h.Register(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("got %d, want 201 — body: %s", rr.Code, rr.Body.String())
		}
		var resp map[string]string
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if resp["id"] == "" {
			t.Error("expected non-empty id in response")
		}
	})

	t.Run("duplicate username", func(t *testing.T) {
		body := `{"username":"alice","password":"password123"}`
		req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body))
		rr := httptest.NewRecorder()

		h.Register(rr, req)

		if rr.Code != http.StatusConflict {
			t.Errorf("got %d, want 409", rr.Code)
		}
	})

	t.Run("missing username", func(t *testing.T) {
		body := `{"username":"","password":"password123"}`
		req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body))
		rr := httptest.NewRecorder()

		h.Register(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("got %d, want 400", rr.Code)
		}
	})

	t.Run("missing password", func(t *testing.T) {
		body := `{"username":"bob","password":""}`
		req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body))
		rr := httptest.NewRecorder()

		h.Register(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("got %d, want 400", rr.Code)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader("not-json"))
		rr := httptest.NewRecorder()

		h.Register(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("got %d, want 400", rr.Code)
		}
	})
}

func TestLogin(t *testing.T) {
	h := newHandler()
	_ = testDB.Collection("users").Drop(context.Background())

	// Seed a user via Register.
	seed := `{"username":"charlie","password":"secret123"}`
	reg := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(seed))
	h.Register(httptest.NewRecorder(), reg)

	t.Run("success", func(t *testing.T) {
		body := `{"username":"charlie","password":"secret123"}`
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
		rr := httptest.NewRecorder()

		h.Login(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("got %d, want 200 — body: %s", rr.Code, rr.Body.String())
		}
		var resp map[string]string
		json.NewDecoder(rr.Body).Decode(&resp)
		if resp["token"] == "" {
			t.Error("expected non-empty token in response")
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		body := `{"username":"charlie","password":"wrongpassword"}`
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
		rr := httptest.NewRecorder()

		h.Login(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", rr.Code)
		}
	})

	t.Run("unknown user", func(t *testing.T) {
		body := `{"username":"nobody","password":"password"}`
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
		rr := httptest.NewRecorder()

		h.Login(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", rr.Code)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("bad"))
		rr := httptest.NewRecorder()

		h.Login(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("got %d, want 400", rr.Code)
		}
	})
}
