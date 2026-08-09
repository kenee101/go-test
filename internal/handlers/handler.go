package handlers

import (
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Handler holds shared dependencies for all HTTP handlers.
type Handler struct {
	db     *mongo.Database
	secret string
}

// New constructs a Handler with the given MongoDB database and JWT secret.
func New(db *mongo.Database, secret string) *Handler {
	return &Handler{db: db, secret: secret}
}
