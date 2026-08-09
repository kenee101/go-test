package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/kenee101/go-test/internal/models"
)

// seedTask inserts a task directly into the test DB and returns it.
func seedTask(t *testing.T, userID bson.ObjectID, title string) models.Task {
	t.Helper()
	now := time.Now().UTC()
	task := models.Task{
		ID:          bson.NewObjectID(),
		Title:       title,
		Description: "test description",
		Completed:   false,
		UserID:      userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_, err := testDB.Collection("tasks").InsertOne(context.Background(), task)
	if err != nil {
		t.Fatalf("seedTask: %v", err)
	}
	return task
}

func TestCreateTask(t *testing.T) {
	h := newHandler()
	_ = testDB.Collection("tasks").Drop(context.Background())

	userID := bson.NewObjectID()

	t.Run("success", func(t *testing.T) {
		body := `{"title":"Write tests","description":"Cover all handlers","completed":false}`
		req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(body))
		req = withUserCtx(req, userID.Hex(), "user")
		rr := httptest.NewRecorder()

		h.CreateTask(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("got %d, want 201 — body: %s", rr.Code, rr.Body.String())
		}
		var task models.Task
		if err := json.NewDecoder(rr.Body).Decode(&task); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if task.Title != "Write tests" {
			t.Errorf("title: got %q, want %q", task.Title, "Write tests")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader("bad"))
		req = withUserCtx(req, userID.Hex(), "user")
		rr := httptest.NewRecorder()

		h.CreateTask(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("got %d, want 400", rr.Code)
		}
	})

	t.Run("no user context", func(t *testing.T) {
		body := `{"title":"Task"}`
		req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(body))
		// deliberately no withUserCtx
		rr := httptest.NewRecorder()

		h.CreateTask(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", rr.Code)
		}
	})
}

func TestGetTasks(t *testing.T) {
	h := newHandler()
	_ = testDB.Collection("tasks").Drop(context.Background())

	userID := bson.NewObjectID()
	otherID := bson.NewObjectID()
	seedTask(t, userID, "My Task 1")
	seedTask(t, userID, "My Task 2")
	seedTask(t, otherID, "Other User Task")

	t.Run("returns only user tasks", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
		req = withUserCtx(req, userID.Hex(), "user")
		rr := httptest.NewRecorder()

		h.GetTasks(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("got %d, want 200", rr.Code)
		}
		var tasks []models.Task
		json.NewDecoder(rr.Body).Decode(&tasks)
		if len(tasks) != 2 {
			t.Errorf("got %d tasks, want 2", len(tasks))
		}
	})
}

