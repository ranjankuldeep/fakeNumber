package models

import (
	"context"
	"log"
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

// InitializeRechargeAPICollection initializes the collection for "recharge-apis"
func InitializeRechargeAPICollection(db *mongo.Database) *mongo.Collection {
	collection := db.Collection("recharge-apis")

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Drop the unwanted index `userId_1` if it exists
	_, err := collection.Indexes().DropOne(ctx, "userId_1")
	if err == nil {
		log.Println("INFO: Dropped index `userId_1` successfully")
	} else if err != mongo.ErrNilDocument {
		log.Println("ERROR: Failed to drop index `userId_1`:", err)
	}

	// // Ensure the required indexes are created
	// err = EnsureIndexes(ctx, collection)
	// if err != nil {
	// 	log.Println("ERROR: Failed to ensure indexes for recharge-apis collection:", err)
	// }

	return collection
}
