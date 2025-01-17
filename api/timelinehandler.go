// Package api contains the data models for the API
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"github.com/vtpl1/cacheserver/db"
	"github.com/vtpl1/cacheserver/models"
	"go.mongodb.org/mongo-driver/bson"
)

// TimeLineHandler handles timeline requests
func TimeLineHandler(c *fiber.Ctx) error {
	siteID, channelID, timeStamp, timeStampEnd, err := parseParams(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	logger := log.With().
		Int("siteId", siteID).
		Int("channelId", channelID).
		Uint64("timeStamp", timeStamp).
		Uint64("timeStampEnd", timeStampEnd).
		Str("span", time.UnixMilli(int64(timeStampEnd)).Sub(time.UnixMilli(int64(timeStamp))).String()).
		Logger()
	if timeStampEnd < timeStamp {
		logger.Error().Msg("Invalid time range")
		return c.Status(fiber.StatusBadRequest).SendString("Invalid time range")
	}
	client, err := db.GetDefaultMongoClient()
	if err != nil {
		logger.Error().Err(err).Msg("Error connecting to MongoDB")
		return c.Status(fiber.StatusInternalServerError).SendString("Error connecting to MongoDB")
	}

	timeline := models.NewTimeLineResponse()
	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	var recordings []models.Recording
	var humans []models.Human
	var vehicles []models.Vehicle
	var events []models.Event
	var recordingsQueryErr error
	var humansQueryErr error
	var vehiclesQueryErr error
	var eventsQueryErr error

	wg.Add(4) // We have 4 goroutines to wait for
	filterTimeStamp := bson.M{"$or": []bson.M{
		{"startTimestamp": bson.M{"$gte": timeStamp, "$lte": timeStampEnd}},
		{"endTimestamp": bson.M{"$gte": timeStamp, "$lte": timeStampEnd}},
		{"$and": []bson.M{
			{"startTimestamp": bson.M{"$lte": timeStamp}},
			{"endTimestamp": bson.M{"$gte": timeStampEnd}},
		}},
	}}

	// Fetch recordings in parallel
	go func() {
		defer wg.Done()
		recordings, recordingsQueryErr = fetchRecordings(ctx, client.Database("ivms_30"), fmt.Sprintf("vVideoClips_%d_%d", siteID, channelID), filterTimeStamp, siteID, channelID)
	}()

	// Fetch humans in parallel
	go func() {
		defer wg.Done()
		humans, humansQueryErr = fetchHumans(ctx, client.Database("pvaDB"), fmt.Sprintf("pva_HUMAN_%d_%d", siteID, channelID), filterTimeStamp, siteID, channelID)
	}()

	// Fetch vehicles in parallel
	go func() {
		defer wg.Done()
		vehicles, vehiclesQueryErr = fetchVehicles(ctx, client.Database("pvaDB"), fmt.Sprintf("pva_VEHICLE_%d_%d", siteID, channelID), filterTimeStamp, siteID, channelID)
	}()

	// Fetch events in parallel
	go func() {
		defer wg.Done()
		events, eventsQueryErr = fetchEvents(ctx, client.Database("dasEvents"), "dasEvents", bson.M{
			"siteId":         siteID,
			"channelId":      channelID,
			"startTimestamp": bson.M{"$gte": timeStamp, "$lte": timeStampEnd},
		}, siteID, channelID)
	}()

	// Wait for all goroutines to complete
	wg.Wait()

	// If any query failed, return the error
	if recordingsQueryErr != nil || humansQueryErr != nil || vehiclesQueryErr != nil || eventsQueryErr != nil {
		err = errors.Join(recordingsQueryErr, humansQueryErr, vehiclesQueryErr, eventsQueryErr)
		logger.Error().Err(err).Msg("Error fetching data")
		return c.Status(fiber.StatusInternalServerError).SendString("Error fetching data")
	}

	// Populate timeline response
	if recordings != nil {
		logger.Info().
			Int("count", len(recordings)).
			Msg("Recordings found")
		timeline.Results[0].Recordings = recordings
	}
	if humans != nil {
		logger.Info().
			Int("count", len(humans)).
			Msg("Humans found")
		timeline.Results[0].Humans = humans
	}
	if vehicles != nil {
		logger.Info().Int("count", len(vehicles)).
			Msg("Vehicles found")
		timeline.Results[0].Vehicles = vehicles
	}
	if events != nil {
		logger.Info().Int("count", len(events)).
			Msg("Events found")
		timeline.Results[0].Events = events
	}

	// Marshal timeline response
	data, err := json.MarshalIndent(timeline, "", "  ")
	if err != nil {
		logger.Error().Err(err).Msg("Error marshaling JSON")
		return c.Status(fiber.StatusInternalServerError).SendString("Error marshaling JSON")
	}

	return c.Send(data)
}
