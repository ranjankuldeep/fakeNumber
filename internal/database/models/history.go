package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// RechargeHistory represents the recharge history document structure
type RechargeHistory struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	UserID        string             `bson:"userId" json:"userId"`
	TransactionID string             `bson:"transaction_id" json:"transaction_id"`
	Amount        string             `bson:"amount" json:"amount"`
	PaymentType   string             `bson:"payment_type" json:"payment_type"`
	DateTime      string             `bson:"date_time" json:"date_time"`
	Status        string             `bson:"status" json:"status"`
}

// TransactionHistory represents the transaction history document structure
type TransactionHistory struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	UserID        string             `bson:"userId" json:"userId"`
	TransactionID string             `bson:"id" json:"id"`
	Number        string             `bson:"number" json:"number"`
	OTP           string             `bson:"otp" json:"otp"`
	DateTime      string             `bson:"date_time" json:"date_time"`
	Service       string             `bson:"service" json:"service"`
	Server        string             `bson:"server" json:"server"`
	Price         string             `bson:"price" json:"price"`
	Status        string             `bson:"status" json:"status"`
}

// InitializeRechargeHistoryCollection initializes the recharge history collection
func InitializeRechargeHistoryCollection(db *mongo.Database) *mongo.Collection {
	return db.Collection("rechargehistories")
}

// InitializeTransactionHistoryCollection initializes the transaction history collection
func InitializeTransactionHistoryCollection(db *mongo.Database) *mongo.Collection {
	return db.Collection("transactionhistories")
}
