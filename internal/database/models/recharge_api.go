package models

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// RechargeAPI represents the structure of a recharge API document
type RechargeAPI struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	RechargeType string             `bson:"recharge_type" json:"recharge_type" validate:"required"`
	APIKey       string             `bson:"api_key,omitempty" json:"api_key"`
	Maintenance  bool               `bson:"maintenance" json:"maintenance" default:"false"`
	CreatedAt    time.Time          `bson:"createdAt,omitempty" json:"createdAt"`
	UpdatedAt    time.Time          `bson:"updatedAt,omitempty" json:"updatedAt"`
}

// InitializeRechargeAPICollection initializes the collection for "recharge-api"
func InitializeRechargeAPICollection(db *mongo.Database) *mongo.Collection {
	collection := db.Collection("recharge-apis")

	// Optionally, ensure indexes
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = EnsureIndexes(ctx, collection)

	return collection
}
