package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// UnsendTrx represents the structure for the unsend transaction document
type UnsendTrx struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	Email         string             `bson:"email" json:"email" validate:"required,email"`
	TrxAddress    string             `bson:"trxAddress" json:"trxAddress" validate:"required"`
	TrxPrivateKey string             `bson:"trxPrivateKey" json:"trxPrivateKey" validate:"required"`
	CreatedAt     time.Time          `bson:"createdAt,omitempty" json:"createdAt"`
	UpdatedAt     time.Time          `bson:"updatedAt,omitempty" json:"updatedAt"`
}

// InitializeUnsendTrxCollection initializes the collection for "unsend-trx"
func InitializeUnsendTrxCollection(db *mongo.Database) *mongo.Collection {
	return db.Collection("unsend-trx")
}
