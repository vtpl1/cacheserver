package api

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/vtpl1/cacheserver/db"
	"go.mongodb.org/mongo-driver/bson"
)

var errInvalidTimeRange = errors.New("invalid time range")
var errInvalidCommand = errors.New("invalid command")

type collectionConfig struct {
	name       string
	dbName     string
	collName   string
	resultType interface{}
	filter     bson.M
}

// fetchFromCollection fetches data from a MongoDB collection and sends results to a channel
func fetchFromCollection(ctx context.Context, c *websocket.Conn, socketMutex *sync.Mutex, config collectionConfig) ([]interface{}, error) {
	client, err := db.GetDefaultMongoClient()
	if err != nil {
		writeErrorResponse(c, socketMutex, err)
		return nil, err
	}
	collection := client.Database(config.dbName).Collection(config.collName)
	cursor, err := collection.Find(ctx, config.filter)
	if err != nil {
		writeErrorResponse(c, socketMutex, err)
		return nil, err
	}
	defer cursor.Close(ctx) //nolint:errcheck
	var results []interface{}

	if err = writeResponse(c, socketMutex, config.name, "start"); err != nil {
		return results, err
	}
	for cursor.Next(ctx) {
		result := config.resultType
		if err = cursor.Decode(result); err != nil {
			writeErrorResponse(c, socketMutex, err)
			return results, err
		}
		results = append(results, result)
		if err = writeResponse(c, socketMutex, config.name, result); err != nil {
			return results, err
		}
	}
	if err = cursor.Err(); err != nil {
		writeErrorResponse(c, socketMutex, err)
	}
	if err == nil {
		_ = writeResponse(c, socketMutex, config.name, "done")
	}
	return results, err
}

func writeErrorResponse(c *websocket.Conn, socketMutex *sync.Mutex, err error) {
	socketMutex.Lock()
	_ = c.WriteJSON(fiber.Map{"error": err.Error()})
	socketMutex.Unlock()
}

func writeResponse(c *websocket.Conn, socketMutex *sync.Mutex, msgKey string, msg interface{}) error {
	socketMutex.Lock()
	err := c.WriteJSON(fiber.Map{msgKey: msg})
	socketMutex.Unlock()
	return err
}

// TimeLineWSHandler handles WebSocket connections for the timeline endpoint
func TimeLineWSHandler(ctx context.Context, c *websocket.Conn) {
	siteID, channelID, err := parseParamsSiteIDChannelIDFromWS(c)
	if err != nil {
		writeErrorResponse(c, &sync.Mutex{}, err)
		return
	}
	logger := log.With().Int("siteId", siteID).Int("channelId", channelID).Logger()
	var socketMutex sync.Mutex
	var cancel context.CancelFunc
	for {

		var cmd Command
		if err = c.ReadJSON(&cmd); err != nil {
			logger.Error().Err(err).Msg("Failed to read websocket command")
			writeErrorResponse(c, &socketMutex, errInvalidCommand)
			break
		}
		if cmd.DomainMax < cmd.DomainMin {
			writeErrorResponse(c, &socketMutex, errInvalidTimeRange)
			continue
		}
		if cancel != nil {
			cancel()
			cancel = nil
		}

		ctx1, cancel1 := context.WithCancel(ctx)
		cancel = cancel1
		go writeResults(ctx1, cmd, c, &socketMutex, siteID, channelID, &logger, err)
	}
}

func writeResults(ctx context.Context, cmd Command, c *websocket.Conn, socketMutex *sync.Mutex, siteID int, channelID int, logger *zerolog.Logger, err error) {
	logger.Info().Msg("Write result start")
	defer logger.Info().Msg("Write result end")
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

	collectionConfigs := []collectionConfig{
		{"recordings", "ivms_30", fmt.Sprintf("vVideoClips_%d_%d", siteID, channelID), &Recording{}, filter},
		{"humans", "pvaDB", fmt.Sprintf("pva_HUMAN_%d_%d", siteID, channelID), &Human{}, filter},
		{"vehicles", "pvaDB", fmt.Sprintf("pva_VEHICLE_%d_%d", siteID, channelID), &Vehicle{}, filter},
		{"events", "dasDB", "dasEvents", &Event{}, filterWithSiteIDChannelID},
	}

	var wg sync.WaitGroup
	counts := make(map[string]int, len(collectionConfigs))
	for _, config := range collectionConfigs {
		wg.Add(1)
		go func(config collectionConfig,
		) {
			defer wg.Done()
			results, err1 := fetchFromCollection(ctx, c, socketMutex, config)
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
	if err = c.WriteJSON(fiber.Map{"status": fiber.Map{
		"status":    "done",
		"command":   cmd,
		"siteId":    siteID,
		"channelId": channelID,
		"counts":    counts,
	}}); err != nil {
		logger.Error().Err(err).Msg("Recording WriteJSON error")
	}
	socketMutex.Unlock()
	logger.Info().Msg("Timeline data sent")
}
