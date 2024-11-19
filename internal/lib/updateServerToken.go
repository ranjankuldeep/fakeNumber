package lib

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ranjankuldeep/fakeNumber/internal/database/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func fetchTokenFromAPI(apiKey string) (string, error) {
	apiURL := fmt.Sprintf("http://www.phantomunion.com:10023/pickCode-api/push/ticket?key=%s", apiKey)
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch token: %s", resp.Status)
	}

	var responseData struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Token string `json:"token"`
		} `json:"data"`
	}

	err = json.NewDecoder(resp.Body).Decode(&responseData)
	if err != nil {
		return "", fmt.Errorf("failed to parse response JSON: %w", err)
	}

	if responseData.Code != "200" || responseData.Data.Token == "" {
		return "", errors.New("token not found or invalid response")
	}

	return responseData.Data.Token, nil
}

func UpdateServerToken(db *mongo.Database) error {
	serverCollection := models.InitializeServerCollection(db)
	var server models.Server
	err := serverCollection.FindOne(context.TODO(), bson.M{"server": 9}).Decode(&server)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			log.Println("Server document not found.")
			return mongo.ErrNoDocuments
		}
		return fmt.Errorf("error finding server document: %w", err)
	}

	newToken, err := fetchTokenFromAPI(server.APIKey)
	if err != nil {
		return fmt.Errorf("error fetching token: %w", err)
	}
	update := bson.M{
		"$set": bson.M{
			"token":     newToken,
			"updatedAt": time.Now(),
		},
	}

	_, err = serverCollection.UpdateOne(context.TODO(), bson.M{"server": 9}, update)
	if err != nil {
		log.Println("Error updating server document:", err)
		return fmt.Errorf("error updating server document: %w", err)
	}
	return nil
}
