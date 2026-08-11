// @title			Task Management API
// @version		1.0
// @description	REST API for managing tasks with JWT-based authentication and role-based authorization.
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

	"github.com/kenee101/go-test/internal/app"
	"github.com/kenee101/go-test/internal/config"
	"github.com/kenee101/go-test/internal/db"
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

	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      app.NewRouter(cfg, database),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Printf("listening on %s", cfg.Addr)
	log.Fatal(srv.ListenAndServe())
}
