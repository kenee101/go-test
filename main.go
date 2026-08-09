// @title			Task Management API
// @version		1.0
// @description	REST API for managing tasks with JWT-based authentication and role-based authorization.
// @host			localhost:8080
// @BasePath		/
//
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				Type "Bearer" followed by a space and your JWT token.
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "github.com/kenee101/go-test/docs"
	"github.com/kenee101/go-test/internal/config"
	"github.com/kenee101/go-test/internal/db"
	"github.com/kenee101/go-test/internal/handlers"
	"github.com/kenee101/go-test/internal/middleware"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	database, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("db: %v", err)
	}

	h := handlers.New(database, cfg.JWTSecret)

	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
	}))
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	// Public routes
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)

	// Authenticated task routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		r.Post("/tasks", h.CreateTask)
		r.Get("/tasks", h.GetTasks)
		r.Get("/tasks/{id}", h.GetTask)
		r.Put("/tasks/{id}", h.UpdateTask)
		r.Delete("/tasks/{id}", h.DeleteTask)
		r.Get("/admin/tasks", h.AdminGetTasks)
	})

	// Swagger UI at /swagger/index.html
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Printf("listening on %s", cfg.Addr)
	log.Fatal(srv.ListenAndServe())
}
