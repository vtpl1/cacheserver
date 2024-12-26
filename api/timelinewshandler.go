package api

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"github.com/vtpl1/cacheserver/db"
	"go.mongodb.org/mongo-driver/bson"
)

// TimeLineHandler handles timeline requests
func TimeLineWSHandler(c *websocket.Conn) {

	siteId, channelId, err := parseParamsSiteIdChannelIdFromWS(c)
	if err != nil {
		c.WriteJSON(fiber.Map{"error": err.Error()})
		return
	}
	logger := log.With().
		Int("siteId", siteId).
		Int("channelId", channelId).
		Logger()
	client, err := db.GetDefaultMongoClient()
	if err != nil {
		logger.Error().Err(err).Msg("Error connecting to MongoDB")
		c.WriteJSON(fiber.Map{"error": err.Error()})
		return
	}
	for {
		var cmd Command
		if err = c.ReadJSON(&cmd); err != nil {
			logger.Error().Err(err).Msg("read websocket")
			break
		}
		logger.Info().Msgf("read websocket Command %v", cmd)
		timeStamp := cmd.DomainMin
		timeStampEnd := cmd.DomainMax

		if timeStampEnd < timeStamp {
			logger.Error().Msg("Invalid time range")
			c.WriteJSON(fiber.Map{"error": errors.New("invalid time range").Error()})
			continue
		}

		var wg sync.WaitGroup
		var socket_mu sync.Mutex
		var recordings []Recording
		var humans []Human
		var vehicles []Vehicle
		var events []Event
		var recordingsQueryErr error
		var humansQueryErr error
		var vehiclesQueryErr error
		var eventsQueryErr error
		ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
		defer cancel()
		wg.Add(3) // We have 4 goroutines to wait for
		filterTimeStamp := bson.M{"$or": []bson.M{
			{"startTimestamp": bson.M{"$gte": timeStamp, "$lte": timeStampEnd}},
			{"endTimestamp": bson.M{"$gte": timeStamp, "$lte": timeStampEnd}},
			{"$and": []bson.M{
				{"startTimestamp": bson.M{"$lte": timeStamp}},
				{"endTimestamp": bson.M{"$gte": timeStampEnd}},
			}}}}

		// Fetch recordings in parallel
		go func() {
			defer wg.Done()
			db := client.Database("ivms_30")
			collectionName := fmt.Sprintf("vVideoClips_%d_%d", siteId, channelId)
			collection := db.Collection(collectionName)
			cursor, err := collection.Find(ctx, filterTimeStamp)
			if err != nil {
				c.WriteJSON(fiber.Map{"error": err.Error()})
				return
			}
			defer cursor.Close(ctx)
			for cursor.Next(ctx) {
				var result Recording
				if err := cursor.Decode(&result); err != nil {
					logger.Error().Err(err).Msg("Recording cursor Decode error")
					recordingsQueryErr = err
					socket_mu.Lock()
					c.WriteJSON(fiber.Map{"error": err.Error()})
					socket_mu.Unlock()
					return
				}
				result.SiteId = siteId
				result.ChannelId = channelId
				socket_mu.Lock()
				recordings = append(recordings, result)
				if err = c.WriteJSON(fiber.Map{"commandId": cmd.CommandId, "recording": result}); err != nil {
					logger.Error().Err(err).Msg("Recording WriteJSON error")
				}
				socket_mu.Unlock()
			}
			if err := cursor.Err(); err != nil {
				recordingsQueryErr = err
				socket_mu.Lock()
				c.WriteJSON(fiber.Map{"error": err.Error()})
				socket_mu.Unlock()
			}
		}()

		// Fetch humans in parallel
		go func() {
			defer wg.Done()
			db := client.Database("pvaDB")
			collectionName := fmt.Sprintf("pva_HUMAN_%d_%d", siteId, channelId)
			collection := db.Collection(collectionName)
			cursor, err := collection.Find(ctx, filterTimeStamp)
			if err != nil {
				c.WriteJSON(fiber.Map{"error": err.Error()})
				return
			}
			defer cursor.Close(ctx)
			for cursor.Next(ctx) {
				var result Human
				if err := cursor.Decode(&result); err != nil {
					logger.Error().Err(err).Msg("Humans cursor Decode error")
					humansQueryErr = err
					socket_mu.Lock()
					c.WriteJSON(fiber.Map{"error": err.Error()})
					socket_mu.Unlock()
					return
				}
				result.SiteId = siteId
				result.ChannelId = channelId
				socket_mu.Lock()
				humans = append(humans, result)
				if err = c.WriteJSON(fiber.Map{"commandId": cmd.CommandId, "humans": result}); err != nil {
					logger.Error().Err(err).Msg("Humans WriteJSON error")
				}
				socket_mu.Unlock()
			}
			if err := cursor.Err(); err != nil {
				humansQueryErr = err
				socket_mu.Lock()
				c.WriteJSON(fiber.Map{"error": err.Error()})
				socket_mu.Unlock()
			}
		}()

		// Fetch vehicles in parallel
		go func() {
			defer wg.Done()
			db := client.Database("pvaDB")
			collectionName := fmt.Sprintf("pva_VEHICLE_%d_%d", siteId, channelId)
			collection := db.Collection(collectionName)
			cursor, err := collection.Find(ctx, filterTimeStamp)
			if err != nil {
				c.WriteJSON(fiber.Map{"error": err.Error()})
				return
			}
			defer cursor.Close(ctx)
			for cursor.Next(ctx) {
				var result Vehicle
				if err := cursor.Decode(&result); err != nil {
					logger.Error().Err(err).Msg("vehicles cursor Decode error")
					vehiclesQueryErr = err
					socket_mu.Lock()
					c.WriteJSON(fiber.Map{"error": err.Error()})
					socket_mu.Unlock()
					return
				}
				result.SiteId = siteId
				result.ChannelId = channelId
				socket_mu.Lock()
				vehicles = append(vehicles, result)
				if err = c.WriteJSON(fiber.Map{"commandId": cmd.CommandId, "vehicles": result}); err != nil {
					logger.Error().Err(err).Msg("vehicles WriteJSON error")
				}
				socket_mu.Unlock()
			}
			if err := cursor.Err(); err != nil {
				vehiclesQueryErr = err
				socket_mu.Lock()
				c.WriteJSON(fiber.Map{"error": err.Error()})
				socket_mu.Unlock()
			}
		}()

		// // Fetch events in parallel
		// go func() {
		// 	defer wg.Done()
		// 	db := client.Database("dasEvents")
		// 	collectionName := "dasEvents"
		// 	collection := db.Collection(collectionName)
		// 	filterTimeStamp["siteId"] = siteId
		// 	filterTimeStamp["channelId"] = channelId
		// 	logger.Info().Msgf("HERE: filterTimeStamp %v", filterTimeStamp)
		// 	cursor, err := collection.Find(ctx, filterTimeStamp)
		// 	if err != nil {
		// 		c.WriteJSON(fiber.Map{"error": err.Error()})
		// 		return
		// 	}
		// 	defer cursor.Close(ctx)
		// 	for cursor.Next(ctx) {
		// 		var result Event
		// 		if err := cursor.Decode(&result); err != nil {
		// 			logger.Error().Err(err).Msg("Events cursor Decode error")
		// 			eventsQueryErr = err
		// 			socket_mu.Lock()
		// 			c.WriteJSON(fiber.Map{"error": err.Error()})
		// 			socket_mu.Unlock()
		// 			return
		// 		}
		// 		result.SiteId = siteId
		// 		result.ChannelId = channelId
		// 		socket_mu.Lock()
		// 		events = append(events, result)
		// 		if err = c.WriteJSON(fiber.Map{"commandId": cmd.CommandId, "events": result}); err != nil {
		// 			logger.Error().Err(err).Msg("Events WriteJSON error")
		// 		}
		// 		socket_mu.Unlock()
		// 	}
		// 	if err := cursor.Err(); err != nil {
		// 		eventsQueryErr = err
		// 		socket_mu.Lock()
		// 		c.WriteJSON(fiber.Map{"error": err.Error()})
		// 		socket_mu.Unlock()
		// 	}
		// }()

		// Wait for all goroutines to complete
		wg.Wait()
		// If any query failed, return the error
		if recordingsQueryErr != nil || humansQueryErr != nil || vehiclesQueryErr != nil || eventsQueryErr != nil {
			err := errors.Join(recordingsQueryErr, humansQueryErr, vehiclesQueryErr, eventsQueryErr)
			logger.Error().Err(err).Msg("Error fetching data")
			c.WriteJSON(fiber.Map{"error": err.Error()})
		}
		// Populate timeline response
		var recordingsCount int
		if recordings != nil {
			recordingsCount = len(recordings)
			logger.Info().
				Int("count", recordingsCount).
				Msg("Recordings found")
		}
		var humansCount int
		if humans != nil {
			humansCount = len(humans)
			logger.Info().
				Int("count", humansCount).
				Msg("Humans found")
		}
		var vehiclesCount int
		if vehicles != nil {
			vehiclesCount = len(vehicles)
			logger.Info().Int("count", vehiclesCount).
				Msg("Vehicles found")
		}
		var eventsCount int
		if events != nil {
			eventsCount = len(events)
			logger.Info().Int("count", eventsCount).
				Msg("Events found")
		}
		socket_mu.Lock()
		if err = c.WriteJSON(fiber.Map{
			"commandId":       cmd.CommandId,
			"command":         cmd,
			"siteId":          siteId,
			"channelId":       channelId,
			"status":          "done",
			"recordingsCount": recordingsCount,
			"humansCount":     humansCount,
			"vehiclesCount":   vehiclesCount,
			"eventsCount":     eventsCount}); err != nil {
			logger.Error().Err(err).Msg("Recording WriteJSON error")
		}
		socket_mu.Unlock()
	}
}
