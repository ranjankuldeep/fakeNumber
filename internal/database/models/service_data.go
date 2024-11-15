package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Server represents the nested structure for individual servers within ServerList
type ServerData struct {
	Server int    `bson:"server" json:"server"`
	Price  string `bson:"price" json:"price"`
	Code   string `bson:"code" json:"code"`
	Otp    string `bson:"otp" json:"otp"`
}

// ServerList represents the main structure for the server list document
type ServerList struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Name      string             `bson:"name" json:"name"`
	Servers   []ServerData       `bson:"servers" json:"servers"`
	CreatedAt time.Time          `bson:"createdAt,omitempty" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt,omitempty" json:"updatedAt"`
}

// InitializeServerListCollection initializes the collection for "ServerList"
func InitializeServerListCollection(db *mongo.Database) *mongo.Collection {
	return db.Collection("serverlists")
}
