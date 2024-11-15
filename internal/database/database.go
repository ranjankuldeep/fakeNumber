package database

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func ConnectDB(uri string) (*mongo.Client, error) {
	// Set client options
	clientOptions := options.Client().ApplyURI(uri)

	// Connect to MongoDB
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatalf("Could not connect to MongoDB: %v", err)
	}

	// Ping the database to verify the connection
	err = client.Ping(context.Background(), readpref.Primary())
	if err != nil {
		log.Fatalf("Could not ping MongoDB: %v", err)
		return nil, err
	}

	// Get the database name
	dbName := "Express-Backend"

	fmt.Printf("Connected to DB: %s\n", dbName)
	return client, nil
}
