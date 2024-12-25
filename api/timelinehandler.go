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

	siteId, err := strconv.Atoi(c.Params("siteId"))
	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).SendString("Error unmarshalling siteId")
	}
	channelId, err := strconv.Atoi(c.Params("channelId"))
	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).SendString("Error unmarshalling channelId")
	}
	timeStamp, err := strconv.ParseUint(c.Params("timeStamp"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).SendString("Error unmarshalling timeStamp")
	}

	timeStampEnd, err := strconv.ParseUint(c.Params("timeStampEnd"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).SendString("Error unmarshalling timeStampEnd")
	}

	logger := log.With().
		Int("siteId", siteId).Int("channelId", channelId).Uint64("timeStamp", timeStamp).Uint64("timeStampEnd", timeStampEnd).
		Logger()

	// MongoDB client setup
	client, err := db.GetDefaultMongoClient() // Assume this function returns a *mongo.Client
	if err != nil {
		logger.Error().Err(err).Msg("Error Connecting to MongoDB")
		return c.Status(fiber.StatusInternalServerError).SendString("Error Connecting to MongoDB")
	}
	timeline := NewTimeLineResponse()

	ivms_30 := client.Database("ivms_30") // Replace with your database name
	ctx, cancel := context.WithTimeout(c.Context(), 1*time.Second)
	defer cancel()
	// Query Recordings
	recordingCursor, err := ivms_30.Collection(fmt.Sprintf("vVideoClips_%d_%d", siteId, channelId)).Find(ctx, bson.M{
		"startTimestamp": bson.M{"$gte": timeStamp, "$lte": timeStampEnd},
	})
	result := Result{}
	if err == nil {
		defer recordingCursor.Close(ctx)
		var recordings []Recording
		err = recordingCursor.All(ctx, &recordings)
		if err == nil {
			logger.Info().Int("count", len(recordings)).Msg("Recordings")
			result.Recordings = append(result.Recordings, recordings...)
		} else {
			logger.Error().Err(err).Msg("Here1")
		}
	} else {
		logger.Error().Err(err).Msg("Here2")
	}

	// // Query Events
	// eventCursor, err := db.Collection("Event").Find(ctx, bson.M{
	// 	"siteId":    siteId,
	// 	"timestamp": bson.M{"$gte": timeStamp, "$lte": timeStampEnd},
	// })
	// if err == nil {
	// 	defer eventCursor.Close(ctx)
	// 	err = eventCursor.All(ctx, &timeline.Results)
	// }

	// // Query Humans
	// humanCursor, err := db.Collection("Human").Find(ctx, bson.M{
	// 	"siteId":    siteId,
	// 	"timestamp": bson.M{"$gte": timeStamp, "$lte": timeStampEnd},
	// })
	// if err == nil {
	// 	defer humanCursor.Close(ctx)
	// 	err = humanCursor.All(ctx, &timeline.Results)
	// }

	// // Query Vehicles
	// vehicleCursor, err := db.Collection("Vehicle").Find(ctx, bson.M{
	// 	"siteId":    siteId,
	// 	"timestamp": bson.M{"$gte": timeStamp, "$lte": timeStampEnd},
	// })
	// if err == nil {
	// 	defer vehicleCursor.Close(ctx)
	// 	err = vehicleCursor.All(ctx, &timeline.Results)
	// }

	if err != nil {
		logger.Error().Err(err).Msg("Error querying MongoDB")
		return c.Status(fiber.StatusInternalServerError).SendString("Error querying MongoDB")
	}
	timeline.Results = append(timeline.Results, result)
	// Send JSON response
	data, err := json.MarshalIndent(timeline, "", "  ")
	if err != nil {
		logger.Error().Err(err).Msg("Error marshalling JSON")
		return c.Status(fiber.StatusInternalServerError).SendString("Error marshalling JSON")
	}
	logger.Info().Msgf("%v", string(data))
	return c.Send(data)
}
