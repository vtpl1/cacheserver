package db

import (
	"context"
	"sync"
	"time"

	"github.com/go-logr/zerologr"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type mongoClientInstanceOrError struct {
	clientInstance *mongo.Client
	err            error
}

var (
	clientInstances     = make(map[string]*mongoClientInstanceOrError) // Map of clients per connection string
	clientInstancesLock sync.Mutex                                     // Mutex to handle concurrent access
)

// GetMongoClient returns a singleton MongoDB client instance
func GetMongoClient(connectionString string) (*mongo.Client, error) {
	clientInstancesLock.Lock()
	defer clientInstancesLock.Unlock()
	// Check if an instance already exists for this connection string
	if client, exists := clientInstances[connectionString]; exists {
		return client.clientInstance, client.err
	}

	logger := log.With().
		Str("ConnectionString", connectionString).
		Logger()
	sink := zerologr.New(&logger).GetSink()
	// Create a client with our logger options.
	loggerOptions := options.
		Logger().
		SetSink(sink).
		SetMaxDocumentLength(25).
		SetComponentLevel(options.LogComponentCommand, options.LogLevelInfo)
	// MongoDB connection URI
	uri := connectionString // Replace with your MongoDB URI

	// Create MongoDB client options
	clientOptions := options.Client().ApplyURI(uri).SetLoggerOptions(loggerOptions)

	// Connect to MongoDB
	clientInstance, err := mongo.Connect(clientOptions)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to connect to MongoDB:")
		clientInstances[connectionString] = &mongoClientInstanceOrError{nil, err}
		return nil, err
	}

	// Set a timeout for connecting to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	// Ping the MongoDB server to verify the connection
	err = clientInstance.Ping(ctx, nil)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to ping MongoDB")
		clientInstances[connectionString] = &mongoClientInstanceOrError{nil, err}
		return nil, err
	}
	logger.Info().Msg("Connected to MongoDB successfully.")
	// Store the client instance in the map
	mongoClientInstanceOrError := &mongoClientInstanceOrError{clientInstance, nil}
	clientInstances[connectionString] = mongoClientInstanceOrError

	return mongoClientInstanceOrError.clientInstance, nil
}

func GetDefaultMongoClient() (*mongo.Client, error) {
	clientInstancesLock.Lock()
	defer clientInstancesLock.Unlock()
	for _, client := range clientInstances {
		return client.clientInstance, client.err
	}
	log.Error().Msg("No MongoDB client is registered with a connection string")
	return nil, ErrNoDefaultMongoClient
}
