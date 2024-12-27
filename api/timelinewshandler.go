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

// fetchFromCollection fetches data from a MongoDB collection and sends results to a channel
func fetchFromCollection(ctx context.Context, dbName, collectionName string, filter bson.M, resultType interface{}, c *websocket.Conn, cmd Command, resultKey string, socketMutex *sync.Mutex) ([]interface{}, error) {
	client, err := db.GetDefaultMongoClient()
	if err != nil {
		writeErrorResponse(c, err, socketMutex)
		return nil, err
	}
	collection := client.Database(dbName).Collection(collectionName)
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		writeErrorResponse(c, err, socketMutex)
		return nil, err
	}
	defer cursor.Close(ctx) //nolint:errcheck
	var results []interface{}
	for cursor.Next(ctx) {
		result := resultType
		if err = cursor.Decode(result); err != nil {
			writeErrorResponse(c, err, socketMutex)
			return results, err
		}
		results = append(results, result)
		socketMutex.Lock()
		if err = c.WriteJSON(fiber.Map{
			"commandId": cmd.CommandID,
			resultKey:   result,
		}); err != nil {
			socketMutex.Unlock()
			return results, err
		}
		socketMutex.Unlock()
	}
	if err = cursor.Err(); err != nil {
		writeErrorResponse(c, err, socketMutex)
		return results, err
	}
	return results, nil
}

func writeErrorResponse(c *websocket.Conn, err error, socketMutex *sync.Mutex) {
	socketMutex.Lock()
	_ = c.WriteJSON(fiber.Map{"error": err.Error()})
	socketMutex.Unlock()
}

// TimeLineWSHandler handles WebSocket connections for the timeline endpoint
func TimeLineWSHandler(ctx context.Context, c *websocket.Conn) {
	siteID, channelID, err := parseParamsSiteIDChannelIDFromWS(c)
	if err != nil {
		writeErrorResponse(c, err, &sync.Mutex{})
		return
	}
	logger := log.With().Int("siteId", siteID).Int("channelId", channelID).Logger()
	var socketMutex sync.Mutex

	for {
		var cmd Command
		if err = c.ReadJSON(&cmd); err != nil {
			logger.Error().Err(err).Msg("Failed to read websocket command")
			break
		}

		if cmd.DomainMax < cmd.DomainMin {
			writeErrorResponse(c, errInvalidTimeRange, &socketMutex)
			continue
		}

		filter := bson.M{"$or": []bson.M{
			{"startTimestamp": bson.M{"$gte": cmd.DomainMin, "$lte": cmd.DomainMax}},
			{"endTimestamp": bson.M{"$gte": cmd.DomainMin, "$lte": cmd.DomainMax}},
			{"$and": []bson.M{
				{"startTimestamp": bson.M{"$lte": cmd.DomainMin}},
				{"endTimestamp": bson.M{"$gte": cmd.DomainMax}},
			}},
		}}

		filterWithSiteIDChannelID := bson.M{"siteId": siteID, "channelId": channelID}

		// Copy from the original map to the target map
		for key, value := range filter {
			filterWithSiteIDChannelID[key] = value
		}

		collectionConfigs := []struct {
			name       string
			dbName     string
			collName   string
			resultType interface{}
			filter     bson.M
		}{
			{"recordings", "ivms_30", fmt.Sprintf("vVideoClips_%d_%d", siteID, channelID), &Recording{}, filter},
			{"humans", "pvaDB", fmt.Sprintf("pva_HUMAN_%d_%d", siteID, channelID), &Human{}, filter},
			{"vehicles", "pvaDB", fmt.Sprintf("pva_VEHICLE_%d_%d", siteID, channelID), &Vehicle{}, filter},
			{"events", "dasDB", "dasEvents", &Event{}, filterWithSiteIDChannelID},
		}

		var wg sync.WaitGroup
		counts := make(map[string]int, len(collectionConfigs))
		for _, config := range collectionConfigs {
			wg.Add(1)
			go func(config struct {
				name       string
				dbName     string
				collName   string
				resultType interface{}
				filter     bson.M
			},
			) {
				defer wg.Done()
				results, err1 := fetchFromCollection(ctx, config.dbName, config.collName, config.filter, config.resultType, c, cmd, config.name, &socketMutex)
				if err1 != nil {
					logger.Error().Err(err1).Msg("Failed to fetch data from collection")
					return
				}
				counts[config.name] = len(results)
				logger.Info().Msgf("Fetched %d %s", len(results), config.name)
			}(config)
		}
		wg.Wait()

		socketMutex.Lock()
		if err = c.WriteJSON(fiber.Map{
			"commandId": cmd.CommandID,
			"command":   cmd,
			"siteId":    siteID,
			"channelId": channelID,
			"status":    "done",
			"counts":    counts,
			// "recordingsCount": recordingsCount,
			// "humansCount":     humansCount,
			// "vehiclesCount":   vehiclesCount,
			// "eventsCount":     eventsCount,
		}); err != nil {
			logger.Error().Err(err).Msg("Recording WriteJSON error")
		}
		socketMutex.Unlock()
		logger.Info().Msg("Timeline data sent")
	}
}
