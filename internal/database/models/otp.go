package models

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// OTP represents the structure of an OTP document
type OTP struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Email     string             `bson:"email" json:"email" validate:"required,email"`
	OTP       string             `bson:"otp" json:"otp" validate:"required"`
	CreatedAt time.Time          `bson:"createdAt,omitempty" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt,omitempty" json:"updatedAt"`
}

// InitializeVerifyOTPCollection initializes the collection for "verifyOtp"
func InitializeVerifyOTPCollection(db *mongo.Database) *mongo.Collection {
	collection := db.Collection("verifyOtp")
	return collection
}

// InitializeForgotOTPCollection initializes the collection for "forgotOtp"
func InitializeForgotOTPCollection(db *mongo.Database) *mongo.Collection {
	collection := db.Collection("forgototps")
	return collection
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
