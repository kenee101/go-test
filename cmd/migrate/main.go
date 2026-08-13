package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	migrate "github.com/xakep666/mongo-migrate"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/joho/godotenv"
	"github.com/kenee101/go-test/internal/db/migrations"
)

func main() {
	_ = godotenv.Load()

	if len(os.Args) < 2 {
		log.Fatal("usage: migrate <up|down> [n]")
	}
	direction := os.Args[1]

	n := migrate.AllAvailable
	if len(os.Args) >= 3 {
		parsed, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatalf("invalid n: %v", err)
		}
		n = parsed
	}

	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	dbName := os.Getenv("MONGO_DB")
	if dbName == "" {
		dbName = "taskdb"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Disconnect(context.Background())

	m := migrate.NewMigrate(client.Database(dbName), migrations.All...)

	switch direction {
	case "up":
		if err := m.Up(ctx, n); err != nil {
			log.Fatalf("up: %v", err)
		}
		log.Println("migrations applied")
	case "down":
		if err := m.Down(ctx, n); err != nil {
			log.Fatalf("down: %v", err)
		}
		log.Println("migrations rolled back")
	default:
		log.Fatalf("unknown direction %q — use 'up' or 'down'", direction)
	}
}
