package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Server represents the nested structure for individual servers within ServerList
type ServerData struct {
	ServerNumber int    `bson:"serverNumber" json:"serverNumber"`
	Price        string `bson:"price" json:"price"`
	Code         string `bson:"code" json:"code"`
	ServiceName  string `bson:"serviceName" json:"serviceName"`
	Block        bool   `bson:"block" json:"block" default:"false"`
	OtpText      bool   `bson:"otpText" json:"otpText"`
}

// ServerList represents the main structure for the server list document
type ServerList struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Name        string             `bson:"name" json:"name"`
	LowestPrice string             `bson:"lowestPrice" json:"lowestPrice"`   // New field
	ServiceCode string             `bson:"service_code" json:"service_code"` // New field
	Servers     []Server           `bson:"servers" json:"servers"`
	CreatedAt   time.Time          `bson:"createdAt,omitempty" json:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt,omitempty" json:"updatedAt"`
}

// InitializeServerListCollection initializes the collection for "ServerList"
func InitializeServerListCollection(db *mongo.Database) *mongo.Collection {
	return db.Collection("ServerList")
}
