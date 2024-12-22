package cacheserver

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/natefinch/lumberjack"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	logLevelPath = "./loglevel.conf" // Path to the log level configuration file
	upgrader     = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func main() {
	app := &cli.App{
		Name:  "cache-server-cli",
		Usage: "Start the cache server with WebSocket and MongoDB streaming",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "host",
				Value: "127.0.0.1",
				Usage: "The host address for the server",
			},
			&cli.IntFlag{
				Name:  "port",
				Value: 8080,
				Usage: "The port number for the server",
			},
			&cli.StringFlag{
				Name:  "logfile",
				Value: fmt.Sprintf("%s.log", filepath.Join(getLogFolder(), getApplicationName())),
				Usage: "The log file path for the rotating logger",
			},
			&cli.StringFlag{
				Name:  "logLevel",
				Value: "debug",
				Usage: "The initial log level",
			},
			&cli.StringFlag{
				Name:  "mongoUri",
				Value: "mongodb://localhost:27017",
				Usage: "The MongoDB connection URI",
			},
			&cli.StringFlag{
				Name:  "database",
				Value: "test",
				Usage: "The MongoDB database name",
			},
			&cli.StringFlag{
				Name:  "collection",
				Value: "example",
				Usage: "The MongoDB collection name",
			},
		},
		Action: func(c *cli.Context) error {
			initLogger(c.String("logfile"), c.String("logLevel"))

			// Start monitoring log level changes
			go monitorLogLevel()

			// MongoDB connection
			client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(c.String("mongoUri")))
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to connect to MongoDB")
			}
			defer client.Disconnect(context.Background())

			db := client.Database(c.String("database"))
			collection := db.Collection(c.String("collection"))

			host := c.String("host")
			port := c.Int("port")
			address := fmt.Sprintf("%s:%d", host, port)

			http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
				handleWebSocket(w, r, collection)
			})

			server := &http.Server{
				Addr:         address,
				ReadTimeout:  10 * time.Second,
				WriteTimeout: 10 * time.Second,
				IdleTimeout:  120 * time.Second,
			}

			go func() {
				log.Info().Msgf("Starting server at %s", address)
				if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					log.Fatal().Err(err).Msg("Server failed to start")
				}
			}()

			waitForTerminationRequest()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := server.Shutdown(ctx); err != nil {
				log.Error().Err(err).Msg("Error during server shutdown")
			} else {
				log.Info().Msg("Server shut down gracefully")
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Send()
	}
}

func monitorLogLevel() {
	ticker := time.NewTicker(10 * time.Second) // Periodically check for log level changes
	defer ticker.Stop()

	for range ticker.C {
		content, err := ioutil.ReadFile(logLevelPath)
		if err != nil {
			log.Warn().Msgf("Could not read log level file: %v", err)
			continue
		}

		newLevel := strings.TrimSpace(string(content))
		level, err := zerolog.ParseLevel(newLevel)
		if err != nil {
			log.Warn().Msgf("Invalid log level in file: %s", newLevel)
			continue
		}

		currentLevel := zerolog.GlobalLevel()
		if level != currentLevel {
			zerolog.SetGlobalLevel(level)
			log.Info().Msgf("Log level updated to: %s", level.String())
		}
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request, collection *mongo.Collection) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to upgrade to WebSocket")
		return
	}
	defer conn.Close()

	ctx := context.Background()
	pipeline := mongo.Pipeline{}
	changeStream, err := collection.Watch(ctx, pipeline, options.ChangeStream().SetFullDocument(options.UpdateLookup))
	if err != nil {
		log.Error().Err(err).Msg("Failed to start change stream")
		return
	}
	defer changeStream.Close(ctx)

	log.Info().Msg("WebSocket connection established")

	for changeStream.Next(ctx) {
		var event map[string]interface{}
		if err := changeStream.Decode(&event); err != nil {
			log.Error().Err(err).Msg("Failed to decode change stream event")
			continue
		}

		if err := conn.WriteJSON(event); err != nil {
			log.Error().Err(err).Msg("Failed to send data via WebSocket")
			break
		}
	}

	if err := changeStream.Err(); err != nil {
		log.Error().Err(err).Msg("Change stream encountered an error")
	}
}

func waitForTerminationRequest() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Info().Msg("Shutting down server...")
}

func initLogger(logFile string, logLevel string) {
	rotatingLogger := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    10,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}

	bufferedWriter := diode.NewWriter(rotatingLogger, 1000, 0, func(missed int) {
		fmt.Printf("Dropped %d log messages due to buffer overflow\n", missed)
	})
	log.Logger = zerolog.New(bufferedWriter).With().Timestamp().Logger()

	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		fmt.Printf("Invalid log level: %s\n", logLevel)
		os.Exit(1)
	}
	zerolog.SetGlobalLevel(level)

	log.Info().Msg("Logger initialized with file rotation and diode buffering")
}

func getApplicationName() string {
	return "cache-server"
}

func getLogFolder() string {
	return getFolder(filepath.Join("logs", getApplicationName()))
}

func getFolder(s string) string {
	err := os.MkdirAll(s, os.ModePerm)
	if err != nil {
		fmt.Printf("Unable to create folder %s: %v\n", s, err)
	}
	return s
}
