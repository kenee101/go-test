package db

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/kenee101/go-test/internal/config"
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
	if err := ensureIndexes(ctx, db); err != nil {
		return nil, err
	}

	return db, nil
}

func ensureIndexes(ctx context.Context, db *mongo.Database) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    map[string]int{"username": 1},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    map[string]int{"email": 1},
			Options: options.Index().SetUnique(true),
		},
	}
	_, err := db.Collection("users").Indexes().CreateMany(ctx, indexes)
	return err
}
