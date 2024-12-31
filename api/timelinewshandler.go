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
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	maxTimeGapAllowedInmSecForMinute  = 60000
	maxTimeGapAllowedInmSecForHour    = 60 * maxTimeGapAllowedInmSecForMinute
	maxTimeGapAllowedInmSecForDay     = 24 * maxTimeGapAllowedInmSecForHour
	maxTimeGapAllowedInmSecFor3Days   = 3 * maxTimeGapAllowedInmSecForDay
	maxTimeGapAllowedInmSecForWeek    = 7 * maxTimeGapAllowedInmSecForDay
	maxTimeGapAllowedInmSecForMonth   = 30 * maxTimeGapAllowedInmSecForDay
	maxTimeGapAllowedInmSecFor3Months = 3 * maxTimeGapAllowedInmSecForMonth
	maxTimeGapAllowedInmSecFor6Months = 3 * maxTimeGapAllowedInmSecForMonth
	maxTimeGapAllowedInmSecForYear    = 12 * maxTimeGapAllowedInmSecForMonth
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
			if cancel != nil {
				cancel()
			}
			break
		}
		if cancel != nil {
			cancel()
		}
		var ctx1 context.Context
		ctx1, cancel = context.WithCancel(ctx)
		go writeResults(ctx1, cmd, c, &socketMutex, siteID, channelID, &logger)
	}
}

