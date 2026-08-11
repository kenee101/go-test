package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	h := newHandler()
	_ = testDB.Collection("users").Drop(context.Background())

	t.Run("success", func(t *testing.T) {
		body := `{"username":"alice","email":"alice@example.com","password":"password123"}`
		req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		h.Register(rr, req)

		require.Equal(t, http.StatusCreated, rr.Code, "body: %s", rr.Body.String())
		var resp map[string]string
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		assert.NotEmpty(t, resp["id"])
	})

	t.Run("duplicate username", func(t *testing.T) {
		body := `{"username":"alice","email":"alice2@example.com","password":"password123"}`
		req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body))
		rr := httptest.NewRecorder()

		h.Register(rr, req)

		assert.Equal(t, http.StatusConflict, rr.Code)
	})

	t.Run("duplicate email", func(t *testing.T) {
		body := `{"username":"alice2","email":"alice@example.com","password":"password123"}`
		req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body))
		rr := httptest.NewRecorder()

		h.Register(rr, req)

		assert.Equal(t, http.StatusConflict, rr.Code)
	})

	t.Run("missing username", func(t *testing.T) {
		body := `{"username":"","email":"noname@example.com","password":"password123"}`
		req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body))
		rr := httptest.NewRecorder()

		h.Register(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("missing email", func(t *testing.T) {
		body := `{"username":"bob","email":"","password":"password123"}`
		req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body))
		rr := httptest.NewRecorder()

		h.Register(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("missing password", func(t *testing.T) {
		body := `{"username":"bob","email":"bob@example.com","password":""}`
		req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body))
		rr := httptest.NewRecorder()

		h.Register(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader("not-json"))
		rr := httptest.NewRecorder()

		h.Register(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestLogin(t *testing.T) {
	h := newHandler()
	_ = testDB.Collection("users").Drop(context.Background())

	// Seed a user via Register.
	seed := `{"username":"charlie","email":"charlie@example.com","password":"secret123"}`
	h.Register(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(seed)))

	t.Run("success with username", func(t *testing.T) {
		body := `{"username":"charlie","password":"secret123"}`
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
		rr := httptest.NewRecorder()

		h.Login(rr, req)

		require.Equal(t, http.StatusOK, rr.Code, "body: %s", rr.Body.String())
		var resp map[string]string
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		assert.NotEmpty(t, resp["token"])
	})

	t.Run("success with email", func(t *testing.T) {
		body := `{"email":"charlie@example.com","password":"secret123"}`
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
		rr := httptest.NewRecorder()

		h.Login(rr, req)

		require.Equal(t, http.StatusOK, rr.Code, "body: %s", rr.Body.String())
		var resp map[string]string
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		assert.NotEmpty(t, resp["token"])
	})

	t.Run("wrong password", func(t *testing.T) {
		body := `{"username":"charlie","password":"wrongpassword"}`
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
		rr := httptest.NewRecorder()

		h.Login(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("unknown user", func(t *testing.T) {
		body := `{"username":"nobody","password":"password"}`
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
		rr := httptest.NewRecorder()

		h.Login(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("missing identifier and password", func(t *testing.T) {
		body := `{"password":"secret123"}`
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
		rr := httptest.NewRecorder()

		h.Login(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("bad"))
		rr := httptest.NewRecorder()

		h.Login(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
