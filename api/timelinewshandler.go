package api

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"github.com/vtpl1/cacheserver/db"
	"go.mongodb.org/mongo-driver/bson"
)

var errInvalidTimeRange = errors.New("invalid time range")

// TimeLineWSHandler handles timeline requests over wbsocket
func TimeLineWSHandler(c *websocket.Conn) {

	siteID, channelID, err := parseParamsSiteIDChannelIDFromWS(c)
	if err != nil {
		if err = c.WriteJSON(fiber.Map{"error": err.Error()}); err != nil {
			log.Error().Err(err).Msg("Unable to comunicate back to client")
		}
		return
	}
	logger := log.With().
		Int("siteId", siteID).
		Int("channelId", channelID).
		Logger()
	client, err := db.GetDefaultMongoClient()
	if err != nil {
		logger.Error().Err(err).Msg("Error connecting to MongoDB")
		if err = c.WriteJSON(fiber.Map{"error": err.Error()}); err != nil {
			log.Error().Err(err).Msg("Unable to comunicate back to client")
		}
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
			if err = c.WriteJSON(fiber.Map{"error": errInvalidTimeRange.Error()}); err != nil {
				logger.Error().Err(err).Msg("Unable to comunicate back to client")
			}
			continue
		}

		var wg sync.WaitGroup
		var socketMutex sync.Mutex
		var recordings []Recording
		var humans []Human
		var vehicles []Vehicle
		var events []Event
		var recordingsQueryErr error
		var humansQueryErr error
		var vehiclesQueryErr error
		var eventsQueryErr error
		// ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
		// defer cancel()
		ctx := context.TODO()
		wg.Add(4) // We have 4 goroutines to wait for
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
			collectionName := fmt.Sprintf("vVideoClips_%d_%d", siteID, channelID)
			collection := db.Collection(collectionName)
			cursor, err1 := collection.Find(ctx, filterTimeStamp)

			if err1 != nil {
				if err = c.WriteJSON(fiber.Map{"error": err1.Error()}); err != nil {
					logger.Error().Err(err1).Msg("Unable to comunicate back to client")
				}
				return
			}
			defer cursor.Close(ctx) //nolint:errcheck
			for cursor.Next(ctx) {
				var result Recording
				if err = cursor.Decode(&result); err != nil {
					logger.Error().Err(err).Msg("Recording cursor Decode error")
					recordingsQueryErr = err
					socketMutex.Lock()
					if err = c.WriteJSON(fiber.Map{"error": err.Error()}); err != nil {
						logger.Error().Err(err1).Msg("Unable to comunicate back to client")
					}
					socketMutex.Unlock()
					return
				}
				result.SiteID = siteID
				result.ChannelID = channelID
				socketMutex.Lock()
				recordings = append(recordings, result)
				if err = c.WriteJSON(fiber.Map{"commandId": cmd.CommandID, "recording": result}); err != nil {
					logger.Error().Err(err).Msg("Recording WriteJSON error")
				}
				socketMutex.Unlock()
			}
			if err = cursor.Err(); err != nil {
				recordingsQueryErr = err
				socketMutex.Lock()
				if err = c.WriteJSON(fiber.Map{"error": err.Error()}); err != nil {
					logger.Error().Err(err).Msg("Unable to comunicate back to client")
				}
				socketMutex.Unlock()
			}
		}()

		// Fetch humans in parallel
		go func() {
			defer wg.Done()
			db := client.Database("pvaDB")
			collectionName := fmt.Sprintf("pva_HUMAN_%d_%d", siteID, channelID)
			collection := db.Collection(collectionName)
			cursor, err1 := collection.Find(ctx, filterTimeStamp)
			if err1 != nil {
				if err = c.WriteJSON(fiber.Map{"error": err.Error()}); err != nil {
					logger.Error().Err(err1).Msg("Unable to comunicate back to client")
				}
				return
			}
			defer cursor.Close(ctx) //nolint:errcheck
			for cursor.Next(ctx) {
				var result Human
				if err = cursor.Decode(&result); err != nil {
					logger.Error().Err(err).Msg("Humans cursor Decode error")
					humansQueryErr = err
					socketMutex.Lock()
					if err = c.WriteJSON(fiber.Map{"error": err.Error()}); err != nil {
						logger.Error().Err(err1).Msg("Unable to comunicate back to client")
					}
					socketMutex.Unlock()
					return
				}
				result.SiteID = siteID
				result.ChannelID = channelID
				socketMutex.Lock()
				humans = append(humans, result)
				if err = c.WriteJSON(fiber.Map{"commandId": cmd.CommandID, "humans": result}); err != nil {
					logger.Error().Err(err).Msg("Humans WriteJSON error")
				}
				socketMutex.Unlock()
			}
			if err = cursor.Err(); err != nil {
				humansQueryErr = err
				socketMutex.Lock()
				if err = c.WriteJSON(fiber.Map{"error": err.Error()}); err != nil {
					logger.Error().Err(err).Msg("Unable to comunicate back to client")
				}
				socketMutex.Unlock()
			}
		}()

		// Fetch vehicles in parallel
		go func() {
			defer wg.Done()
			db := client.Database("pvaDB")
			collectionName := fmt.Sprintf("pva_VEHICLE_%d_%d", siteID, channelID)
			collection := db.Collection(collectionName)
			cursor, err1 := collection.Find(ctx, filterTimeStamp)
			if err1 != nil {
				if err = c.WriteJSON(fiber.Map{"error": err.Error()}); err != nil {
					logger.Error().Err(err1).Msg("Unable to comunicate back to client")
				}
				return
			}
			defer cursor.Close(ctx) //nolint:errcheck
			for cursor.Next(ctx) {
				var result Vehicle
				if err = cursor.Decode(&result); err != nil {
					logger.Error().Err(err).Msg("vehicles cursor Decode error")
					vehiclesQueryErr = err
					socketMutex.Lock()
					if err = c.WriteJSON(fiber.Map{"error": err.Error()}); err != nil {
						logger.Error().Err(err1).Msg("Unable to comunicate back to client")
					}
					socketMutex.Unlock()
					return
				}
				result.SiteID = siteID
				result.ChannelID = channelID
				socketMutex.Lock()
				vehicles = append(vehicles, result)
				if err = c.WriteJSON(fiber.Map{"commandId": cmd.CommandID, "vehicles": result}); err != nil {
					logger.Error().Err(err).Msg("vehicles WriteJSON error")
				}
				socketMutex.Unlock()
			}
			if err = cursor.Err(); err != nil {
				vehiclesQueryErr = err
				socketMutex.Lock()
				if err = c.WriteJSON(fiber.Map{"error": err.Error()}); err != nil {
					logger.Error().Err(err).Msg("Unable to comunicate back to client")
				}
				socketMutex.Unlock()
			}
		}()

		// Fetch events in parallel
		go func() {
			defer wg.Done()
			db := client.Database("dasDB")
			collectionName := "dasEvents"
			collection := db.Collection(collectionName)
			// Create the target map
			filterTimeStamp1 := bson.M{"siteId": siteID, "channelId": channelID}

			// Copy from the original map to the target map
			for key, value := range filterTimeStamp {
				filterTimeStamp1[key] = value
			}

			logger.Info().Msgf("HERE: filterTimeStamp %v", filterTimeStamp1)

			cursor, err1 := collection.Find(ctx, filterTimeStamp1)
			if err1 != nil {
				if err = c.WriteJSON(fiber.Map{"error": err.Error()}); err != nil {
					logger.Error().Err(err1).Msg("Unable to comunicate back to client")
				}
				return
			}
			defer cursor.Close(ctx) //nolint:errcheck
			for cursor.Next(ctx) {
				var result Event
				if err = cursor.Decode(&result); err != nil {
					logger.Error().Err(err).Msg("events cursor Decode error")
					eventsQueryErr = err
					socketMutex.Lock()
					if err = c.WriteJSON(fiber.Map{"error": err.Error()}); err != nil {
						logger.Error().Err(err1).Msg("Unable to comunicate back to client")
					}
					socketMutex.Unlock()
					return
				}
				result.SiteID = siteID
				result.ChannelID = channelID
				socketMutex.Lock()
				events = append(events, result)
				if err = c.WriteJSON(fiber.Map{"commandId": cmd.CommandID, "events": result}); err != nil {
					logger.Error().Err(err).Msg("events WriteJSON error")
				}
				socketMutex.Unlock()
			}
			if err = cursor.Err(); err != nil {
				eventsQueryErr = err
				socketMutex.Lock()
				if err = c.WriteJSON(fiber.Map{"error": err.Error()}); err != nil {
					logger.Error().Err(err).Msg("Unable to comunicate back to client")
				}
				socketMutex.Unlock()
			}
		}()

		// Wait for all goroutines to complete
		wg.Wait()
		// If any query failed, return the error
		if recordingsQueryErr != nil || humansQueryErr != nil || vehiclesQueryErr != nil || eventsQueryErr != nil {
			err = errors.Join(recordingsQueryErr, humansQueryErr, vehiclesQueryErr, eventsQueryErr)
			logger.Error().Err(err).Msg("Error fetching data")
			socketMutex.Lock()
			if err = c.WriteJSON(fiber.Map{"error": err.Error()}); err != nil {
				logger.Error().Err(err).Msg("Unable to comunicate back to client")
			}
			socketMutex.Unlock()
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
		socketMutex.Lock()
		if err = c.WriteJSON(fiber.Map{
			"commandId":       cmd.CommandID,
			"command":         cmd,
			"siteId":          siteID,
			"channelId":       channelID,
			"status":          "done",
			"recordingsCount": recordingsCount,
			"humansCount":     humansCount,
			"vehiclesCount":   vehiclesCount,
			"eventsCount":     eventsCount}); err != nil {
			logger.Error().Err(err).Msg("Recording WriteJSON error")
		}
		socketMutex.Unlock()
	}
}
