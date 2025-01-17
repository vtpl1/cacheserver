// package main is the main entrypoint for the application.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/gofiber/contrib/fiberzerolog"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/natefinch/lumberjack"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
	"github.com/vtpl1/cacheserver/api"
	"github.com/vtpl1/cacheserver/db"
)

func getFolder(s string) string {
	err := os.MkdirAll(s, 0o750)
	if err != nil {
		fmt.Printf("Unable to create folder %s, %v", s, err)
	}
	return s
}

func getApplicationName() string {
	return "cache-server"
}

func getSessionFolder() string { //nolint:unused
	return getFolder(filepath.Join("session", getApplicationName()))
}

func getLogFolder() string {
	return getFolder(filepath.Join("logs", getApplicationName()))
}

func getConfigFilePath() string { //nolint:unused
	suggestedConfigFile := filepath.ToSlash(fmt.Sprintf("%s/%s.yaml",
		getSessionFolder(),
		getApplicationName()))

	return suggestedConfigFile
}

// GitCommit hash set by the Go linker
var (
	GitCommit string //nolint:gochecknoglobals
	// BuildTime set by the Go linker
	BuildTime string //nolint:gochecknoglobals
)

func getVersion() string {
	if GitCommit != "" && BuildTime != "" {
		return GitCommit + " " + BuildTime
	}
	GitCommit = "unknown"
	BuildTime = "unknown"

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return GitCommit
	}
	modified := false

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			GitCommit = setting.Value
		case "vcs.time":
			BuildTime = setting.Value
		case "vcs.modified":
			modified = true
		}
	}
	if modified {
		GitCommit += "+CHANGES"
	}

	return GitCommit + " " + BuildTime
}

// ws://localhost:8080/ws/timeline/site/1/channel/1
// ./cacheserver --mongo-connection-string mongodb://root:root%40central1234@45.33.125.226:27017/ --host 0.0.0.0
// ./cacheserver --mongo-connection-string mongodb://root:root%40central1234@172.236.106.28:27017/ --host 0.0.0.0

func main() {
	// Create the CLI application

	cmd := &cli.Command{
		EnableShellCompletion: true,
		Name:                  "cache-server",
		Version:               getVersion(),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "host",
				Value: "0.0.0.0",
				Usage: "The host address for the server",
			},
			&cli.IntFlag{
				Name:    "port",
				Value:   8084,
				Usage:   "The port number for the server",
				Sources: cli.EnvVars("PORT"),
			},
			&cli.StringFlag{
				Name:    "mongo-connection-string",
				Value:   "mongodb://restheart:R3ste4rt%21@127.0.0.1:27017/",
				Usage:   "The connection string for the MongoDB server",
				Sources: cli.EnvVars("MONGO_CONNECTION_STRING"),
			},
			&cli.StringFlag{
				Name:  "logfile",
				Value: fmt.Sprintf("%s.log", filepath.Join(getLogFolder(), getApplicationName())),
				Usage: "The log file path for the rotating logger",
			},
			&cli.StringFlag{
				Name:  "logLevel",
				Value: "debug",
				Usage: "The log level",
			},
		},
		Action: startServer,
	}

	// Run the CLI application
	err := cmd.Run(context.Background(), os.Args)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	fmt.Println("Server shut down gracefully 2")
}

func startServer(ctx context.Context, cmd *cli.Command) error {
	// Initialize logging
	bufferWriter, err := initLogger(cmd.String("logfile"), cmd.String("logLevel"))
	if err != nil {
		return err
	}
	defer bufferWriter.Close() //nolint:errcheck
	host := cmd.String("host")
	port := cmd.Int("port")
	address := fmt.Sprintf("%s:%d", host, port)

	mongoConnectionString := cmd.String("mongo-connection-string")
	mongoClient, err := db.GetMongoClient(ctx, mongoConnectionString)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to MongoDB")
		return err
	}
	defer mongoClient.Disconnect(ctx) //nolint:errcheck

	// Configure the HTTP app with timeouts
	app := fiber.New(fiber.Config{
		// Prefork:       true,
		// CaseSensitive: true,
		// StrictRouting: true,
		ServerHeader: "Videonetics",
		AppName:      fmt.Sprintf("CacheServer %v", getVersion()),
	})

	app.Use(fiberzerolog.New(fiberzerolog.Config{
		Logger: &log.Logger,
	}))

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*", // Allow all origins
	}))

	app.Use("/ws", func(c *fiber.Ctx) error {
		// IsWebSocketUpgrade returns true if the client
		// requested upgrade to the WebSocket protocol.
		log.Info().Msg("Server ws")
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			c.Locals("ctx", c.Context())
			return c.Next()
		}
		log.Info().Msg("Server ErrUpgradeRequired")
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

	app.Get("site/:siteId/channel/:channelId/:timeStamp/:timeStampEnd/timeline/all", api.TimeLineHandler)

	// Start the server in a goroutine
	go func() {
		log.Info().Msgf("Starting server at %s", address)
		if err := app.Listen(address); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("Server failed to start")
		}
	}()
	waitForTerminationRequest()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	log.Info().Msg("Starting shutdown")
	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Error().Err(err).Msg("Error during server shutdown")
	}
	log.Info().Msg("Server shut down gracefully")
	fmt.Println("Server shut down gracefully")
	return nil
}

// gracefulShutdown handles termination signals to gracefully shut down the server.
func waitForTerminationRequest() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Info().Msg("Shutting down server...")
}

// initLogger initializes the logger with zerolog, diode, and a rotating logger.
func initLogger(logFile string, logLevel string) (diode.Writer, error) {
	// Configure Lumberjack for log rotation
	rotatingLogger := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    10,   // Max size in MB before rotation
		MaxBackups: 3,    // Max number of old log files to keep
		MaxAge:     28,   // Max number of days to retain old log files
		Compress:   true, // Compress rotated files
	}

	// Wrap Lumberjack with Diode for non-blocking logging
	bufferedWriter := diode.NewWriter(rotatingLogger, 1000, 0, func(missed int) {
		fmt.Printf("Dropped %d log messages due to buffer overflow\n", missed)
	})

	// Set up Zerolog with the buffered writer
	log.Logger = zerolog.New(bufferedWriter).With().Timestamp().Logger()

	// Set log level
	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		fmt.Printf("Invalid log level: %s\n", logLevel)
		return bufferedWriter, err
	}
	zerolog.SetGlobalLevel(level)

	// Log application startup
	log.Info().Msgf("App started %s %s", getApplicationName(), getVersion())
	return bufferedWriter, nil
}
