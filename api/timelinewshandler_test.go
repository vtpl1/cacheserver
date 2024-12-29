package api_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/vtpl1/cacheserver/api"
	"github.com/vtpl1/cacheserver/db"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TestTimeLineWSHandler_Success(t *testing.T) {
	app := fiber.New()
	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		api.TimeLineWSHandler(context.Background(), c)
	}))

	// Mock MongoDB and other dependencies
	db.GetDefaultMongoClient = func() (*mongo.Client, error) {
		return &mongo.Client{}, nil // Replace with a real mock client
	}

	mockCmd := api.Command{
		DomainMin: 1000,
		DomainMax: 2000,
	}
	mockData, _ := json.Marshal(mockCmd)

	server := httptest.NewServer(app)
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws://"+server.Listener.Addr().String()+"/ws", nil)
	assert.NoError(t, err)
	defer conn.Close()

	// Send a valid command
	err = conn.WriteMessage(websocket.TextMessage, mockData)
	assert.NoError(t, err)

	// Read response
	_, resp, err := conn.ReadMessage()
	assert.NoError(t, err)
	assert.Contains(t, string(resp), `"status":"done"`)
}

func TestTimeLineWSHandler_InvalidParams(t *testing.T) {
	// Similar to the success test, but send invalid commands and assert error responses
}

// Additional tests for error handling and concurrency
