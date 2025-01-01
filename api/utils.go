package api

import (
	"context"
	"errors"
	"strconv"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/vtpl1/cacheserver/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var (
	errInvalidSiteID       = errors.New("invalid siteId")
	errInvalidChannelID    = errors.New("invalid channelId")
	errInvalidTimeStamp    = errors.New("invalid timeStamp")
	errInvalidTimeStampEnd = errors.New("invalid timeStampEnd")
)

func parseParamsSiteIDChannelIDFromWS(c *websocket.Conn) (int, int, error) {
	siteID, err := strconv.Atoi(c.Params("siteId"))
	if err != nil {
		return 0, 0, errInvalidSiteID
	}

	channelID, err := strconv.Atoi(c.Params("channelId"))
	if err != nil {
		return 0, 0, errInvalidChannelID
	}

	return siteID, channelID, nil
}

// parseParams parses query parameters from the request context
func parseParamsSiteIDChannelID(c *fiber.Ctx) (int, int, error) {
	siteID, err := strconv.Atoi(c.Params("siteId"))
	if err != nil {
		return 0, 0, errInvalidSiteID
	}

	channelID, err := strconv.Atoi(c.Params("channelId"))
	if err != nil {
		return 0, 0, errInvalidChannelID
	}

	return siteID, channelID, nil
}

// parseParams parses query parameters from the request context
func parseParams(c *fiber.Ctx) (int, int, uint64, uint64, error) {
	siteID, channelID, err := parseParamsSiteIDChannelID(c)
	if err != nil {
		return 0, 0, 0, 0, err
	}

	timeStamp, err := strconv.ParseUint(c.Params("timeStamp"), 10, 64)
	if err != nil {
		return 0, 0, 0, 0, errInvalidTimeStamp
	}

	timeStampEnd, err := strconv.ParseUint(c.Params("timeStampEnd"), 10, 64)
	if err != nil {
		return 0, 0, 0, 0, errInvalidTimeStampEnd
	}

	return siteID, channelID, timeStamp, timeStampEnd, nil
}

func fetchRecordings(ctx context.Context, db *mongo.Database, collectionName string, filter bson.M, siteID, channelID int) ([]models.Recording, error) {
	collection := db.Collection(collectionName)
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx) //nolint:errcheck

	var results []models.Recording
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	// Update metadata for each result
	for i := range results {
		results[i].SiteID = siteID
		results[i].ChannelID = channelID
	}
	return results, nil
}

func fetchHumans(ctx context.Context, db *mongo.Database, collectionName string, filter bson.M, siteID, channelID int) ([]models.Human, error) {
	collection := db.Collection(collectionName)
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx) //nolint:errcheck

	var results []models.Human
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	// Update metadata for each result
	for i := range results {
		results[i].SiteID = siteID
		results[i].ChannelID = channelID
	}
	return results, nil
}

func fetchVehicles(ctx context.Context, db *mongo.Database, collectionName string, filter bson.M, siteID, channelID int) ([]models.Vehicle, error) {
	collection := db.Collection(collectionName)
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx) //nolint:errcheck

	var results []models.Vehicle
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	// Update metadata for each result
	for i := range results {
		results[i].SiteID = siteID
		results[i].ChannelID = channelID
	}
	return results, nil
}

func fetchEvents(ctx context.Context, db *mongo.Database, collectionName string, filter bson.M, siteID, channelID int) ([]models.Event, error) {
	collection := db.Collection(collectionName)
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx) //nolint:errcheck

	var results []models.Event
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	// Update metadata for each result
	for i := range results {
		results[i].SiteID = siteID
		results[i].ChannelID = channelID
	}
	return results, nil
}
