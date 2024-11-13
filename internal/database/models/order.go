package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Order represents the schema structure for an order document
type Order struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	UserID         primitive.ObjectID `bson:"userId" json:"userId"`
	Service        string             `bson:"service" json:"service" validate:"required"`
	Price          float64            `bson:"price" json:"price" validate:"required"`
	Server         int                `bson:"server" json:"server" validate:"required"`
	NumberID       string             `bson:"numberId" json:"numberId" validate:"required"`
	Number         string             `bson:"number" json:"number" validate:"required"`
	OrderTime      time.Time          `bson:"orderTime" json:"orderTime"`
	ExpirationTime time.Time          `bson:"expirationTime" json:"expirationTime" validate:"required"`
	Status         string             `bson:"status" json:"status" validate:"required,oneof=ACTIVE EXPIRED"`
}

// NewOrderCollection initializes and returns the orders collection with indexes if needed
func InitializeOrderCollection(db *mongo.Database) *mongo.Collection {
	collection := db.Collection("orders")

	// Optionally, create indexes here if needed
	// EnsureIndexes can be implemented similarly to previous examples

	return collection
}
