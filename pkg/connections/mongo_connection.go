// Package connections owns the database and cache clients.
//
// Connections are established explicitly by the caller rather than in an
// init() function, so importing this package has no side effects and tests can
// run without a live database.
package connections

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/aekam27/Semantic/pkg/config"
)

// Mongo is a connected MongoDB database handle.
type Mongo struct {
	client   *mongo.Client
	database *mongo.Database
}

// NewMongo dials MongoDB and verifies the connection with a ping.
func NewMongo(ctx context.Context, cfg config.MongoConfig) (*Mongo, error) {
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(cfg.URI).SetServerAPIOptions(serverAPI)

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("connect to mongo: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		// Release the pool the failed client already opened.
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("ping mongo: %w", err)
	}

	log.Println("Mongo connected successfully")
	return &Mongo{client: client, database: client.Database(cfg.Database)}, nil
}

// VFind runs an aggregation pipeline against the named collection.
func (m *Mongo) VFind(ctx context.Context, pipeline bson.A, collectionName string) (*mongo.Cursor, error) {
	return m.database.Collection(collectionName).Aggregate(ctx, pipeline)
}

// Close disconnects the underlying client.
func (m *Mongo) Close(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}
