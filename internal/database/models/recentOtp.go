package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type RecentOTP struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	TransactionID string             `bson:"transaction_id" json:"transaction_id"`
	OTP           string             `bson:"otp" json:"otp" validate:"required"`
	CreatedAt     time.Time          `bson:"createdAt,omitempty" json:"createdAt"`
	UpdatedAt     time.Time          `bson:"updatedAt,omitempty" json:"updatedAt"`
}

func InitializeVerifyRecentOTPCollection(db *mongo.Database) *mongo.Collection {
	collection := db.Collection("recentOtp")
	return collection
}
