package models

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ApiWalletUser represents the schema in Go
type ApiWalletUser struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	UserID        primitive.ObjectID `bson:"userId,omitempty"`
	APIKey        string             `bson:"api_key"`
	Balance       float64            `bson:"balance"`
	TRXAddress    string             `bson:"trxAddress,omitempty"`
	TRXPrivateKey string             `bson:"trxPrivateKey,omitempty"`
	CreatedAt     time.Time          `bson:"createdAt,omitempty"`
	UpdatedAt     time.Time          `bson:"updatedAt,omitempty"`
}

func EnsureIndexesApi(ctx context.Context, db *mongo.Database, collectionName string) error {
	// Define validation schema
	validator := bson.M{
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": []string{"balance"},
			"properties": bson.M{
				"balance": bson.M{
					"bsonType":    "double",
					"minimum":     0.01,
					"description": "Balance must be a positive number greater than 0",
				},
			},
		},
	}

	// Check if the collection already exists
	collections, err := db.ListCollectionNames(ctx, bson.M{"name": collectionName})
	if err != nil {
		return err
	}
	if len(collections) == 0 {
		// Create the collection with validation if it doesn't exist
		err = db.CreateCollection(ctx, collectionName, options.CreateCollection().SetValidator(validator))
		if err != nil {
			return err
		}
	}

	// Define and create unique index for userId
	collection := db.Collection(collectionName)
	indexModel := mongo.IndexModel{
		Keys:    bson.M{"userId": 1},
		Options: options.Index().SetUnique(true),
	}
	_, err = collection.Indexes().CreateOne(ctx, indexModel)
	return err
}

func InitializeApiWalletuserCollection(db *mongo.Database) *mongo.Collection {
	const collectionName = "apikey_and_balances"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Ensure validation and indexes
	err := EnsureIndexesApi(ctx, db, collectionName)
	if err != nil {
		panic("Failed to ensure indexes and validation rules: " + err.Error())
	}

	// Return the collection reference
	return db.Collection(collectionName)
}
