package db

// Description: Define error types for db package.

import "errors"

var (
	ErrNoDefaultMongoClient = errors.New("no mongodb client is registered with a connection string")
)
