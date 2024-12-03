package models

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Block represents the structure of a block document
type Block struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	BlockType string             `bson:"block_type" json:"block_type" validate:"required"`
	Status    bool               `bson:"status" json:"status" default:"false"`
	CreatedAt time.Time          `bson:"createdAt,omitempty" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt,omitempty" json:"updatedAt"`
}

// NewBlockCollection initializes and returns the block collection with indexes
func InitializeBlockCollection(db *mongo.Database) *mongo.Collection {
	collection := db.Collection("block-users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = EnsureIndexes(ctx, collection)
	return collection
}
