package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// UserDiscount represents the structure of the user discount document
type UserDiscount struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	UserID    primitive.ObjectID `bson:"userId" json:"userId" validate:"required"`
	Service   string             `bson:"service" json:"service" validate:"required"`
	Server    int                `bson:"server" json:"server" validate:"required"`
	Discount  float64            `bson:"discount,omitempty" json:"discount" default:"0"`
	CreatedAt time.Time          `bson:"createdAt,omitempty" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt,omitempty" json:"updatedAt"`
}

// InitializeUserDiscountCollection initializes the collection for "user-discount"
func InitializeUserDiscountCollection(db *mongo.Database) *mongo.Collection {
	return db.Collection("userDiscount")
}
