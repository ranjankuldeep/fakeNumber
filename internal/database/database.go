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
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatalf("Could not connect to MongoDB: %v", err)
	}

	err = client.Ping(context.Background(), readpref.Primary())
	if err != nil {
		log.Fatalf("Could not ping MongoDB: %v", err)
		return nil, err
	}

	dbName := "Express-Backend"
	fmt.Printf("Connected to DB: %s\n", dbName)
	return client, nil
}
