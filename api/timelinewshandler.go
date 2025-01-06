package api

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

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
	start := time.Now()
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
	log.Info().Str("command_id", config.commandID).Str("collection", config.collName).Int64("aggregation_time_in_millis", time.Since(start).Milliseconds()).Send()

	defer cursor.Close(ctx) //nolint:errcheck
	var results []interface{}
	var resultsToSend []interface{}

	isStartStatusSent := false
	for cursor.Next(ctx) {
		result := config.resultType
		if err = cursor.Decode(result); err != nil {
			_ = writeResponse(c, socketMutex, config.name, fiber.Map{"commandId": config.commandID, "status": "error"})
			return results, err
		}
		// Ensure deep copy of result
		var copiedResult interface{}
		switch v := result.(type) {
		case *models.Recording:
			copied := *v // Create a new copy of the struct
			copied.CommandID = config.commandID
			copiedResult = &copied
		case *models.Human:
			copied := *v // Create a new copy of the struct
			copied.CommandID = config.commandID
			copiedResult = &copied
		case *models.Vehicle:
			copied := *v // Create a new copy of the struct
			copied.CommandID = config.commandID
			copiedResult = &copied
		case *models.Event:
			copied := *v // Create a new copy of the struct
			copied.CommandID = config.commandID
			copiedResult = &copied
		default:
			copiedResult = result

		}
		resultsToSend = append(resultsToSend, copiedResult)
		if len(resultsToSend) >= 200 {
			if !isStartStatusSent {
				isStartStatusSent = true
				if err = writeResponse(c, socketMutex, config.name, fiber.Map{"commandId": config.commandID, "status": "start"}); err != nil {
					return results, err
				}
				log.Info().Str("command_id", config.commandID).Str("collection", config.collName).Str("sent", "start").Send()
			}

			if err = writeResponse(c, socketMutex, config.name, resultsToSend); err != nil {
				return results, err
			}
			log.Info().Str("command_id", config.commandID).Str("collection", config.collName).Str("sent", "data").Int("count", len(resultsToSend)).Send()

			results = append(results, resultsToSend...)
			resultsToSend = nil
		}
	}
	if len(resultsToSend) > 0 {
		if !isStartStatusSent {
			if err = writeResponse(c, socketMutex, config.name, fiber.Map{"commandId": config.commandID, "status": "start"}); err != nil {
				return results, err
			}
			log.Info().Str("command_id", config.commandID).Str("collection", config.collName).Str("sent", "start").Send()
		}
		if err = writeResponse(c, socketMutex, config.name, resultsToSend); err != nil {
			return results, err
		}
		log.Info().Str("command_id", config.commandID).Str("collection", config.collName).Str("sent_last", "data").Int("count", len(resultsToSend)).Send()

		results = append(results, resultsToSend...)
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
	defer socketMutex.Unlock()
	_ = c.WriteJSON(fiber.Map{"type": "error", "error": err.Error()})
}

func writeResponse(c *websocket.Conn, socketMutex *sync.Mutex, msgKey string, msg interface{}) error {
	socketMutex.Lock()
	defer socketMutex.Unlock()
	err := c.WriteJSON(fiber.Map{"type": msgKey, msgKey: msg})
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
	var ctx1 context.Context

	for {
		var cmd models.Command
		if err = c.ReadJSON(&cmd); err != nil {
			logger.Error().Err(err).Msg("Failed to read websocket command")
			writeErrorResponse(c, &socketMutex, errInvalidCommand)
			break
		}
		logger.Info().Msg("command:" + fmt.Sprint(cmd))

		if cancel != nil {
			logger.Info().Msg("Canceling previous command:" + fmt.Sprint(cmd))
			cancel()
			time.Sleep(100 * time.Millisecond)
			cancel = nil
		}

		ctx1, cancel = context.WithCancel(ctx)
		defer cancel()
		go func() {
			writeResults(ctx1, cmd, c, &socketMutex, siteID, channelID, &logger)
			if cancel != nil {
				cancel()
				cancel = nil
			}
		}()
	}
}

func writeResults(ctx context.Context, cmd models.Command, c *websocket.Conn, socketMutex *sync.Mutex, siteID int, channelID int, logger *zerolog.Logger) {
	// logger.Info().Str("command_id", cmd.CommandID).Str("Entering", "deferred").Send()
	// defer func() {
	// 	logger.Info().Str("command_id", cmd.CommandID).Str("Exiting", "deferred").Send()
	// }()
	if cmd.DomainMax < cmd.DomainMin {
		logger.Error().Err(errInvalidTimeRange)
		writeErrorResponse(c, socketMutex, errInvalidTimeRange)
		return
	}

	var maxTimeGapAllowedInmSec int64
	domainMax := int64(cmd.DomainMax)
	domainMin := int64(cmd.DomainMin)
	diff := domainMax - domainMin
	maxTimeGapAllowedInmSec = diff / 5000
	if maxTimeGapAllowedInmSec <= 0 {
		maxTimeGapAllowedInmSec = 100
	}
	// switch {
	// case diff < maxTimeGapAllowedInmSecForDay:
	// 	maxTimeGapAllowedInmSec = 100
	// case diff < maxTimeGapAllowedInmSecFor15Day:
	// 	maxTimeGapAllowedInmSec = maxTimeGapAllowedInmSecFor10Second
	// case diff < maxTimeGapAllowedInmSecForMonth:
	// 	maxTimeGapAllowedInmSec = maxTimeGapAllowedInmSecForMinute
	// case diff < maxTimeGapAllowedInmSecFor3Months:
	// 	maxTimeGapAllowedInmSec = maxTimeGapAllowedInmSecFor12Minutes
	// case diff < maxTimeGapAllowedInmSecFor6Months:
	// 	maxTimeGapAllowedInmSec = maxTimeGapAllowedInmSecFor24Minutes
	// default:
	// 	maxTimeGapAllowedInmSec = maxTimeGapAllowedInmSecForHour
	// }
	logger.Info().Str("command_id", cmd.CommandID).Int64("max_time_gap_in_ms", maxTimeGapAllowedInmSec).Send()

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

	if err := writeResponse(c, socketMutex, "status", fiber.Map{
		"status":    "start",
		"command":   cmd,
		"siteId":    siteID,
		"channelId": channelID,
	}); err != nil {
		logger.Error().Err(err).Msg("writeResponse error")
	}

	var wg sync.WaitGroup
	counts := make(map[string]int, len(collectionConfigs))
	for _, config := range collectionConfigs {
		wg.Add(1)
		go func(config collectionConfig,
		) {
			defer wg.Done()
			start := time.Now()

			results, err1 := fetchFromCollection(ctx, c, socketMutex, config)
			if err1 != nil {
				logger.Error().Str("command_id", cmd.CommandID).Str("fetching", config.name).Err(err1).Send()
				return
			}
			counts[config.name] = len(results)
			logger.Info().Str("command_id", cmd.CommandID).Str("fetched-sent", config.name).Int("count", len(results)).Int64("time_taken_in_millis", time.Since(start).Milliseconds()).Send()
		}(config)
	}
	wg.Wait()

	if err := writeResponse(c, socketMutex, "status", fiber.Map{
		"status":    "done",
		"command":   cmd,
		"siteId":    siteID,
		"channelId": channelID,
		"counts":    counts,
	}); err != nil {
		logger.Error().Err(err).Msg("writeResponse error")
	}

	logger.Info().Msg("Timeline data sent")
}
