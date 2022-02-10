package arangodb_adapter

import (
	"context"
	"github.com/arangodb/go-driver"
)

// ArangoDatabase abstraction
type ArangoDatabase interface {
	ReadDB() driver.Database
	WriteDB() driver.Database
	Health() map[string]error
	Disconnect(ctx context.Context) error
}
