package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/crypto/bcrypt"

	"github.com/kenee101/go-test/internal/models"
)

type registerReq struct {
	Username string `json:"username" example:"johndoe"`
	Password string `json:"password" example:"s3cr3t"`
}

type loginReq struct {
	Username string `json:"username" example:"johndoe"`
	Password string `json:"password" example:"s3cr3t"`
}

// Register godoc
//
//	@Summary		Register a new user
//	@Description	Creates a new user account with the "user" role.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		registerReq			true	"Registration credentials"
//	@Success		201		{object}	map[string]string	"created user id"
//	@Failure		400		{string}	string				"invalid request or username exists"
//	@Failure		409		{string}	string				"username already exists"
//	@Failure		500		{string}	string				"server error"
//	@Router			/register [post]
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Username == "" || req.Password == "" {
		http.Error(w, "username and password are required", http.StatusBadRequest)
		return
	}

	col := h.db.Collection("users")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var existing models.User
	if err := col.FindOne(ctx, bson.M{"username": req.Username}).Decode(&existing); err == nil {
		http.Error(w, "username already exists", http.StatusConflict)
		return
	} else if err != mongo.ErrNoDocuments {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	now := time.Now().UTC()
	user := models.User{
		ID:           bson.NewObjectID(),
		Username:     req.Username,
		PasswordHash: string(hashed),
		Role:         "user",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if _, err := col.InsertOne(ctx, user); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": user.ID.Hex()})
}

// Login godoc
//
//	@Summary		Authenticate a user
//	@Description	Validates credentials and returns a signed JWT bearer token.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		loginReq			true	"Login credentials"
//	@Success		200		{object}	map[string]string	"JWT token"
//	@Failure		400		{string}	string				"invalid request"
//	@Failure		401		{string}	string				"invalid credentials"
//	@Failure		500		{string}	string				"server error"
//	@Router			/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	col := h.db.Collection("users")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var user models.User
	if err := col.FindOne(ctx, bson.M{"username": req.Username}).Decode(&user); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	claims := jwt.MapClaims{
		"sub":  user.ID.Hex(),
		"role": user.Role,
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString([]byte(h.secret))
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": ss})
}
