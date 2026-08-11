package app

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"go.mongodb.org/mongo-driver/v2/mongo"

	_ "github.com/kenee101/go-test/docs"
	"github.com/kenee101/go-test/internal/config"
	"github.com/kenee101/go-test/internal/handlers"
	"github.com/kenee101/go-test/internal/middleware"
)

// NewRouter builds and returns the application router.
// Both main() and integration tests call this so routing is never duplicated.
func NewRouter(cfg *config.Config, db *mongo.Database) http.Handler {
	h := handlers.New(db, cfg.JWTSecret)

	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
	}))
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	r.Post("/register", h.Register)
	r.Post("/login", h.Login)

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		r.Post("/tasks", h.CreateTask)
		r.Get("/tasks", h.GetTasks)
		r.Get("/tasks/{id}", h.GetTask)
		r.Put("/tasks/{id}", h.UpdateTask)
		r.Delete("/tasks/{id}", h.DeleteTask)
		r.Get("/admin/tasks", h.AdminGetTasks)
	})

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	return r
}
