package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// ServerDiscount represents the schema for a server discount document
type ServerDiscount struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Server    int                `bson:"server" json:"server" validate:"required"`
	Discount  float64            `bson:"discount" json:"discount"`
	CreatedAt time.Time          `bson:"createdAt,omitempty" json:"createdAt,omitempty"`
	UpdatedAt time.Time          `bson:"updatedAt,omitempty" json:"updatedAt,omitempty"`
}

// InitializeServerDiscountCollection initializes the 'server-discount' collection and sets up indexes
func InitializeServerDiscountCollection(db *mongo.Database) *mongo.Collection {
	collection := db.Collection("server-discounts")
	return collection
}
