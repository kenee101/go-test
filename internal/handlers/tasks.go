package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/kenee101/go-test/internal/middleware"
	"github.com/kenee101/go-test/internal/models"
)

type taskReq struct {
	Title       string `json:"title"       example:"Learn Go"`
	Description string `json:"description" example:"Complete the MongoDB section"`
	Completed   bool   `json:"completed"   example:"false"`
}

func (h *Handler) userFromContext(r *http.Request) (bson.ObjectID, string, bool) {
	uid, _ := r.Context().Value(middleware.CtxUserID).(string)
	role, _ := r.Context().Value(middleware.CtxRole).(string)
	oid, err := bson.ObjectIDFromHex(uid)
	if err != nil {
		return bson.NilObjectID, "", false
	}
	return oid, role, true
}

// CreateTask godoc
//
//	@Summary		Create a task
//	@Description	Creates a new task for the authenticated user.
//	@Tags			tasks
//	@Accept			json
//	@Produce		json
//	@Param			body	body		taskReq			true	"Task details"
//	@Success		201		{object}	models.Task		"created task"
//	@Failure		400		{string}	string			"invalid request"
//	@Failure		401		{string}	string			"unauthorized"
//	@Failure		500		{string}	string			"server error"
//	@Security		BearerAuth
//	@Router			/tasks [post]
func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req taskReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	userID, _, ok := h.userFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	now := time.Now().UTC()
	t := models.Task{
		ID:          bson.NewObjectID(),
		Title:       req.Title,
		Description: req.Description,
		Completed:   req.Completed,
		UserID:      userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if _, err := h.db.Collection("tasks").InsertOne(ctx, t); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}

// GetTasks godoc
//
//	@Summary		List tasks
//	@Description	Returns all tasks belonging to the authenticated user.
//	@Tags			tasks
//	@Produce		json
//	@Success		200	{array}		models.Task	"user's tasks"
//	@Failure		401	{string}	string		"unauthorized"
//	@Failure		500	{string}	string		"server error"
//	@Security		BearerAuth
//	@Router			/tasks [get]
func (h *Handler) GetTasks(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := h.userFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	cur, err := h.db.Collection("tasks").Find(ctx, bson.M{"userId": userID})
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// GetTask godoc
//
//	@Summary		Get a task
//	@Description	Returns a single task by ID. Admins can access any task; users can only access their own.
//	@Tags			tasks
//	@Produce		json
//	@Param			id	path		string		true	"Task ID"
//	@Success		200	{object}	models.Task	"task"
//	@Failure		400	{string}	string		"invalid id"
//	@Failure		401	{string}	string		"unauthorized"
//	@Failure		403	{string}	string		"forbidden"
//	@Failure		404	{string}	string		"not found"
//	@Security		BearerAuth
//	@Router			/tasks/{id} [get]
func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	oid, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var t models.Task
	if err := h.db.Collection("tasks").FindOne(ctx, bson.M{"_id": oid}).Decode(&t); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	userID, role, ok := h.userFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if t.UserID != userID && role != "admin" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

// UpdateTask godoc
//
//	@Summary		Update a task
//	@Description	Updates title, description, and completed status. Only the owner or an admin may update.
//	@Tags			tasks
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Task ID"
//	@Param			body	body		taskReq				true	"Updated task fields"
//	@Success		200		{object}	map[string]string	"updated task id"
//	@Failure		400		{string}	string				"invalid id or request"
//	@Failure		401		{string}	string				"unauthorized"
//	@Failure		403		{string}	string				"forbidden"
//	@Failure		404		{string}	string				"not found"
//	@Failure		500		{string}	string				"server error"
//	@Security		BearerAuth
//	@Router			/tasks/{id} [put]
func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	oid, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req taskReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	col := h.db.Collection("tasks")
	var existing models.Task
	if err := col.FindOne(ctx, bson.M{"_id": oid}).Decode(&existing); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	userID, role, ok := h.userFromContext(r)
	if !ok {
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": oid.Hex()})
}

// DeleteTask godoc
//
//	@Summary		Delete a task
//	@Description	Deletes a task by ID. Only the owner or an admin may delete.
//	@Tags			tasks
//	@Param			id	path		string	true	"Task ID"
//	@Success		204	{string}	string	"no content"
//	@Failure		400	{string}	string	"invalid id"
//	@Failure		401	{string}	string	"unauthorized"
//	@Failure		403	{string}	string	"forbidden"
//	@Failure		404	{string}	string	"not found"
//	@Failure		500	{string}	string	"server error"
//	@Security		BearerAuth
//	@Router			/tasks/{id} [delete]
func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	oid, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	col := h.db.Collection("tasks")
	var existing models.Task
	if err := col.FindOne(ctx, bson.M{"_id": oid}).Decode(&existing); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	userID, role, ok := h.userFromContext(r)
	if !ok {
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

// AdminGetTasks godoc
//
//	@Summary		List all tasks (admin)
//	@Description	Returns every task in the system. Requires admin role.
//	@Tags			admin
//	@Produce		json
//	@Success		200	{array}		models.Task	"all tasks"
//	@Failure		401	{string}	string		"unauthorized"
//	@Failure		403	{string}	string		"forbidden"
//	@Failure		500	{string}	string		"server error"
//	@Security		BearerAuth
//	@Router			/admin/tasks [get]
func (h *Handler) AdminGetTasks(w http.ResponseWriter, r *http.Request) {
	_, role, ok := h.userFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if role != "admin" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	cur, err := h.db.Collection("tasks").Find(ctx, bson.M{})
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}
