package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/ranjankuldeep/fakeNumber/internal/database/models"
	"github.com/ranjankuldeep/fakeNumber/internal/handlers"
	"github.com/ranjankuldeep/fakeNumber/logs"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ServerDataUpload struct {
	Server int    `bson:"server" json:"server"`
	Price  string `bson:"price" json:"price"`
	Code   string `bson:"code" json:"code"`
	Otp    string `bson:"otp" json:"otp"`
}

type ServerListUpload struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	Name         string             `bson:"name" json:"name"`
	Service_Code string             `bson:"service_code" json:"service_code"`
	Servers      []ServerDataUpload `bson:"servers" json:"servers"`
	CreatedAt    time.Time          `bson:"createdAt,omitempty" json:"createdAt"`
	UpdatedAt    time.Time          `bson:"updatedAt,omitempty" json:"updatedAt"`
}

func FetchServerData(url string) ([]ServerListUpload, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var serverData []ServerListUpload
	if err := json.Unmarshal(body, &serverData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return serverData, nil
}

func UpdateServerData(db *mongo.Database, ctx context.Context) error {
	url := "https://own5k.in/p/final.php"
	serverData, err := FetchServerData(url)
	if err != nil {
		logs.Logger.Error(err)
		return err
	}
	marginMap, exchangeMap, err := handlers.FetchMarginAndExchangeRate(ctx, db)
	if err != nil {
		logs.Logger.Error(err)
		return err
	}

	for serviceIndex, service := range serverData {
		for serverIndex, server := range service.Servers {
			priceFloat, err := strconv.ParseFloat(server.Price, 64)
			if err != nil {
				fmt.Printf("Invalid price for server %d: %v\n", server.Server, err)
				continue
			}
			serverData[serviceIndex].Servers[serverIndex].Price = fmt.Sprintf("%.2f", priceFloat*exchangeMap[server.Server]+marginMap[server.Server])
		}
	}

	serverListCollection := models.InitializeServerListCollection(db)
	_, err = serverListCollection.DeleteMany(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("failed to clear server list collection: %w", err)
	}

	var documents []interface{}
	for _, data := range serverData {
		data.CreatedAt = time.Now()
		data.UpdatedAt = time.Now()
		documents = append(documents, data)
	}
	_, err = serverListCollection.InsertMany(ctx, documents)
	if err != nil {
		return fmt.Errorf("failed to insert data in batch: %w", err)
	}
	return nil
}

func StartUpdateServerDataTicker(db *mongo.Database) {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	go func() {
		if err := UpdateServerData(db, context.TODO()); err != nil {
			log.Printf("Error in UpdateServerData: %v", err)
		}
	}()

	for {
		select {
		case <-ticker.C:
			if err := UpdateServerData(db, context.TODO()); err != nil {
				log.Printf("Error in UpdateServerData: %v", err)
			}
		}
	}
}
