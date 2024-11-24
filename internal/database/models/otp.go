package models

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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

	// Ensuring the email field is unique
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = EnsureIndexes(ctx, collection)

	return collection
}

// InitializeForgotOTPCollection initializes the collection for "forgotOtp"
func InitializeForgotOTPCollection(db *mongo.Database) *mongo.Collection {
	collection := db.Collection("forgototps")

	// Ensuring the email field is unique
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = EnsureIndexes(ctx, collection)

	return collection
}
