package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Ip struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	UserId    primitive.ObjectID `bson:"userId"`
	Details   string             `bson:"details"`
	CreatedAt time.Time          `bson:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt"`
}

func InitializeIpCollection(db *mongo.Database) *mongo.Collection {
	collection := db.Collection("ip_details")
	return collection
}
