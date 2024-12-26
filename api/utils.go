package api

import (
	"context"
	"errors"
	"strconv"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func parseParamsSiteIdChannelIdFromWS(c *websocket.Conn) (int, int, error) {
	siteId, err := strconv.Atoi(c.Params("siteId"))
	if err != nil {
		return 0, 0, errors.New("invalid siteId")
	}

	channelId, err := strconv.Atoi(c.Params("channelId"))
	if err != nil {
		return 0, 0, errors.New("invalid channelId")
	}

	return siteId, channelId, nil
}

// parseParams parses query parameters from the request context
func parseParamsSiteIdChannelId(c *fiber.Ctx) (int, int, error) {
	siteId, err := strconv.Atoi(c.Params("siteId"))
	if err != nil {
		return 0, 0, errors.New("invalid siteId")
	}

	channelId, err := strconv.Atoi(c.Params("channelId"))
	if err != nil {
		return 0, 0, errors.New("invalid channelId")
	}

	return siteId, channelId, nil
}

// parseParams parses query parameters from the request context
func parseParams(c *fiber.Ctx) (int, int, uint64, uint64, error) {
	siteId, channelId, err := parseParamsSiteIdChannelId(c)
	if err != nil {
		return 0, 0, 0, 0, errors.New("invalid siteId or channelId")
	}

	timeStamp, err := strconv.ParseUint(c.Params("timeStamp"), 10, 64)
	if err != nil {
		return 0, 0, 0, 0, errors.New("invalid timeStamp")
	}

	timeStampEnd, err := strconv.ParseUint(c.Params("timeStampEnd"), 10, 64)
	if err != nil {
		return 0, 0, 0, 0, errors.New("invalid timeStampEnd")
	}

	return siteId, channelId, timeStamp, timeStampEnd, nil
}

func fetchRecordings(ctx context.Context, db *mongo.Database, collectionName string, filter bson.M, siteId, channelId int) ([]Recording, error) {
	collection := db.Collection(collectionName)
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []Recording
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	// Update metadata for each result
	for i := range results {
		results[i].SiteId = siteId
		results[i].ChannelId = channelId
	}
	return results, nil
}

func fetchHumans(ctx context.Context, db *mongo.Database, collectionName string, filter bson.M, siteId, channelId int) ([]Human, error) {
	collection := db.Collection(collectionName)
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []Human
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	// Update metadata for each result
	for i := range results {
		results[i].SiteId = siteId
		results[i].ChannelId = channelId
	}
	return results, nil
}

func fetchVehicles(ctx context.Context, db *mongo.Database, collectionName string, filter bson.M, siteId, channelId int) ([]Vehicle, error) {
	collection := db.Collection(collectionName)
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []Vehicle
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	// Update metadata for each result
	for i := range results {
		results[i].SiteId = siteId
		results[i].ChannelId = channelId
	}
	return results, nil
}

func fetchEvents(ctx context.Context, db *mongo.Database, collectionName string, filter bson.M, siteId, channelId int) ([]Event, error) {
	collection := db.Collection(collectionName)
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []Event
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	// Update metadata for each result
	for i := range results {
		results[i].SiteId = siteId
		results[i].ChannelId = channelId
	}
	return results, nil
}
