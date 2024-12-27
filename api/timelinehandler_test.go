package api_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/contrib/fiberzerolog"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/vtpl1/cacheserver/api"
	"github.com/vtpl1/cacheserver/db"
)

func TestTimeLineHandler(t *testing.T) {
	app := fiber.New()
	ctx := context.Background()
	mongoConnectionString := "mongodb://root:root%40central1234@172.236.106.28:27017/"
	// mongoConnectionString := "mongo"
	mongoClient, err := db.GetMongoClient(ctx, mongoConnectionString)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to MongoDB")
		t.Fatal(err)
	}
	defer mongoClient.Disconnect(ctx) //nolint:errcheck
	app.Use(fiberzerolog.New(fiberzerolog.Config{
		Logger: &log.Logger,
	}))

	app.Get("site/:siteId/channel/:channelId/:timeStamp/:timeStampEnd/timeline/all", api.TimeLineHandler)

	req := httptest.NewRequest("GET", "/site/5/channel/5/1733931560425/1733932680391/timeline/all", nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, 2000)
	if err != nil {
		t.Fatalf("Error during request: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	decoder := json.NewDecoder(resp.Body)
	var timeLineResponse2 api.TimeLineResponse
	if err := decoder.Decode(&timeLineResponse2); err != nil {
		t.Fatalf("Error decoding JSON response: %v", err)
	}
	timeLineResponse1 := api.TimeLineResponse{
		ReturnValue: "SUCCESS",
		Code:        0,
		Status:      200,
		Description: "OK",
		Message:     "Successfully Retrieved!",
		Results: []api.Result{
			{
				Recordings: []api.Recording{
					{
						SiteID:       5,
						ChannelID:    5,
						TimeStamp:    1733931560425,
						TimeStampEnd: 1733932161301,
					},
					{
						SiteID:       5,
						ChannelID:    5,
						TimeStamp:    1733932341866,
						TimeStampEnd: 1733932641866,
					},
					{
						SiteID:       5,
						ChannelID:    5,
						TimeStamp:    1733932680391,
						TimeStampEnd: 1733932980391,
					},
				},
				Events:   []api.Event{},
				Humans:   []api.Human{},
				Vehicles: []api.Vehicle{},
			},
		},
	}

	assert.Equal(t, timeLineResponse1, timeLineResponse2)
}
