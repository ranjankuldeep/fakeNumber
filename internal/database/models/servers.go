package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Server represents the structure of the server document
type Server struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	ServerNumber int                `bson:"server" json:"server" validate:"required"`
	Maintenance  bool               `bson:"maintainance" json:"maintainance" default:"false"`
	APIKey       string             `bson:"api_key,omitempty" json:"api_key"`
	Block        bool               `bson:"block" json:"block" default:"false"`
	Token        string             `bson:"token,omitempty" json:"token"`
	ExchangeRate float64            `bson:"exchangeRate,omitempty" json:"exchangeRate" default:"0.0"`
	Margin       float64            `bson:"margin,omitempty" json:"margin" default:"0.0"`
	CreatedAt    time.Time          `bson:"createdAt,omitempty" json:"createdAt"`
	UpdatedAt    time.Time          `bson:"updatedAt,omitempty" json:"updatedAt"`
}

// InitializeServerCollection initializes the collection for "servers"
func InitializeServerCollection(db *mongo.Database) *mongo.Collection {
	collection := db.Collection("servers")
	return collection
}
