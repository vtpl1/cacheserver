package db_test

import (
	"context"
	"testing"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestMongo(t *testing.T) {
	// Requires the MongoDB Go Driver
	// https://go.mongodb.org/mongo-driver
	ctx := context.TODO()

	// Set client options
	clientOptions := options.Client().ApplyURI("mongodb://root:root%40central1234@172.236.106.28:27017/")

	// Connect to MongoDB
	client, err := mongo.Connect(clientOptions)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			log.Fatal().Err(err).Send()
		}
	}()

	// Open an aggregation cursor
	coll := client.Database("pvaDB").Collection("pva_HUMAN_1_1")
	cursor, err := coll.Aggregate(ctx, bson.A{
		bson.D{
			{
				"$match",
				bson.D{
					{
						"$or",
						bson.A{
							bson.D{
								{
									"startTimestamp",
									bson.D{
										{"$gte", 1732271925859},
										{"$lte", 1732271960058},
									},
								},
							},
							bson.D{
								{
									"endTimestamp",
									bson.D{
										{"$gte", 1732271925859},
										{"$lte", 1732271960058},
									},
								},
							},
							bson.D{
								{
									"$and",
									bson.A{
										bson.D{
											{"startTimestamp", bson.D{{"$lte", 1732271925859}}},
											{"endTimestamp", bson.D{{"$gte", 1732271960058}}},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		bson.D{{"$set", bson.D{{"maxTimeGapAllowed", 3000}}}},
		bson.D{
			{
				"$setWindowFields",
				bson.D{
					{"partitionBy", "channelId"},
					{"sortBy", bson.D{{"startTimestamp", 1}}},
					{
						"output",
						bson.D{
							{
								"prevTimeStamp",
								bson.D{
									{
										"$shift",
										bson.D{
											{"output", "$startTimestamp"},
											{"by", -1},
										},
									},
								},
							},
							{
								"nextTimeStamp",
								bson.D{
									{
										"$shift",
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
			{
				"$set",
				bson.D{
					{
						"prevTimeStampDifference",
						bson.D{
							{
								"$subtract",
								bson.A{
									"$startTimestamp",
									"$prevTimeStamp",
								},
							},
						},
					},
					{
						"nextTimeStampDifference",
						bson.D{
							{
								"$subtract",
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
			{
				"$unset",
				bson.A{
					"prevTimeStamp",
					"nextTimeStamp",
				},
			},
		},
		bson.D{{"$set", bson.D{{"state", true}}}},
		bson.D{
			{
				"$setWindowFields",
				bson.D{
					{"partitionBy", "channelId"},
					{"sortBy", bson.D{{"startTimestamp", 1}}},
					{
						"output",
						bson.D{
							{
								"prevState",
								bson.D{
									{
										"$shift",
										bson.D{
											{"output", "$state"},
											{"by", -1},
											{"default", false},
										},
									},
								},
							},
							{
								"nextState",
								bson.D{
									{
										"$shift",
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
			{
				"$set",
				bson.D{
					{
						"prevState",
						bson.D{
							{
								"$cond",
								bson.A{
									bson.D{
										{
											"$lt",
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
					{
						"nextState",
						bson.D{
							{
								"$cond",
								bson.A{
									bson.D{
										{
											"$lt",
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
			{
				"$set",
				bson.D{
					{
						"startTimestampTemp",
						bson.D{
							{
								"$cond",
								bson.A{
									bson.D{
										{
											"$and",
											bson.A{
												bson.D{
													{
														"$eq",
														bson.A{
															"$prevState",
															false,
														},
													},
												},
												bson.D{
													{
														"$eq",
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
					{
						"endTimestampTemp",
						bson.D{
							{
								"$cond",
								bson.A{
									bson.D{
										{
											"$and",
											bson.A{
												bson.D{
													{
														"$eq",
														bson.A{
															"$state",
															true,
														},
													},
												},
												bson.D{
													{
														"$eq",
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
			{
				"$unset",
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
			{
				"$match",
				bson.D{
					{
						"$or",
						bson.A{
							bson.D{{"startTimestampTemp", bson.D{{"$exists", true}}}},
							bson.D{{"endTimestampTemp", bson.D{{"$exists", true}}}},
						},
					},
				},
			},
		},
		bson.D{
			{
				"$setWindowFields",
				bson.D{
					{"partitionBy", "channelId"},
					{"sortBy", bson.D{{"startTimestamp", 1}}},
					{
						"output",
						bson.D{
							{
								"endTimestamp",
								bson.D{
									{
										"$shift",
										bson.D{
											{"output", "$endTimestampTemp"},
											{"by", 1},
										},
									},
								},
							},
							{
								"startTimestamp",
								bson.D{
									{
										"$shift",
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
			{
				"$match",
				bson.D{
					{
						"$or",
						bson.A{
							bson.D{
								{
									"$and",
									bson.A{
										bson.D{{"startTimestampTemp", bson.D{{"$ne", bson.Null{}}}}},
										bson.D{{"endTimestamp", bson.D{{"$ne", bson.Null{}}}}},
									},
								},
							},
							bson.D{
								{
									"$and",
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
			{
				"$set",
				bson.D{
					{
						"startTimestamp",
						bson.D{
							{
								"$cond",
								bson.A{
									bson.D{
										{
											"$eq",
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
					{
						"endTimestamp",
						bson.D{
							{
								"$cond",
								bson.A{
									bson.D{
										{
											"$eq",
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
			{
				"$set",
				bson.D{
					{
						"timeStampDifference",
						bson.D{
							{
								"$subtract",
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
			{
				"$unset",
				bson.A{
					"startTimestampTemp",
				},
			},
		},
	})

	count := 0
	// Process results
	for cursor.Next(ctx) {
		var result map[string]interface{}
		if err := cursor.Decode(&result); err != nil {
			log.Error().Err(err).Send()
			break
		}
		log.Info().Interface("result", result).Send()
		count++
	}

	if err := cursor.Err(); err != nil {
		t.Fatalf("Cursor error: %v", err)
	}
	assert.Equal(t, count, 2)
}

func TestMongo2(t *testing.T) {
	// Requires the MongoDB Go Driver
	// https://go.mongodb.org/mongo-driver
	ctx := context.TODO()

	// Set client options
	clientOptions := options.Client().ApplyURI("mongodb://root:root%40central1234@172.236.106.28:27017/")

	// Connect to MongoDB
	client, err := mongo.Connect(clientOptions)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			log.Fatal().Err(err).Send()
		}
	}()

	coll := client.Database("pvaDB").Collection("pva_HUMAN_1_1")

	// Define aggregation pipeline
	pipeline := mongo.Pipeline{
		bson.D{
			{"$match", bson.D{
				{"$or", bson.A{
					bson.D{
						{"startTimestamp", bson.D{
							{"$gte", 1732271925859},
							{"$lte", 1732271960058},
						}},
					},
					bson.D{
						{"endTimestamp", bson.D{
							{"$gte", 1732271925859},
							{"$lte", 1732271960058},
						}},
					},
					bson.D{
						{"$and", bson.A{
							bson.D{{"startTimestamp", bson.D{{"$lte", 1732271925859}}}},
							bson.D{{"endTimestamp", bson.D{{"$gte", 1732271960058}}}},
						}},
					},
				}},
			}},
		},
		bson.D{{"$set", bson.D{{"maxTimeGapAllowed", 3000}}}},
		bson.D{
			{"$set", bson.D{
				{"prevTimeStampDifference", bson.D{
					{"$subtract", bson.A{"$startTimestamp", "$prevTimeStamp"}},
				}},
				{"nextTimeStampDifference", bson.D{
					{"$subtract", bson.A{"$nextTimeStamp", "$startTimestamp"}},
				}},
			}},
		},
		bson.D{{"$unset", bson.A{"prevTimeStamp", "nextTimeStamp"}}},
		bson.D{{"$set", bson.D{{"state", true}}}},
		bson.D{
			{"$setWindowFields", bson.D{
				{"partitionBy", "channelId"},
				{"sortBy", bson.D{{"startTimestamp", 1}}},
				{"output", bson.D{
					{"prevState", bson.D{
						{"$shift", bson.D{
							{"output", "$state"},
							{"by", -1},
							{"default", false},
						}},
					}},
					{"nextState", bson.D{
						{"$shift", bson.D{
							{"output", "$state"},
							{"by", 1},
							{"default", false},
						}},
					}},
				}},
			}},
		},
		bson.D{{"$unset", bson.A{"state", "nextState", "prevState"}}},
	}

	// Execute the aggregation
	cursor, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		t.Fatalf("Aggregation failed: %v", err)
	}
	defer cursor.Close(ctx)
	count := 0
	// Process results
	for cursor.Next(ctx) {
		var result map[string]interface{}
		if err := cursor.Decode(&result); err != nil {
			log.Error().Err(err).Send()
			break
		}
		log.Info().Interface("result", result).Send()
		count++
	}

	if err := cursor.Err(); err != nil {
		t.Fatalf("Cursor error: %v", err)
	}
	assert.Equal(t, count, 2)
}

func TestMongo3(t *testing.T) {
	// Requires the MongoDB Go Driver
	// https://go.mongodb.org/mongo-driver
	ctx := context.TODO()

	// Set client options
	clientOptions := options.Client().ApplyURI("mongodb://root:root%40central1234@172.236.106.28:27017/")

	// Connect to MongoDB
	client, err := mongo.Connect(clientOptions)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			log.Fatal().Err(err).Send()
		}
	}()

	coll := client.Database("pvaDB").Collection("pva_HUMAN_1_1")
	domainMin := 1732271925860
	domainMax := 1732272027958
	maxTimeGapAllowedInmSec := 15000
	// Define aggregation pipeline
	pipeline := bson.A{
		bson.D{
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
		},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "startTimestamp", Value: 1}}}},
		bson.D{
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
		},
		bson.D{
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
		},
		bson.D{
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
		},
		bson.D{
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
		},
		bson.D{
			{
				Key: "$group",
				Value: bson.D{
					{Key: "_id", Value: "$groupId"},
					{Key: "startTimestamp", Value: bson.D{{Key: "$first", Value: "$startTimestamp"}}},
					{Key: "endTimestamp", Value: bson.D{{Key: "$last", Value: "$endTimestamp"}}},
					{Key: "objectCount", Value: bson.D{{Key: "$sum", Value: "$objectCount"}}},
				},
			},
		},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "startTimestamp", Value: 1}}}},
	}

	// Execute the aggregation
	cursor, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		t.Fatalf("Aggregation failed: %v", err)
	}
	defer cursor.Close(ctx)
	count := 0
	// Process results
	for cursor.Next(ctx) {
		var result map[string]interface{}
		if err := cursor.Decode(&result); err != nil {
			log.Error().Err(err).Send()
			break
		}
		log.Info().Interface("result", result).Send()
		count++
	}

	if err := cursor.Err(); err != nil {
		t.Fatalf("Cursor error: %v", err)
	}
	assert.Equal(t, count, 2)
}
