package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// MinimumRecharge represents the schema for the minimum recharge document.
type MinimumRecharge struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"`
	MinimumRecharge float64            `bson:"minimumRecharge"`
	CreatedAt       time.Time          `bson:"createdAt"`
	UpdatedAt       time.Time          `bson:"updatedAt"`
}

func InitializeMinimumCollection(db *mongo.Database) *mongo.Collection {
	collection := db.Collection("minimum_recharge")
	return collection
}
