package migrations

import (
	"context"

	migrate "github.com/xakep666/mongo-migrate"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func init() {
	All = append(All, migrate.Migration{
		Version:     1,
		Description: "add unique indexes on users.username and users.email",
		Up: func(ctx context.Context, db *mongo.Database) error {
			indexes := []mongo.IndexModel{
				{
					Keys:    bson.D{{Key: "username", Value: 1}},
					Options: options.Index().SetUnique(true).SetName("username_1"),
				},
				{
					Keys:    bson.D{{Key: "email", Value: 1}},
					Options: options.Index().SetUnique(true).SetName("email_1"),
				},
			}
			_, err := db.Collection("users").Indexes().CreateMany(ctx, indexes)
			return err
		},
		Down: func(ctx context.Context, db *mongo.Database) error {
			col := db.Collection("users")
			if err := col.Indexes().DropOne(ctx, "username_1"); err != nil {
				return err
			}
			return col.Indexes().DropOne(ctx, "email_1")
		},
	})
}
