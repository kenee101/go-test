package db

import (
	"context"
	"time"

	migrate "github.com/xakep666/mongo-migrate"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/kenee101/go-test/internal/config"
	"github.com/kenee101/go-test/internal/db/migrations"
)

func Connect(cfg *config.Config) (*mongo.Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	db := client.Database(cfg.MongoDB)

	if err := migrate.NewMigrate(db, migrations.All...).Up(ctx, migrate.AllAvailable); err != nil {
		return nil, err
	}

	return db, nil
}