func writeResults(ctx context.Context, cmd Command, c *websocket.Conn, socketMutex *sync.Mutex, siteID int, channelID int, logger *zerolog.Logger) {
	if cmd.DomainMax < cmd.DomainMin {
		logger.Error().Err(errInvalidTimeRange)
		writeErrorResponse(c, socketMutex, errInvalidTimeRange)
		return
	}

	// filter := bson.M{"$or": []bson.M{
	// 	{"startTimestamp": bson.M{"$gte": cmd.DomainMin, "$lte": cmd.DomainMax}},
	// 	{"endTimestamp": bson.M{"$gte": cmd.DomainMin, "$lte": cmd.DomainMax}},
	// 	{"$and": []bson.M{
	// 		{"startTimestamp": bson.M{"$lte": cmd.DomainMin}},
	// 		{"endTimestamp": bson.M{"$gte": cmd.DomainMax}},
	// 	}},
	// }}
	var maxTimeGapAllowedInmSec int
	domainMax := cmd.DomainMax
	domainMin := cmd.DomainMin
	diff := domainMax - domainMin
	switch {
	case diff < maxTimeGapAllowedInmSecForMinute:
		maxTimeGapAllowedInmSec = 100
	case diff < maxTimeGapAllowedInmSecForHour:
		maxTimeGapAllowedInmSec = maxTimeGapAllowedInmSecForMinute
	case diff < maxTimeGapAllowedInmSecForDay:
		maxTimeGapAllowedInmSec = maxTimeGapAllowedInmSecForMinute
	case diff < maxTimeGapAllowedInmSecForWeek:
		maxTimeGapAllowedInmSec = maxTimeGapAllowedInmSecForHour
	case diff < maxTimeGapAllowedInmSecFor6Months:
		maxTimeGapAllowedInmSec = maxTimeGapAllowedInmSec
	case diff < maxTimeGapAllowedInmSecForYear:
		maxTimeGapAllowedInmSec = maxTimeGapAllowedInmSecFor3Days
	default:
		maxTimeGapAllowedInmSec = maxTimeGapAllowedInmSecForYear
	}
	logger.Info().Int("maxTimeGapAllowedInmSec", maxTimeGapAllowedInmSec).Send()
	filter := bson.A{
		bson.D{
			{"$match",
				bson.D{
					{"$or",
						bson.A{
							bson.D{
								{"startTimestamp",
									bson.D{
										{"$gte", domainMin},
										{"$lte", domainMax},
									},
								},
							},
							bson.D{
								{"endTimestamp",
									bson.D{
										{"$gte", domainMin},
										{"$lte", domainMax},
									},
								},
							},
							bson.D{
								{"$and",
									bson.A{
										bson.D{
											{"startTimestamp", bson.D{{"$lte", domainMin}}},
											{"endTimestamp", bson.D{{"$gte", domainMax}}},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		bson.D{{"$set", bson.D{{"maxTimeGapAllowed", maxTimeGapAllowedInmSec}}}},
		bson.D{
			{"$setWindowFields",
				bson.D{
					{"partitionBy", "channelId"},
					{"sortBy", bson.D{{"startTimestamp", 1}}},
					{"output",
						bson.D{
							{"prevTimeStamp",
								bson.D{
									{"$shift",
										bson.D{
											{"output", "$startTimestamp"},
											{"by", -1},
										},
									},
								},
							},
							{"nextTimeStamp",
								bson.D{
									{"$shift",
										bson.D{
											{"output", "$startTimestamp"},
											{"by", 1},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		bson.D{
			{"$set",
				bson.D{
					{"prevTimeStampDifference",
						bson.D{
							{"$subtract",
								bson.A{
									"$startTimestamp",
									"$prevTimeStamp",
								},
							},
						},
					},
					{"nextTimeStampDifference",
						bson.D{
							{"$subtract",
								bson.A{
									"$nextTimeStamp",
									"$startTimestamp",
								},
							},
						},
					},
				},
			},
		},
		bson.D{
			{"$unset",
				bson.A{
					"prevTimeStamp",
					"nextTimeStamp",
				},
			},
		},
		bson.D{{"$set", bson.D{{"state", true}}}},
		bson.D{
			{"$setWindowFields",
				bson.D{
					{"partitionBy", "channelId"},
					{"sortBy", bson.D{{"startTimestamp", 1}}},
					{"output",
						bson.D{
							{"prevState",
								bson.D{
									{"$shift",
										bson.D{
											{"output", "$state"},
											{"by", -1},
											{"default", false},
										},
									},
								},
							},
							{"nextState",
								bson.D{
									{"$shift",
										bson.D{
											{"output", "$state"},
											{"by", 1},
											{"default", false},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		bson.D{
			{"$set",
				bson.D{
					{"prevState",
						bson.D{
							{"$cond",
								bson.A{
									bson.D{
										{"$lt",
											bson.A{
												"$prevTimeStampDifference",
												"$maxTimeGapAllowed",
											},
										},
									},
									"$prevState",
									false,
								},
							},
						},
					},
					{"nextState",
						bson.D{
							{"$cond",
								bson.A{
									bson.D{
										{"$lt",
											bson.A{
												"$nextTimeStampDifference",
												"$maxTimeGapAllowed",
											},
										},
									},
									"$nextState",
									false,
								},
							},
						},
					},
				},
			},
		},
		bson.D{
			{"$set",
				bson.D{
					{"startTimestampTemp",
						bson.D{
							{"$cond",
								bson.A{
									bson.D{
										{"$and",
											bson.A{
												bson.D{
													{"$eq",
														bson.A{
															"$prevState",
															false,
														},
													},
												},
												bson.D{
													{"$eq",
														bson.A{
															"$state",
															true,
														},
													},
												},
											},
										},
									},
									"$startTimestamp",
									"$$REMOVE",
								},
							},
						},
					},
					{"endTimestampTemp",
						bson.D{
							{"$cond",
								bson.A{
									bson.D{
										{"$and",
											bson.A{
												bson.D{
													{"$eq",
														bson.A{
															"$state",
															true,
														},
													},
												},
												bson.D{
													{"$eq",
														bson.A{
															"$nextState",
															false,
														},
													},
												},
											},
										},
									},
									"$endTimestamp",
									"$$REMOVE",
								},
							},
						},
					},
				},
			},
		},
		bson.D{
			{"$unset",
				bson.A{
					"state",
					"nextState",
					"prevState",
					"prevTimeStampDifference",
					"nextTimeStampDifference",
				},
			},
		},
		bson.D{
			{"$match",
				bson.D{
					{"$or",
						bson.A{
							bson.D{{"startTimestampTemp", bson.D{{"$exists", true}}}},
							bson.D{{"endTimestampTemp", bson.D{{"$exists", true}}}},
						},
					},
				},
			},
		},
		bson.D{
			{"$setWindowFields",
				bson.D{
					{"partitionBy", "channelId"},
					{"sortBy", bson.D{{"startTimestamp", 1}}},
					{"output",
						bson.D{
							{"endTimestamp",
								bson.D{
									{"$shift",
										bson.D{
											{"output", "$endTimestampTemp"},
											{"by", 1},
										},
									},
								},
							},
							{"startTimestamp",
								bson.D{
									{"$shift",
										bson.D{
											{"output", "$startTimestampTemp"},
											{"by", -1},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		bson.D{
			{"$match",
				bson.D{
					{"$or",
						bson.A{
							bson.D{
								{"$and",
									bson.A{
										bson.D{{"startTimestampTemp", bson.D{{"$ne", bson.Null{}}}}},
										bson.D{{"endTimestamp", bson.D{{"$ne", bson.Null{}}}}},
									},
								},
							},
							bson.D{
								{"$and",
									bson.A{
										bson.D{{"startTimestampTemp", bson.D{{"$ne", bson.Null{}}}}},
										bson.D{{"endTimestampTemp", bson.D{{"$ne", bson.Null{}}}}},
									},
								},
							},
						},
					},
				},
			},
		},
		bson.D{
			{"$set",
				bson.D{
					{"startTimestamp",
						bson.D{
							{"$cond",
								bson.A{
									bson.D{
										{"$eq",
											bson.A{
												"$startTimestamp",
												bson.Null{},
											},
										},
									},
									"$startTimestampTemp",
									"$startTimestamp",
								},
							},
						},
					},
					{"endTimestamp",
						bson.D{
							{"$cond",
								bson.A{
									bson.D{
										{"$eq",
											bson.A{
												"$endTimestamp",
												bson.Null{},
											},
										},
									},
									"$endTimestampTemp",
									"$endTimestamp",
								},
							},
						},
					},
				},
			},
		},
		bson.D{
			{"$set",
				bson.D{
					{"timeStampDifference",
						bson.D{
							{"$subtract",
								bson.A{
									"$endTimestamp",
									"$startTimestamp",
								},
							},
						},
					},
				},
			},
		},
		bson.D{
			{"$unset",
				bson.A{
					"startTimestampTemp",
				},
			},
		},
	}
	// filterWithSiteIDChannelID := bson.M{"siteId": siteID, "channelId": channelID}

	// // Copy from the original map to the target map
	// for key, value := range filter {
	// 	filterWithSiteIDChannelID[key] = value
	// }

	collectionConfigs := []collectionConfig{
		{"recordings", "ivms_30", fmt.Sprintf("vVideoClips_%d_%d", siteID, channelID), &Recording{}, filter},
		{"humans", "pvaDB", fmt.Sprintf("pva_HUMAN_%d_%d", siteID, channelID), &Human{}, filter},
		{"vehicles", "pvaDB", fmt.Sprintf("pva_VEHICLE_%d_%d", siteID, channelID), &Vehicle{}, filter},
		// {"events", "dasDB", "dasEvents", &Event{}, filterWithSiteIDChannelID},
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
	if err := c.WriteJSON(fiber.Map{"status": fiber.Map{
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
