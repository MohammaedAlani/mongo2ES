package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/MohammaedAlani/Mongo2ES/internal/config"
	"github.com/MohammaedAlani/Mongo2ES/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"go.uber.org/zap"
)

// Client wraps the Elasticsearch client
type Client struct {
	client    *elasticsearch.Client
	config    config.Config
	logger    *zap.SugaredLogger
	indexName string
}

// NewClient creates a new Elasticsearch client
func NewClient(cfg config.Config, logger *zap.SugaredLogger) (*Client, error) {
	// Parse retry backoff
	retryBackoff, err := cfg.GetESRetryBackoff()
	if err != nil {
		return nil, fmt.Errorf("invalid retry backoff: %w", err)
	}

	esCfg := elasticsearch.Config{
		Addresses: cfg.Elasticsearch.URLs,
		Username:  cfg.Elasticsearch.Username,
		Password:  cfg.Elasticsearch.Password,
		RetryBackoff: func(attempt int) time.Duration {
			return retryBackoff * time.Duration(attempt)
		},
		MaxRetries: cfg.Elasticsearch.MaxRetries,
	}

	// Create Elasticsearch client
	client, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}

	// Verify Elasticsearch connection
	res, err := client.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Elasticsearch: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("Elasticsearch returned error: %s", res.String())
	}

	// Read and log cluster info
	var info map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("error parsing Elasticsearch info response: %w", err)
	}

	version := info["version"].(map[string]interface{})["number"]
	logger.Infof("Connected to Elasticsearch cluster %q version %q", info["cluster_name"], version)

	return &Client{
		client:    client,
		config:    cfg,
		logger:    logger.With("component", "elasticsearch"),
		indexName: cfg.Elasticsearch.IndexName,
	}, nil
}

// formatIndexName replaces date patterns in the index name
func (c *Client) formatIndexName(indexName string) string {
	now := time.Now()

	// Replace %{+yyyy} with the current year
	indexName = strings.Replace(indexName, "%{+yyyy}", fmt.Sprintf("%d", now.Year()), -1)

	// Replace %{+MM} with the current month (zero-padded)
	indexName = strings.Replace(indexName, "%{+MM}", fmt.Sprintf("%02d", now.Month()), -1)

	// Replace %{+dd} with the current day (zero-padded)
	indexName = strings.Replace(indexName, "%{+dd}", fmt.Sprintf("%02d", now.Day()), -1)

	// Handle the combined date pattern - %{+yyyy.MM.dd}
	indexName = strings.Replace(indexName, "%{+yyyy.MM.dd}",
		fmt.Sprintf("%d.%02d.%02d", now.Year(), now.Month(), now.Day()), -1)

	return indexName
}

// EnsureIndex ensures that the index exists with proper settings
func (c *Client) EnsureIndex(ctx context.Context) error {
	// Format the index name first
	indexName := c.formatIndexName(c.indexName)

	// Check if index exists
	res, err := c.client.Indices.Exists([]string{indexName})
	if err != nil {
		return fmt.Errorf("failed to check if index exists: %w", err)
	}

	// If index does not exist, create it
	if res.StatusCode == 404 {
		indexSettings := map[string]interface{}{
			"settings": map[string]interface{}{
				"number_of_shards":   3,
				"number_of_replicas": 1,
				"refresh_interval":   c.config.Elasticsearch.IndexRefreshInterval,
			},
		}

		body, err := json.Marshal(indexSettings)
		if err != nil {
			return fmt.Errorf("failed to marshal index settings: %w", err)
		}

		res, err = c.client.Indices.Create(
			indexName,
			c.client.Indices.Create.WithBody(bytes.NewReader(body)),
			c.client.Indices.Create.WithContext(ctx),
		)
		if err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
		defer res.Body.Close()

		if res.IsError() {
			return fmt.Errorf("error creating index: %s", res.String())
		}

		c.logger.Infof("Created index %q", indexName)
	}

	return nil
}

// BulkIndex indexes a batch of documents
// BulkIndex indexes a batch of documents
func (c *Client) BulkIndex(ctx context.Context, docs []models.Document) (models.BulkResponse, error) {
	var response models.BulkResponse
	var buf bytes.Buffer

	// Format index name with date patterns
	indexName := c.formatIndexName(c.indexName)

	// Build the bulk request body
	for _, doc := range docs {
		// Store MongoDB ID before removing it
		var mongoID string
		if id, ok := doc["_id"].(primitive.ObjectID); ok {
			mongoID = id.Hex()
			delete(doc, "_id")
		}

		// Get timestamp and format it correctly
		if timestamp, ok := doc["timestamp"].(time.Time); ok {
			// Create @timestamp field that ES expects
			doc["@timestamp"] = timestamp.Format(time.RFC3339)
		}

		// Add additional fields to match existing schema
		doc["mongo_id"] = mongoID
		doc["source"] = "mongodb_logs"
		doc["host"] = "ELK-STACK"
		doc["@version"] = "1"

		// Add metadata line for the action
		meta := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": indexName,
				"_id":    mongoID,
			},
		}

		if err := json.NewEncoder(&buf).Encode(meta); err != nil {
			return response, fmt.Errorf("failed to encode metadata: %w", err)
		}

		// Add source document line
		if err := json.NewEncoder(&buf).Encode(doc); err != nil {
			return response, fmt.Errorf("failed to encode document: %w", err)
		}
	}

	// Execute bulk request
	res, err := c.client.Bulk(
		bytes.NewReader(buf.Bytes()),
		c.client.Bulk.WithContext(ctx),
		c.client.Bulk.WithRefresh("true"),
	)
	if err != nil {
		return response, fmt.Errorf("failed to perform bulk request: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return response, fmt.Errorf("bulk request failed: %s", res.String())
	}

	// Parse response
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return response, fmt.Errorf("failed to parse response: %w", err)
	}

	return response, nil
}

// Close closes the Elasticsearch connection
func (c *Client) Close() {
	// The official Elasticsearch client doesn't have a close method
	// This is a placeholder for future implementations or custom clients
}
