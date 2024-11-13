package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// User represents the structure of a user document
type User struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	Email         string             `bson:"email" json:"email" validate:"required,email"`
	Password      string             `bson:"password,omitempty" json:"password"`
	GoogleID      string             `bson:"googleId,omitempty" json:"googleId"`
	DisplayName   string             `bson:"displayName,omitempty" json:"displayName"`
	ProfileImg    string             `bson:"profileImg,omitempty" json:"profileImg"`
	Blocked       bool               `bson:"blocked" json:"blocked" default:"false"`
	BlockedReason *string            `bson:"blocked_reason,omitempty" json:"blocked_reason" default:"null"`
	CreatedAt     time.Time          `bson:"createdAt,omitempty" json:"createdAt"`
	UpdatedAt     time.Time          `bson:"updatedAt,omitempty" json:"updatedAt"`
}

// InitializeUserCollection initializes the collection for "users"
func InitializeUserCollection(db *mongo.Database) *mongo.Collection {
	return db.Collection("users")
}
