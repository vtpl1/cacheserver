package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/natefinch/lumberjack"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
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

func main() {
	// Create the CLI application

	cmd := &cli.Command{
		EnableShellCompletion: true,
		Name:                  "cache-server-cli",
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
		Action: func(ctx context.Context, cmd *cli.Command) error {
			// Initialize logging
			initLogger(cmd.String("logfile"), cmd.String("logLevel"))
			host := cmd.String("host")
			port := cmd.Int("port")
			address := fmt.Sprintf("%s:%d", host, port)

			// Start the HTTP server
			log.Info().Msgf("Starting server at %s", address)
			// Configure the HTTP server with timeouts
			server := &http.Server{
				Addr:         address,
				Handler:      http.HandlerFunc(handleRequest),
				ReadTimeout:  10 * time.Second,  // Timeout for reading request
				WriteTimeout: 10 * time.Second,  // Timeout for writing response
				IdleTimeout:  120 * time.Second, // Timeout for idle connections
			}

			// Start the server in a goroutine
			go func() {
				log.Info().Msgf("Starting server at %s", address)
				if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					log.Fatal().Err(err).Msg("Server failed to start")
				}
			}()
			waitForTerminationRequest()
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			if err := server.Shutdown(ctx); err != nil {
				log.Error().Err(err).Msg("Error during server shutdown")
			} else {
				log.Info().Msg("Server shut down gracefully")
			}

			return nil
		},
	}

	// Run the CLI application
	err := cmd.Run(context.Background(), os.Args)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
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
func initLogger(logFile string, logLevel string) {
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
		os.Exit(1)
	}
	zerolog.SetGlobalLevel(level)

	// Log application startup
	log.Info().Msg("Logger initialized with file rotation and diode buffering")
}
