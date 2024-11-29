package models

import (
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

// Toggle Block reprsents whether block can
type ToggleBlock struct {
	Block     bool      `bson:"block"`
	CreatedAt time.Time `bson:"createdAt,omitempty" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt,omitempty" json:"updatedAt"`
}

func InitializeBlockToggler(db *mongo.Database) *mongo.Collection {
	return db.Collection("block_toggler")
}
