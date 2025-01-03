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
	"github.com/vtpl1/cacheserver/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	maxTimeGapAllowedInmSecForSecond    = 1000
	maxTimeGapAllowedInmSecFor10Second  = 10 * maxTimeGapAllowedInmSecForSecond
	maxTimeGapAllowedInmSecForMinute    = 60 * maxTimeGapAllowedInmSecForSecond
	maxTimeGapAllowedInmSecFor10Minutes = 10 * maxTimeGapAllowedInmSecForMinute
	maxTimeGapAllowedInmSecFor12Minutes = 12 * maxTimeGapAllowedInmSecForMinute
	maxTimeGapAllowedInmSecFor24Minutes = 24 * maxTimeGapAllowedInmSecForMinute
	maxTimeGapAllowedInmSecForHour      = 60 * maxTimeGapAllowedInmSecForMinute
	maxTimeGapAllowedInmSecForDay       = 24 * maxTimeGapAllowedInmSecForHour
	maxTimeGapAllowedInmSecFor3Days     = 3 * maxTimeGapAllowedInmSecForDay
	maxTimeGapAllowedInmSecForWeek      = 7 * maxTimeGapAllowedInmSecForDay
	maxTimeGapAllowedInmSecFor15Day     = 15 * maxTimeGapAllowedInmSecForDay
	maxTimeGapAllowedInmSecForMonth     = int64(30 * maxTimeGapAllowedInmSecForDay)
	maxTimeGapAllowedInmSecFor3Months   = 3 * maxTimeGapAllowedInmSecForMonth
	maxTimeGapAllowedInmSecFor6Months   = 3 * maxTimeGapAllowedInmSecForMonth
	maxTimeGapAllowedInmSecForYear      = 12 * maxTimeGapAllowedInmSecForMonth
)

var (
	errInvalidTimeRange = errors.New("invalid time range")
	errInvalidCommand   = errors.New("invalid command")
)

type collectionConfig struct {
	name       string
	dbName     string
	collName   string
	resultType interface{}
	filter     bson.A
	commandID  string
}

