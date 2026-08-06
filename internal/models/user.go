package models

import (
    "time"

    "go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
    ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    Username     string             `bson:"username" json:"username"`
    PasswordHash string             `bson:"passwordHash" json:"-"`
    Role         string             `bson:"role" json:"role"`
    CreatedAt    time.Time          `bson:"createdAt" json:"createdAt"`
    UpdatedAt    time.Time          `bson:"updatedAt" json:"updatedAt"`
}
