package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/kenee101/go-test/internal/db"
	"github.com/kenee101/go-test/internal/handlers"
	"github.com/kenee101/go-test/internal/middleware"
)

func main() {
	mongoDB, err := db.Connect()
	if err != nil {
		log.Fatal(err)
	}
	handlers.InitDB(mongoDB)

	r := mux.NewRouter()

	r.HandleFunc("/register", handlers.Register).Methods("POST")
	r.HandleFunc("/login", handlers.Login).Methods("POST")

	api := r.PathPrefix("/").Subrouter()
	api.Use(middleware.AuthMiddleware)

	api.HandleFunc("/tasks", handlers.CreateTask).Methods("POST")
	api.HandleFunc("/tasks", handlers.GetTasks).Methods("GET")
	api.HandleFunc("/tasks/{id}", handlers.GetTask).Methods("GET")
	api.HandleFunc("/tasks/{id}", handlers.UpdateTask).Methods("PUT")
	api.HandleFunc("/tasks/{id}", handlers.DeleteTask).Methods("DELETE")
	api.HandleFunc("/admin/tasks", handlers.AdminGetTasks).Methods("GET")

	// Serve Swagger UI and OpenAPI spec from ./docs
	r.PathPrefix("/docs").Handler(http.StripPrefix("/docs", http.FileServer(http.Dir("docs"))))

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	srv := &http.Server{
		Handler:      r,
		Addr:         addr,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Printf("listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}
