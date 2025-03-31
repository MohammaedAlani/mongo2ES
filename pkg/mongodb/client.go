package mongodb

import (
	"context"
	"fmt"
	"github.com/MohammaedAlani/Mongo2ES/internal/config"
	"go.mongodb.org/mongo-driver/bson"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"
)

// Client wraps the MongoDB client
type Client struct {
	client     *mongo.Client
	database   *mongo.Database
	collection *mongo.Collection
	config     config.Config
	logger     *zap.SugaredLogger
}

// NewClient creates a new MongoDB client
func NewClient(ctx context.Context, cfg config.Config, logger *zap.SugaredLogger) (*Client, error) {
	// Initialize MongoDB client options
	clientOpts := options.Client().
		ApplyURI(cfg.MongoDB.URI).
		SetMaxPoolSize(uint64(cfg.MongoDB.ConnectionPoolSize)).
		SetRetryWrites(true).
		SetRetryReads(true)

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping MongoDB to verify connection
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	// Get database and collection
	database := client.Database(cfg.MongoDB.Database)
	collection := database.Collection(cfg.MongoDB.Collection)

	return &Client{
		client:     client,
		database:   database,
		collection: collection,
		config:     cfg,
		logger:     logger.With("component", "mongodb"),
	}, nil
}

// GetCollection returns the MongoDB collection
func (c *Client) GetCollection() *mongo.Collection {
	return c.collection
}

// Disconnect closes the MongoDB connection
func (c *Client) Disconnect(ctx context.Context) error {
	return c.client.Disconnect(ctx)
}

// CreateIndexes ensures that required indexes exist
func (c *Client) CreateIndexes(ctx context.Context) error {
	// Example: Create index on timestamp field for efficient querying
	// This is just an example - adjust according to your actual document structure
	indexModel := mongo.IndexModel{
		Keys: bson.D{{"timestamp", 1}},
		Options: options.Index().
			SetName("timestamp_index").
			SetBackground(true),
	}

	_, err := c.collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	return nil
}
