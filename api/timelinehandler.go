package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"github.com/vtpl1/cacheserver/db"
	"go.mongodb.org/mongo-driver/bson"
)

func TimeLineHandler(c *fiber.Ctx) error {
	// Parse parameters
	siteId, err := strconv.Atoi(c.Params("siteId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid siteId")
	}

	channelId, err := strconv.Atoi(c.Params("channelId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid channelId")
	}

	timeStamp, err := strconv.ParseUint(c.Params("timeStamp"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid timeStamp")
	}

	timeStampEnd, err := strconv.ParseUint(c.Params("timeStampEnd"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid timeStampEnd")
	}

	logger := log.With().
		Int("siteId", siteId).
		Int("channelId", channelId).
		Uint64("timeStamp", timeStamp).
		Uint64("timeStampEnd", timeStampEnd).
		Logger()

	// Get MongoDB client
	client, err := db.GetDefaultMongoClient()
	if err != nil {
		logger.Error().Err(err).Msg("Error connecting to MongoDB")
		return c.Status(fiber.StatusInternalServerError).SendString("Error connecting to MongoDB")
	}

	timeline := NewTimeLineResponse()
	var recordings []Recording
	{
		ivms_30 := client.Database("ivms_30")
		ctx, cancel := context.WithTimeout(c.Context(), 1*time.Second)
		defer cancel()

		// Query Recordings
		collectionName := fmt.Sprintf("vVideoClips_%d_%d", siteId, channelId)
		filter := bson.M{"startTimestamp": bson.M{"$gte": timeStamp, "$lte": timeStampEnd}}

		cursor, err := ivms_30.Collection(collectionName).Find(ctx, filter)
		if err != nil {
			logger.Error().Err(err).Msg("Error querying recordings")
			return c.Status(fiber.StatusInternalServerError).SendString("Error querying recordings")
		}
		defer cursor.Close(ctx)

		if err := cursor.All(ctx, &recordings); err != nil {
			logger.Error().Err(err).Msg("Error decoding recordings")
			return c.Status(fiber.StatusInternalServerError).SendString("Error decoding recordings")
		}

		// Update recording metadata
		for i := range recordings {
			recordings[i].SiteId = siteId
			recordings[i].ChannelId = channelId
		}
		logger.Info().Int("recording_count", len(recordings)).Msg("Recordings fetched successfully")
	}

	var humans []Human
	{
		pvaDB := client.Database("pvaDB")
		ctx, cancel := context.WithTimeout(c.Context(), 1*time.Second)
		defer cancel()

		// Query humans
		collectionName := fmt.Sprintf("pva_HUMAN_%d_%d", siteId, channelId)
		filter := bson.M{"startTimestamp": bson.M{"$gte": timeStamp, "$lte": timeStampEnd}}

		cursor, err := pvaDB.Collection(collectionName).Find(ctx, filter)
		if err != nil {
			logger.Error().Err(err).Msg("Error querying humans")
			return c.Status(fiber.StatusInternalServerError).SendString("Error querying humans")
		}
		defer cursor.Close(ctx)

		if err := cursor.All(ctx, &humans); err != nil {
			logger.Error().Err(err).Msg("Error decoding humans")
			return c.Status(fiber.StatusInternalServerError).SendString("Error decoding humans")
		}

		// Update recording metadata
		for i := range humans {
			humans[i].SiteId = siteId
			humans[i].ChannelId = channelId
		}
		logger.Info().Int("Human_count", len(humans)).Msg("Humans fetched successfully")
	}

	var vehicles []Vehicle
	{
		pvaDB := client.Database("pvaDB")
		ctx, cancel := context.WithTimeout(c.Context(), 1*time.Second)
		defer cancel()

		// Query humans
		collectionName := fmt.Sprintf("pva_VEHICLE_%d_%d", siteId, channelId)
		filter := bson.M{"startTimestamp": bson.M{"$gte": timeStamp, "$lte": timeStampEnd}}

		cursor, err := pvaDB.Collection(collectionName).Find(ctx, filter)
		if err != nil {
			logger.Error().Err(err).Msg("Error querying humans")
			return c.Status(fiber.StatusInternalServerError).SendString("Error querying vehicles")
		}
		defer cursor.Close(ctx)

		if err := cursor.All(ctx, &vehicles); err != nil {
			logger.Error().Err(err).Msg("Error decoding humans")
			return c.Status(fiber.StatusInternalServerError).SendString("Error decoding vehicles")
		}

		// Update recording metadata
		for i := range vehicles {
			vehicles[i].SiteId = siteId
			vehicles[i].ChannelId = channelId
		}
		logger.Info().Int("vehicles_count", len(vehicles)).Msg("Vehicles fetched successfully")
	}

	var events []Event
	{
		dasEvents := client.Database("dasEvents")
		ctx, cancel := context.WithTimeout(c.Context(), 1*time.Second)
		defer cancel()

		// Query events
		collectionName := fmt.Sprint("dasEvents")
		filter := bson.M{"siteId": siteId, "channelId": channelId, "startTimestamp": bson.M{"$gte": timeStamp, "$lte": timeStampEnd}}

		cursor, err := dasEvents.Collection(collectionName).Find(ctx, filter)
		if err != nil {
			logger.Error().Err(err).Msg("Error querying events")
			return c.Status(fiber.StatusInternalServerError).SendString("Error querying events")
		}
		defer cursor.Close(ctx)

		if err := cursor.All(ctx, &events); err != nil {
			logger.Error().Err(err).Msg("Error decoding events")
			return c.Status(fiber.StatusInternalServerError).SendString("Error decoding events")
		}

		// Update events metadata
		for i := range events {
			events[i].SiteId = siteId
			events[i].ChannelId = channelId
		}
		logger.Info().Int("events_count", len(vehicles)).Msg("Events fetched successfully")
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
