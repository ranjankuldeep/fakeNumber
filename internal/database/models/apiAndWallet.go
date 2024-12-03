package models

import (
	"context"
	"time"

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

// EnsureIndexes ensures the necessary indexes are created on the collection
func EnsureIndexes(ctx context.Context, collection *mongo.Collection) error {
	indexModel := mongo.IndexModel{
		Keys:    map[string]interface{}{"userId": 1},
		Options: options.Index().SetUnique(true),
	}
	_, err := collection.Indexes().CreateOne(ctx, indexModel)
	return err
}

// NewApiWalletUserCollection initializes the collection with indexes if needed
func InitializeApiWalletuserCollection(db *mongo.Database) *mongo.Collection {
	collection := db.Collection("apikey_and_balances")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = EnsureIndexes(ctx, collection)
	return collection
}