// fetchFromCollection fetches data from a MongoDB collection and sends results to a channel
func fetchFromCollection(ctx context.Context, c *websocket.Conn, socketMutex *sync.Mutex, config collectionConfig) ([]interface{}, error) {
	client, err := db.GetDefaultMongoClient()
	if err != nil {
		writeErrorResponse(c, socketMutex, err)
		return nil, err
	}
	collection := client.Database(config.dbName).Collection(config.collName)
	// allowDiskUse := true
	opts := options.Aggregate().SetAllowDiskUse(true)

	cursor, err := collection.Aggregate(ctx, config.filter, opts)
	if err != nil {
		writeErrorResponse(c, socketMutex, err)
		return nil, err
	}
	defer cursor.Close(ctx) //nolint:errcheck
	var results []interface{}

	if err = writeResponse(c, socketMutex, config.name, fiber.Map{"commandId": config.commandID, "status": "start"}); err != nil {
		return results, err
	}
	for cursor.Next(ctx) {
		result := config.resultType
		if err = cursor.Decode(result); err != nil {
			_ = writeResponse(c, socketMutex, config.name, fiber.Map{"commandId": config.commandID, "status": "error"})
			return results, err
		}
		switch v := result.(type) {
		case *models.Recording:
			v.CommandID = config.commandID
		case *models.Human:
			v.CommandID = config.commandID
			result = v
		case *models.Vehicle:
			v.CommandID = config.commandID
			result = v
		case *models.Event:
			v.CommandID = config.commandID
			result = v
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
		_ = writeResponse(c, socketMutex, config.name, fiber.Map{"commandId": config.commandID, "status": "done"})
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
		var cmd models.Command
		if err = c.ReadJSON(&cmd); err != nil {
			logger.Error().Err(err).Msg("Failed to read websocket command")
			writeErrorResponse(c, &socketMutex, errInvalidCommand)
			break
		}
		if cancel != nil {
			logger.Info().Msg("Canceling previous command")
			cancel()
		}
		var ctx1 context.Context
		ctx1, cancel = context.WithCancel(ctx)
		defer cancel()
		go func() {
			writeResults(ctx1, cmd, c, &socketMutex, siteID, channelID, &logger)
			cancel()
		}()
	}
}

func writeResults(ctx context.Context, cmd models.Command, c *websocket.Conn, socketMutex *sync.Mutex, siteID int, channelID int, logger *zerolog.Logger) {
	if cmd.DomainMax < cmd.DomainMin {
		logger.Error().Err(errInvalidTimeRange)
		writeErrorResponse(c, socketMutex, errInvalidTimeRange)
		return
	}

	var maxTimeGapAllowedInmSec int64
	domainMax := int64(cmd.DomainMax)
	domainMin := int64(cmd.DomainMin)
	diff := domainMax - domainMin
	switch {
	case diff < maxTimeGapAllowedInmSecForDay:
		maxTimeGapAllowedInmSec = 100
	case diff < maxTimeGapAllowedInmSecFor15Day:
		maxTimeGapAllowedInmSec = maxTimeGapAllowedInmSecFor10Second
	case diff < maxTimeGapAllowedInmSecForMonth:
		maxTimeGapAllowedInmSec = maxTimeGapAllowedInmSecForMinute
	case diff < maxTimeGapAllowedInmSecFor3Months:
		maxTimeGapAllowedInmSec = maxTimeGapAllowedInmSecFor12Minutes
	case diff < maxTimeGapAllowedInmSecFor6Months:
		maxTimeGapAllowedInmSec = maxTimeGapAllowedInmSecFor24Minutes
	default:
		maxTimeGapAllowedInmSec = maxTimeGapAllowedInmSecForHour
	}
	logger.Info().Int64("maxTimeGapAllowedInmSec", maxTimeGapAllowedInmSec).Send()

	matchSiteIDChannelIDStage := bson.D{
		{
			Key: "$match",
			Value: bson.D{
				{Key: "siteId", Value: siteID},
				{Key: "channelId", Value: channelID},
			},
		},
	}
	matchStage := bson.D{
		{
			Key: "$match",
			Value: bson.D{
				{
					Key: "$or",
					Value: bson.A{
						bson.D{
							{
								Key: "startTimestamp",
								Value: bson.D{
									{Key: "$gte", Value: domainMin},
									{Key: "$lte", Value: domainMax},
								},
							},
						},
						bson.D{
							{
								Key: "endTimestamp",
								Value: bson.D{
									{Key: "$gte", Value: domainMin},
									{Key: "$lte", Value: domainMax},
								},
							},
						},
						bson.D{
							{
								Key: "$and",
								Value: bson.A{
									bson.D{{Key: "startTimestamp", Value: bson.D{{Key: "$lte", Value: domainMin}}}},
									bson.D{{Key: "endTimestamp", Value: bson.D{{Key: "$gte", Value: domainMax}}}},
								},
							},
						},
					},
				},
			},
		},
	}
	sortStage := bson.D{{Key: "$sort", Value: bson.D{{Key: "startTimestamp", Value: 1}}}}
	effectiveEndTimestampAddFieldsStage := bson.D{
		{
			Key: "$addFields",
			Value: bson.D{
				{
					Key: "effectiveEndTimestamp",
					Value: bson.D{
						{
							Key: "$add",
							Value: bson.A{
								"$endTimestamp",
								maxTimeGapAllowedInmSec,
							},
						},
					},
				},
			},
		},
	}
	prevEffectiveEndTimestampSetWindowFieldsStage := bson.D{
		{
			Key: "$setWindowFields",
			Value: bson.D{
				{Key: "sortBy", Value: bson.D{{Key: "startTimestamp", Value: 1}}},
				{
					Key: "output",
					Value: bson.D{
						{
							Key: "prevEffectiveEndTimestamp",
							Value: bson.D{
								{
									Key: "$shift",
									Value: bson.D{
										{Key: "output", Value: "$effectiveEndTimestamp"},
										{Key: "by", Value: -1},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	boundarySetStage := bson.D{
		{
			Key: "$set",
			Value: bson.D{
				{
					Key: "boundary",
					Value: bson.D{
						{
							Key: "$sum",
							Value: bson.D{
								{
									Key: "$cond",
									Value: bson.A{
										bson.D{
											{
												Key: "$or",
												Value: bson.A{
													bson.D{
														{
															Key: "$eq",
															Value: bson.A{
																"$prevEffectiveEndTimestamp",
																bson.Null{},
															},
														},
													},
													bson.D{
														{
															Key: "$lt",
															Value: bson.A{
																"$prevEffectiveEndTimestamp",
																"$startTimestamp",
															},
														},
													},
												},
											},
										},
										1,
										0,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	uniqueGroupIDSetWindowFieldStage := bson.D{
		{
			Key: "$setWindowFields",
			Value: bson.D{
				{Key: "sortBy", Value: bson.D{{Key: "startTimestamp", Value: 1}}},
				{
					Key: "output",
					Value: bson.D{
						{
							Key: "groupId",
							Value: bson.D{
								{Key: "$sum", Value: "$boundary"},
								{
									Key: "window",
									Value: bson.D{
										{
											Key: "documents",
											Value: bson.A{
												"unbounded",
												"current",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	recalculateGroupstage := bson.D{
		{
			Key: "$group",
			Value: bson.D{
				{Key: "_id", Value: "$groupId"},
				{Key: "startTimestamp", Value: bson.D{{Key: "$first", Value: "$startTimestamp"}}},
				{Key: "endTimestamp", Value: bson.D{{Key: "$last", Value: "$endTimestamp"}}},
				{Key: "objectCount", Value: bson.D{{Key: "$sum", Value: "$objectCount"}}},
			},
		},
	}
	finalSortStage := bson.D{{Key: "$sort", Value: bson.D{{Key: "startTimestamp", Value: 1}}}}
	filter := bson.A{
		matchStage,
		sortStage,
		effectiveEndTimestampAddFieldsStage,
		prevEffectiveEndTimestampSetWindowFieldsStage,
		boundarySetStage,
		uniqueGroupIDSetWindowFieldStage,
		recalculateGroupstage,
		finalSortStage,
	}
	filterWithSiteIDChannelID := bson.A{matchSiteIDChannelIDStage}
	filterWithSiteIDChannelID = append(filterWithSiteIDChannelID, filter...)

	collectionConfigs := []collectionConfig{
		{"recordings", "ivms_30", fmt.Sprintf("vVideoClips_%d_%d", siteID, channelID), &models.Recording{}, filter, cmd.CommandID},
		{"humans", "pvaDB", fmt.Sprintf("pva_HUMAN_%d_%d", siteID, channelID), &models.Human{}, filter, cmd.CommandID},
		{"vehicles", "pvaDB", fmt.Sprintf("pva_VEHICLE_%d_%d", siteID, channelID), &models.Vehicle{}, filter, cmd.CommandID},
		{"events", "dasDB", "dasEvents", &models.Event{}, filterWithSiteIDChannelID, cmd.CommandID},
	}
	socketMutex.Lock()
	if err := c.WriteJSON(fiber.Map{"status": fiber.Map{
		"status":    "start",
		"command":   cmd,
		"siteId":    siteID,
		"channelId": channelID,
	}}); err != nil {
		logger.Error().Err(err).Msg("WriteJSON error")
	}
	socketMutex.Unlock()
	var wg sync.WaitGroup
	counts := make(map[string]int, len(collectionConfigs))
	for _, config := range collectionConfigs {
		wg.Add(1)
		go func(config collectionConfig,
		) {
			defer wg.Done()
			results, err1 := fetchFromCollection(ctx, c, socketMutex, config)
			if err1 != nil {
				logger.Error().Str("fetching", config.name).Err(err1).Send()
				return
			}
			counts[config.name] = len(results)
			logger.Info().Str("fetching", config.name).Int("count", len(results)).Send()
		}(config)
	}
	wg.Wait()

	socketMutex.Lock()
	if err := c.WriteJSON(fiber.Map{"status": fiber.Map{
		"status":    "done",
		"command":   cmd,
		"siteId":    siteID,
		"channelId": channelID,
		"counts":    counts,
	}}); err != nil {
		logger.Error().Err(err).Msg("WriteJSON error")
	}
	socketMutex.Unlock()
	logger.Info().Msg("Timeline data sent")
}
