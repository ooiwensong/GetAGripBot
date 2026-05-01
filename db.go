package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func connectDB(ctx context.Context) (*mongo.Client, error) {
	URIdb := os.Getenv("URI_MONGO")
	if URIdb == "" {
		return nil, fmt.Errorf("DB uri is not set")
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(URIdb))
	if err != nil {
		return nil, err
	}

	log.Print("Connected to MongoDB.")
	return client, nil
}
