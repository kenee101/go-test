package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/kenee101/go-test/internal/models"
)

func getUserIDFromContext(r *http.Request) (primitive.ObjectID, string, error) {
	uid, ok := r.Context().Value("user_id").(string)
	if !ok {
		return primitive.NilObjectID, "", mongo.ErrNoDocuments
	}
	role, _ := r.Context().Value("role").(string)
	oid, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		return primitive.NilObjectID, "", err
	}
	return oid, role, nil
}

type taskReq struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

func CreateTask(w http.ResponseWriter, r *http.Request) {
	var req taskReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	userID, _, err := getUserIDFromContext(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	col := db.Collection("tasks")
	now := time.Now().UTC()
	t := models.Task{
		ID:          primitive.NewObjectID(),
		Title:       req.Title,
		Description: req.Description,
		Completed:   req.Completed,
		UserID:      userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := col.InsertOne(ctx, t); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}

func GetTasks(w http.ResponseWriter, r *http.Request) {
	userID, _, err := getUserIDFromContext(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	col := db.Collection("tasks")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cur, err := col.Find(ctx, bson.M{"userId": userID})
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	defer cur.Close(ctx)
	var tasks []models.Task
	if err := cur.All(ctx, &tasks); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(tasks)
}

func GetTask(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	col := db.Collection("tasks")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var t models.Task
	if err := col.FindOne(ctx, bson.M{"_id": oid}).Decode(&t); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	userID, role, err := getUserIDFromContext(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if t.UserID != userID && role != "admin" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	json.NewEncoder(w).Encode(t)
}

func UpdateTask(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req taskReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	col := db.Collection("tasks")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var existing models.Task
	if err := col.FindOne(ctx, bson.M{"_id": oid}).Decode(&existing); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	userID, role, err := getUserIDFromContext(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if existing.UserID != userID && role != "admin" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	update := bson.M{
		"$set": bson.M{
			"title":       req.Title,
			"description": req.Description,
			"completed":   req.Completed,
			"updatedAt":   time.Now().UTC(),
		},
	}
	if _, err := col.UpdateByID(ctx, oid, update); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"id": oid.Hex()})
}

func DeleteTask(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	col := db.Collection("tasks")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var existing models.Task
	if err := col.FindOne(ctx, bson.M{"_id": oid}).Decode(&existing); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	userID, role, err := getUserIDFromContext(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if existing.UserID != userID && role != "admin" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if _, err := col.DeleteOne(ctx, bson.M{"_id": oid}); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func AdminGetTasks(w http.ResponseWriter, r *http.Request) {
	_, role, err := getUserIDFromContext(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if role != "admin" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	col := db.Collection("tasks")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cur, err := col.Find(ctx, bson.M{})
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	defer cur.Close(ctx)
	var tasks []models.Task
	if err := cur.All(ctx, &tasks); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(tasks)
}