func TestGetTask(t *testing.T) {
	h := newHandler()
	_ = testDB.Collection("tasks").Drop(context.Background())

	ownerID := bson.NewObjectID()
	otherID := bson.NewObjectID()
	task := seedTask(t, ownerID, "Owner's Task")

	t.Run("owner can get their task", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/tasks/"+task.ID.Hex(), nil)
		req = withUserCtx(req, ownerID.Hex(), "user")
		req = withChiParam(req, "id", task.ID.Hex())
		rr := httptest.NewRecorder()

		h.GetTask(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("got %d, want 200", rr.Code)
		}
	})

	t.Run("other user gets 403", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/tasks/"+task.ID.Hex(), nil)
		req = withUserCtx(req, otherID.Hex(), "user")
		req = withChiParam(req, "id", task.ID.Hex())
		rr := httptest.NewRecorder()

		h.GetTask(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Errorf("got %d, want 403", rr.Code)
		}
	})

	t.Run("admin can get any task", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/tasks/"+task.ID.Hex(), nil)
		req = withUserCtx(req, otherID.Hex(), "admin")
		req = withChiParam(req, "id", task.ID.Hex())
		rr := httptest.NewRecorder()

		h.GetTask(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("got %d, want 200", rr.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/tasks/bad-id", nil)
		req = withUserCtx(req, ownerID.Hex(), "user")
		req = withChiParam(req, "id", "bad-id")
		rr := httptest.NewRecorder()

		h.GetTask(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("got %d, want 400", rr.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/tasks/"+bson.NewObjectID().Hex(), nil)
		ghostID := bson.NewObjectID().Hex()
		req = withUserCtx(req, ownerID.Hex(), "user")
		req = withChiParam(req, "id", ghostID)
		rr := httptest.NewRecorder()

		h.GetTask(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("got %d, want 404", rr.Code)
		}
	})
}

func TestUpdateTask(t *testing.T) {
	h := newHandler()
	_ = testDB.Collection("tasks").Drop(context.Background())

	ownerID := bson.NewObjectID()
	otherID := bson.NewObjectID()
	task := seedTask(t, ownerID, "Original Title")

	t.Run("owner can update", func(t *testing.T) {
		body := `{"title":"Updated Title","description":"new desc","completed":true}`
		req := httptest.NewRequest(http.MethodPut, "/tasks/"+task.ID.Hex(), strings.NewReader(body))
		req = withUserCtx(req, ownerID.Hex(), "user")
		req = withChiParam(req, "id", task.ID.Hex())
		rr := httptest.NewRecorder()

		h.UpdateTask(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("got %d, want 200 — body: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("other user gets 403", func(t *testing.T) {
		body := `{"title":"Hacked"}`
		req := httptest.NewRequest(http.MethodPut, "/tasks/"+task.ID.Hex(), strings.NewReader(body))
		req = withUserCtx(req, otherID.Hex(), "user")
		req = withChiParam(req, "id", task.ID.Hex())
		rr := httptest.NewRecorder()

		h.UpdateTask(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Errorf("got %d, want 403", rr.Code)
		}
	})
}

func TestDeleteTask(t *testing.T) {
	h := newHandler()
	_ = testDB.Collection("tasks").Drop(context.Background())

	ownerID := bson.NewObjectID()
	otherID := bson.NewObjectID()
	task := seedTask(t, ownerID, "Task to delete")

	t.Run("other user gets 403", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/tasks/"+task.ID.Hex(), nil)
		req = withUserCtx(req, otherID.Hex(), "user")
		req = withChiParam(req, "id", task.ID.Hex())
		rr := httptest.NewRecorder()

		h.DeleteTask(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Errorf("got %d, want 403", rr.Code)
		}
	})

	t.Run("owner can delete", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/tasks/"+task.ID.Hex(), nil)
		req = withUserCtx(req, ownerID.Hex(), "user")
		req = withChiParam(req, "id", task.ID.Hex())
		rr := httptest.NewRecorder()

		h.DeleteTask(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Fatalf("got %d, want 204", rr.Code)
		}
	})

	t.Run("already deleted returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/tasks/"+task.ID.Hex(), nil)
		req = withUserCtx(req, ownerID.Hex(), "user")
		req = withChiParam(req, "id", task.ID.Hex())
		rr := httptest.NewRecorder()

		h.DeleteTask(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("got %d, want 404", rr.Code)
		}
	})
}

func TestAdminGetTasks(t *testing.T) {
	h := newHandler()
	_ = testDB.Collection("tasks").Drop(context.Background())

	user1 := bson.NewObjectID()
	user2 := bson.NewObjectID()
	seedTask(t, user1, "Task A")
	seedTask(t, user2, "Task B")

	t.Run("admin sees all tasks", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/tasks", nil)
		req = withUserCtx(req, user1.Hex(), "admin")
		rr := httptest.NewRecorder()

		h.AdminGetTasks(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("got %d, want 200", rr.Code)
		}
		var tasks []models.Task
		json.NewDecoder(rr.Body).Decode(&tasks)
		if len(tasks) < 2 {
			t.Errorf("got %d tasks, want at least 2", len(tasks))
		}
	})

	t.Run("non-admin gets 403", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/tasks", nil)
		req = withUserCtx(req, user1.Hex(), "user")
		rr := httptest.NewRecorder()

		h.AdminGetTasks(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Errorf("got %d, want 403", rr.Code)
		}
	})
}
