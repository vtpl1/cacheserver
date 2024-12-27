package db_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vtpl1/cacheserver/db"
)

func TestGetMongoClient(t *testing.T) {
	client, shouldReturnError := db.GetDefaultMongoClient()

	if !errors.Is(shouldReturnError, db.ErrNoDefaultMongoClient) {
		t.Error("GetDefaultMongoClient should return an error if the connection string is invalid")
	}
	assert.Nil(t, client, "MongoDB client should be nil if an error occurs")

	// Test 1: Ensure GetMongoClient returns a non-nil client
	connectionString := "mongodb://root:root%40central1234@172.236.106.28:27017/"

	ctx := context.Background()

	client1, err := db.GetMongoClient(ctx, connectionString)
	assert.NotNil(t, client1, "MongoDB client should not be nil")
	assert.NoError(t, err, "GetMongoClient should not return an error")

	// Test 2: Ensure GetMongoClient returns the same instance (singleton)
	client2, err := db.GetMongoClient(ctx, connectionString)
	assert.Equal(t, client1, client2, "GetMongoClient should return the same instance (singleton)")
	assert.NoError(t, err, "GetMongoClient should not return an error")

	client3, err := db.GetDefaultMongoClient()
	assert.Equal(t, client1, client3, "GetDefaultMongoClient should return the same instance (singleton)")
	assert.NoError(t, err, "GetDefaultMongoClient should not return an error")

	// Test 3: Verify the client is connected by pinging the server
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = client1.Ping(ctx, nil)
	assert.NoError(t, err, "Ping should succeed if MongoDB client is properly connected")
	err = client1.Disconnect(context.Background())

	assert.NoError(t, err, "Disconnecting MongoDB client should not produce an error")
}

func TestInvalidGetMongoClient(t *testing.T) {
	ctx := context.TODO()
	client, shouldReturnError := db.GetMongoClient(ctx, "connectionString")
	if shouldReturnError == nil {
		t.Error("GetMongoClient should return an error if the connection string is invalid")
	}
	assert.Nil(t, client, "MongoDB client should be nil if an error occurs")
}

func TestIfPingFailsGetMongoClient(t *testing.T) {
	ctx := context.TODO()
	client, shouldReturnError := db.GetMongoClient(ctx, "mongodb://root:root%40central1234@localhost:27017/")
	if shouldReturnError == nil {
		t.Error("GetMongoClient should return an error if mongo server is not reachable")
	}
	assert.Nil(t, client, "MongoDB client should be nil if an error occurs")
}
