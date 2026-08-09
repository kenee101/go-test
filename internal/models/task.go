package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Task struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Title       string        `bson:"title"         json:"title"`
	Description string        `bson:"description"   json:"description"`
	Completed   bool          `bson:"completed"     json:"completed"`
	UserID      bson.ObjectID `bson:"userId"        json:"userId"`
	CreatedAt   time.Time     `bson:"createdAt"     json:"createdAt"`
	UpdatedAt   time.Time     `bson:"updatedAt"     json:"updatedAt"`
}
