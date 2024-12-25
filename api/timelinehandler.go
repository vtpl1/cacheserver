package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"github.com/vtpl1/cacheserver/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// TimeLineHandler handles timeline requests
func TimeLineHandler(c *fiber.Ctx) error {
	siteId, channelId, timeStamp, timeStampEnd, err := parseParams(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	logger := log.With().
		Int("siteId", siteId).
		Int("channelId", channelId).
		Uint64("timeStamp", timeStamp).
		Uint64("timeStampEnd", timeStampEnd).
		Logger()

	client, err := db.GetDefaultMongoClient()
	if err != nil {
		logger.Error().Err(err).Msg("Error connecting to MongoDB")
		return c.Status(fiber.StatusInternalServerError).SendString("Error connecting to MongoDB")
	}

	timeline := NewTimeLineResponse()
	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	var recordings []Recording
	var humans []Human
	var vehicles []Vehicle
	var events []Event
	var recordingsQueryErr error
	var humansQueryErr error
	var vehiclesQueryErr error
	var eventsQueryErr error

	wg.Add(4) // We have 4 goroutines to wait for

	// Fetch recordings in parallel
	go func() {
		defer wg.Done()
		recordings, recordingsQueryErr = fetchRecordings(ctx, client.Database("ivms_30"), fmt.Sprintf("vVideoClips_%d_%d", siteId, channelId), bson.M{
			"startTimestamp": bson.M{"$gte": timeStamp, "$lte": timeStampEnd},
		}, siteId, channelId)
	}()

	// Fetch humans in parallel
	go func() {
		defer wg.Done()
		humans, humansQueryErr = fetchHumans(ctx, client.Database("pvaDB"), fmt.Sprintf("pva_HUMAN_%d_%d", siteId, channelId), bson.M{
			"startTimestamp": bson.M{"$gte": timeStamp, "$lte": timeStampEnd},
		}, siteId, channelId)
	}()

	// Fetch vehicles in parallel
	go func() {
		defer wg.Done()
		vehicles, vehiclesQueryErr = fetchVehicles(ctx, client.Database("pvaDB"), fmt.Sprintf("pva_VEHICLE_%d_%d", siteId, channelId), bson.M{
			"startTimestamp": bson.M{"$gte": timeStamp, "$lte": timeStampEnd},
		}, siteId, channelId)
	}()

	// Fetch events in parallel
	go func() {
		defer wg.Done()
		events, eventsQueryErr = fetchEvents(ctx, client.Database("dasEvents"), "dasEvents", bson.M{
			"siteId":         siteId,
			"channelId":      channelId,
			"startTimestamp": bson.M{"$gte": timeStamp, "$lte": timeStampEnd},
		}, siteId, channelId)
	}()

	// Wait for all goroutines to complete
	wg.Wait()

	// If any query failed, return the error
	if recordingsQueryErr != nil || humansQueryErr != nil || vehiclesQueryErr != nil || eventsQueryErr != nil {
		err := errors.Join(recordingsQueryErr, humansQueryErr, vehiclesQueryErr, eventsQueryErr)
		logger.Error().Err(err).Msg("Error fetching data")
		return c.Status(fiber.StatusInternalServerError).SendString("Error fetching data")
	}

	// Populate timeline response
	if recordings != nil {
		timeline.Results[0].Recordings = recordings
	}
	if humans != nil {
		timeline.Results[0].Humans = humans
	}
	if vehicles != nil {
		timeline.Results[0].Vehicles = vehicles
	}
	if events != nil {
		timeline.Results[0].Events = events
	}

	// Marshal timeline response
	data, err := json.MarshalIndent(timeline, "", "  ")
	if err != nil {
		logger.Error().Err(err).Msg("Error marshalling JSON")
		return c.Status(fiber.StatusInternalServerError).SendString("Error marshalling JSON")
	}

	return c.Send(data)
}

// parseParams parses query parameters from the request context
func parseParams(c *fiber.Ctx) (int, int, uint64, uint64, error) {
	siteId, err := strconv.Atoi(c.Params("siteId"))
	if err != nil {
		return 0, 0, 0, 0, errors.New("invalid siteId")
	}

	channelId, err := strconv.Atoi(c.Params("channelId"))
	if err != nil {
		return 0, 0, 0, 0, errors.New("invalid channelId")
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
