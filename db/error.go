// Package db exports error types for db package.
package db

import "errors"

// ErrNoDefaultMongoClient is returned when no MongoDB client is registered with a connection string
var ErrNoDefaultMongoClient = errors.New("no mongodb client is registered with a connection string")
