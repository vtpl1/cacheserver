package api_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	fasthttp_websocket "github.com/fasthttp/websocket"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/vtpl1/cacheserver/api"
	"github.com/vtpl1/cacheserver/db"
	"github.com/vtpl1/cacheserver/models"
)

func setupTimeLineWSHandlerApp() *fiber.App {
	ctx := context.Background()
	mongoConnectionString := "mongodb://root:root%40central1234@172.236.106.28:27017/"
	// mongoConnectionString := "mongo"
	db.GetMongoClient(ctx, mongoConnectionString)
	app := fiber.New(fiber.Config{DisableStartupMessage: true})

	app.Use("/ws", func(c *fiber.Ctx) error {
		// IsWebSocketUpgrade returns true if the client
		// requested upgrade to the WebSocket protocol.
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			c.Locals("ctx", c.Context())
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Use("/ws/timeline/site/:siteId/channel/:channelId", websocket.New(func(c *websocket.Conn) {
		ctx1, ok := c.Locals("ctx").(context.Context) // Pass context from Fiber request
		if !ok {
			log.Error().Msg("Context does not exists")
			return
		}
		api.TimeLineWSHandler(ctx1, c)
	}))

	address := "localhost:3000"
	// Start the server in a goroutine
	go func() {
		log.Info().Msgf("Starting server at %s", address)
		if err := app.Listen(address); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("Server failed to start")
		}
	}()

	readyCh := make(chan struct{})

	go func() {
		for {
			conn, err := net.Dial("tcp", "localhost:3000")
			if err != nil {
				continue
			}

			if conn != nil {
				readyCh <- struct{}{}
				conn.Close()
				break
			}
		}
	}()

	<-readyCh

	return app
}

func TestTimeLineWSHandlerWithoutSiteIdChannelId(t *testing.T) {
	app := setupTimeLineWSHandlerApp()
	defer app.Shutdown()
	{
		conn, resp, _ := fasthttp_websocket.DefaultDialer.Dial("ws://localhost:3000/ws/timeline", nil)
		defer resp.Body.Close()
		defer conn.Close()
		// require.ErrorAs(t, err, websocket.ErrBadHandshake)

		assert.Equal(t, 404, resp.StatusCode)
	}
	{
		conn, resp, err := fasthttp_websocket.DefaultDialer.Dial("ws://localhost:3000/ws/timeline/site/1/channel/1", nil)
		defer resp.Body.Close()
		defer conn.Close()
		assert.NoError(t, err)

		assert.Equal(t, 101, resp.StatusCode)
		assert.Equal(t, "websocket", resp.Header.Get("Upgrade"))
	}

	// var msg fiber.Map
	// err = conn.ReadJSON(&msg)
	// assert.NoError(t, err)
	// assert.Equal(t, "hello websocket", msg["message"])
}

func TestTimeLineWSHandler_Success(t *testing.T) {
	app := setupTimeLineWSHandlerApp()
	defer app.Shutdown()

	conn, resp, err := fasthttp_websocket.DefaultDialer.Dial("ws://localhost:3000/ws/timeline/site/1/channel/1", nil)
	defer resp.Body.Close()
	defer conn.Close()
	assert.NoError(t, err)

	assert.Equal(t, 101, resp.StatusCode)
	assert.Equal(t, "websocket", resp.Header.Get("Upgrade"))
	err = conn.WriteJSON(models.Command{
		CommandID:  "222",
		PivotPoint: 0,
		DisplayMin: 0,
		DomainMin:  1732271925859,
		DomainMax:  1735717489000,
	})
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)
	err = conn.WriteJSON(models.Command{
		CommandID:  "333",
		PivotPoint: 0,
		DisplayMin: 0,
		DomainMin:  1732271925859,
		DomainMax:  1735717489000,
	})
	assert.NoError(t, err)
	var v interface{}

	err = conn.ReadJSON(v)
	assert.NoError(t, err)
	time.Sleep(10 * time.Second)
}

func TestTimeLineWSHandler_InvalidParams(t *testing.T) {
	// Similar to the success test, but send invalid commands and assert error responses
}

// Additional tests for error handling and concurrency
