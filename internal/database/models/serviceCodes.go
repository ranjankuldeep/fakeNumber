package models

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// ServiceCode represents the structure of the service code document
type ServiceCode struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Code      string             `bson:"code" json:"code" validate:"required"`
	Name      string             `bson:"name" json:"name" validate:"required"`
	CreatedAt time.Time          `bson:"createdAt,omitempty" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt,omitempty" json:"updatedAt"`
}

// InitializeServiceCodeCollection initializes the collection for "serviceCodes"
func InitializeServiceCodeCollection(db *mongo.Database) *mongo.Collection {
	collection := db.Collection("serviceCodes")

	// Ensure the collection has an index on the code field
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	indexModel := mongo.IndexModel{
		Keys:    map[string]interface{}{"code": 1},
		Options: nil, // Add options like unique:true if needed
	}

	_, err := collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		// Handle index creation error
		panic(err)
	}

	return collection
}
