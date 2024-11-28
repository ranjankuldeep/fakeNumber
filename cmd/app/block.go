package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/ranjankuldeep/fakeNumber/internal/database/models"
	"github.com/ranjankuldeep/fakeNumber/internal/services"
	"github.com/ranjankuldeep/fakeNumber/internal/utils"
	"github.com/ranjankuldeep/fakeNumber/logs"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func fetchAllUsers(ctx context.Context, db *mongo.Database) ([]models.User, error) {
	var users []models.User
	userCollection := models.InitializeUserCollection(db)
	cursor, err := userCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("failed to decode users: %w", err)
	}
	return users, nil
}

func fetchRechargeSum(ctx context.Context, rechargeHistoryCollection *mongo.Collection, userID string) (float64, error) {
	cursor, err := rechargeHistoryCollection.Find(ctx, bson.M{"userId": userID})
	if err != nil {
		return 0, fmt.Errorf("failed to query recharge histories for user %s: %w", userID, err)
	}
	defer cursor.Close(ctx)

	totalRecharge := 0.0
	for cursor.Next(ctx) {
		var recharge models.RechargeHistory
		if err := cursor.Decode(&recharge); err != nil {
			log.Printf("Failed to decode recharge history: %v", err)
			continue
		}
		amount, err := strconv.ParseFloat(recharge.Amount, 64)
		if err != nil {
			log.Printf("Invalid recharge amount for user %s: %v", userID, err)
			continue
		}
		totalRecharge += amount
	}
	return totalRecharge, nil
}
func fetchSuccessTransactionSum(ctx context.Context, transactionHistoryCollection *mongo.Collection, userID string) (float64, error) {
	cursor, err := transactionHistoryCollection.Find(ctx, bson.M{"userId": userID, "status": "SUCCESS"})
	if err != nil {
		return 0, fmt.Errorf("failed to query transaction histories for user %s: %w", userID, err)
	}
	defer cursor.Close(ctx)

	totalPrice := 0.0
	for cursor.Next(ctx) {
		var transaction models.TransactionHistory
		if err := cursor.Decode(&transaction); err != nil {
			log.Printf("Failed to decode transaction history: %v", err)
			continue
		}
		price, err := strconv.ParseFloat(transaction.Price, 64)
		if err != nil {
			log.Printf("Invalid transaction price for user %s: %v", userID, err)
			continue
		}
		totalPrice += price
	}
	return totalPrice, nil
}

func fetchPendingTransactionSum(ctx context.Context, transactionHistoryCollection *mongo.Collection, userID string) (float64, error) {
	cursor, err := transactionHistoryCollection.Find(ctx, bson.M{"userId": userID, "status": "PENDING"})
	if err != nil {
		return 0, fmt.Errorf("failed to query transaction histories for user %s: %w", userID, err)
	}
	defer cursor.Close(ctx)

	totalPrice := 0.0
	for cursor.Next(ctx) {
		var transaction models.TransactionHistory
		if err := cursor.Decode(&transaction); err != nil {
			log.Printf("Failed to decode transaction history: %v", err)
			continue
		}
		price, err := strconv.ParseFloat(transaction.Price, 64)
		if err != nil {
			log.Printf("Invalid transaction price for user %s: %v", userID, err)
			continue
		}
		totalPrice += price
	}
	return totalPrice, nil
}

func fetchWalletBalance(ctx context.Context, apiWalletCollection *mongo.Collection, userID primitive.ObjectID) (float64, error) {
	var wallet models.ApiWalletUser
	err := apiWalletCollection.FindOne(ctx, bson.M{"userId": userID}).Decode(&wallet)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch wallet for user %s: %w", userID.Hex(), err)
	}

	return wallet.Balance, nil
}

func CheckAndBlockUsers(db *mongo.Database) {
	userCollection := models.InitializeUserCollection(db)
	rechargeHistoryCollection := models.InitializeRechargeHistoryCollection(db)
	apiWalletCollection := models.InitializeApiWalletuserCollection(db)
	transactionHistoryCollection := models.InitializeTransactionHistoryCollection(db)

	ctx := context.Background()
	users, err := fetchAllUsers(ctx, db)
	if err != nil {
		log.Fatalf("Error fetching users: %v", err)
	}

	for _, user := range users {
		if user.Blocked == true {
			continue
		}

		totalRecharge, err := fetchRechargeSum(ctx, rechargeHistoryCollection, user.ID.Hex())
		if err != nil {
			log.Printf("Error fetching recharge sum for user %s: %v", user.ID.Hex(), err)
			continue
		}

		walletBalance, err := fetchWalletBalance(ctx, apiWalletCollection, user.ID)
		if err != nil {
			log.Printf("Error fetching wallet balance for user %s: %v", user.ID.Hex(), err)
			continue
		}

		totalTransactionSuccessPrice, err := fetchSuccessTransactionSum(ctx, transactionHistoryCollection, user.ID.Hex())
		if err != nil {
			log.Printf("Error fetching successful transaction sum for user %s: %v", user.ID.Hex(), err)
			continue
		}

		totalTransactionPendingPrice, err := fetchPendingTransactionSum(ctx, transactionHistoryCollection, user.ID.Hex())
		if err != nil {
			log.Printf("Error fetching successful transaction sum for user %s: %v", user.ID.Hex(), err)
			continue
		}

		// Adjusted balance difference calculation
		adjustedTotal := totalRecharge - (totalTransactionSuccessPrice + totalTransactionPendingPrice)
		divider := adjustedTotal
		if adjustedTotal == 0 {
			divider = 1
		}
		balanceDifference := (walletBalance - adjustedTotal) / divider * 100
		// logs.Logger.Infof("balnce differnce %f for user %v", balanceDifference, user.ID.String())

		// Block user if difference exceeds ±0.1%
		if balanceDifference < -0.2 || balanceDifference > 0.2 {
			update := bson.M{
				"$set": bson.M{
					"blocked": true,
				},
			}
			_, err = userCollection.UpdateOne(ctx, bson.M{"_id": user.ID}, update)
			if err != nil {
				log.Printf("Failed to block user %s: %v", user.ID.Hex(), err)
			} else {
				log.Printf("User %s blocked due to balance mismatch (%.4f%% difference)", user.ID.Hex(), balanceDifference)
				ipDetails, err := utils.GetIpDetails()

				blockDetails := services.BlockUserDetails{
					Email:     user.Email,
					Reason:    fmt.Sprintf("User %s blocked due to balance mismatch (%.4f%% difference)", user.Email, balanceDifference),
					Date:      time.Now().Format("02-01-2006 03:04:05pm"),
					IpDetails: ipDetails,
				}
				err = services.UserBlockDetails(blockDetails)
				if err != nil {
					logs.Logger.Info("unable to send block details")
					logs.Logger.Error(err)
				}
			}
		}
	}
}
