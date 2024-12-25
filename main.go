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
	"github.com/gofiber/fiber/v2"
	"github.com/natefinch/lumberjack"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
	"github.com/vtpl1/cacheserver/api"
	"github.com/vtpl1/cacheserver/db"
)

func getFolder(s string) string {
	err := os.MkdirAll(s, os.ModePerm)
	if err != nil {
		fmt.Printf("Unable to create folder %s, %v", s, err)
	}
	return s
}

func getApplicationName() string {
	return "cache-server"
}

func getSessionFolder() string {
	return getFolder(filepath.Join("session", getApplicationName()))
}

func getLogFolder() string {
	return getFolder(filepath.Join("logs", getApplicationName()))
}

func getConfigFilePath() string {
	suggestedConfigFile := filepath.ToSlash(fmt.Sprintf("%s/%s.yaml",
		getSessionFolder(),
		getApplicationName()))

	return suggestedConfigFile
}

var GitCommit string

func getVersion() string {
	if GitCommit != "" {
		return GitCommit
	}
	GitCommit = "unknown"
	buildDate := ""

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
			buildDate = setting.Value
		case "vcs.modified":
			modified = true
		}
	}
	if modified {
		GitCommit += "+CHANGES"
	}
	if buildDate != "" {
		GitCommit += " " + buildDate
	}
	return GitCommit
}

func main() {
	// Create the CLI application

	cmd := &cli.Command{
		EnableShellCompletion: true,
		Name:                  "cache-server",
		Version:               getVersion(),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "host",
				Value: "127.0.0.1",
				Usage: "The host address for the server",
			},
			&cli.IntFlag{
				Name:    "port",
				Value:   8080,
				Usage:   "The port number for the server",
				Sources: cli.EnvVars("PORT"),
			},
			&cli.StringFlag{
				Name:    "mongo-connection-string",
				Value:   "mongodb://root:root%40central1234@172.236.106.28:27017/",
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
	err, bufferWriter := initLogger(cmd.String("logfile"), cmd.String("logLevel"))
	if err != nil {
		return err
	}
	defer bufferWriter.Close()
	host := cmd.String("host")
	port := cmd.Int("port")
	address := fmt.Sprintf("%s:%d", host, port)

	// Start the HTTP server
	log.Info().Msgf("Starting server at %s", address)
	mongoConnectionString := cmd.String("mongo-connection-string")
	mongoClient, err := db.GetMongoClient(mongoConnectionString)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to MongoDB")
	}
	defer mongoClient.Disconnect(ctx)

	// Configure the HTTP app with timeouts
	app := fiber.New(fiber.Config{
		// Prefork:       true,
		// CaseSensitive: true,
		// StrictRouting: true,
		ServerHeader: "Videonetics",
		AppName:      fmt.Sprintf("Cache Server %v", getVersion()),
	})

	app.Use(fiberzerolog.New(fiberzerolog.Config{
		Logger: &log.Logger,
	}))
	// app.Use(func(c *fiber.Ctx) error {
	// 	c.Locals("mongo-connection-string", mongoConnectionString)
	// 	return c.Next()
	// })
	app.Get("site/:siteId/channel/:channelId/:timeStamp/:timeStampEnd/timeline", api.TimeLineHandler)

	// Start the server in a goroutine
	go func() {
		log.Info().Msgf("Starting server at %s", address)
		if err := app.Listen(fmt.Sprintf("%s:%d", host, port)); err != nil && !errors.Is(err, http.ErrServerClosed) {
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

func handleRequest(w http.ResponseWriter, r *http.Request) {
	log.Info().Msgf("Received request: %s %s", r.Method, r.URL.Path)
	fmt.Fprintln(w, "Hello, world!")
}

// gracefulShutdown handles termination signals to gracefully shut down the server.
func waitForTerminationRequest() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Info().Msg("Shutting down server...")
}

// initLogger initializes the logger with zerolog, diode, and a rotating logger.
func initLogger(logFile string, logLevel string) (error, diode.Writer) {
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
		return err, bufferedWriter
	}
	zerolog.SetGlobalLevel(level)

	// Log application startup
	log.Info().Msgf("App started %s %s", getApplicationName(), getVersion())
	return nil, bufferedWriter
}
