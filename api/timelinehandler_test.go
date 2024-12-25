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
	mongoClient, err := db.GetMongoClient(mongoConnectionString)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to MongoDB")
		t.Fatal(err)
	}
	defer mongoClient.Disconnect(ctx)
	app.Use(fiberzerolog.New(fiberzerolog.Config{
		Logger: &log.Logger,
	}))

	app.Get("site/:siteId/channel/:channelId/:timeStamp/:timeStampEnd/timeline", api.TimeLineHandler)

	req := httptest.NewRequest("GET", "/site/5/channel/5/1733931560425/1733932680391/timeline", nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	jsonBytes, err := json.Marshal(resp.Body)
	assert.NoError(t, err)
	var out api.TimeLineResponse
	err = json.Unmarshal(jsonBytes, &out)
	assert.NoError(t, err)
	timeLineResponse := api.NewTimeLineResponse()
	assert.Equal(t, timeLineResponse, out)

}
