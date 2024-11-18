package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ServiceCode represents the structure of the service code document
type ServiceCode struct {
	Code      string    `bson:"code" json:"code"`
	Name      string    `bson:"name" json:"name"`
	CreatedAt time.Time `bson:"createdAt,omitempty" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt,omitempty" json:"updatedAt"`
}

// MongoDB connection settings
const mongoURI = "mongodb+srv://test2:amardeep885@cluster0.blfflhg.mongodb.net/Express-Backend?retryWrites=true&w=majority"
const dbName = "Express-Backend"
const collectionName = "serviceCodes"

// Fetch data from the URL
func fetchData(url string) (map[string]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching data: %v", err)
	}
	defer resp.Body.Close()

	var data map[string]string
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, fmt.Errorf("error decoding JSON: %v", err)
	}
	return data, nil
}

// Insert data into MongoDB
func insertDataToMongo(client *mongo.Client, data map[string]string) error {
	collection := client.Database(dbName).Collection(collectionName)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var serviceCodes []interface{}
	currentTime := time.Now()

	for code, name := range data {
		serviceCodes = append(serviceCodes, bson.M{
			"code":      code,
			"name":      name,
			"createdAt": currentTime,
			"updatedAt": currentTime,
		})
	}

	_, err := collection.InsertMany(ctx, serviceCodes)
	if err != nil {
		return fmt.Errorf("error inserting data into MongoDB: %v", err)
	}

	return nil
}

func main() {
	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	// Fetch data from the URL
	url := "https://fastsms.su/stubs/handler_api.php?api_key=d91be54bb695297dd517edfdf7da5add&action=getServices"
	data, err := fetchData(url)
	if err != nil {
		log.Fatalf("Failed to fetch data: %v", err)
	}

	// Insert data into MongoDB
	err = insertDataToMongo(client, data)
	if err != nil {
		log.Fatalf("Failed to insert data into MongoDB: %v", err)
	}

	log.Println("Data inserted successfully into the serviceCodes collection!")
}
