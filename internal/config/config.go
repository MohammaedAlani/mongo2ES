package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Config holds application configuration
type Config struct {
	MongoDB struct {
		URI                  string `json:"URI"`
		Database             string `json:"Database"`
		Collection           string `json:"Collection"`
		BatchSize            int    `json:"BatchSize"`
		QueryTimeout         string `json:"QueryTimeout"`
		DataRetentionPeriod  string `json:"DataRetentionPeriod"`
		RemoveProcessedDocs  bool   `json:"RemoveProcessedDocs"`
		ConnectionPoolSize   int    `json:"ConnectionPoolSize"`
		ReconnectionAttempts int    `json:"ReconnectionAttempts"`
	}
	Elasticsearch struct {
		URLs                 []string `json:"URLs"`
		IndexName            string   `json:"IndexName"`
		Username             string   `json:"Username"`
		Password             string   `json:"Password"`
		BulkSize             int      `json:"BulkSize"`
		RetryBackoff         string   `json:"RetryBackoff"`
		MaxRetries           int      `json:"MaxRetries"`
		IndexRefreshInterval string   `json:"IndexRefreshInterval"`
	}
	App struct {
		WorkerCount        int    `json:"WorkerCount"`
		QueueSize          int    `json:"QueueSize"`
		ProcessingInterval string `json:"ProcessingInterval"`
		LogPath            string `json:"LogPath"`
		MetricsPort        int    `json:"MetricsPort"`
		HealthCheckPort    int    `json:"HealthCheckPort"`
		ShutdownTimeout    string `json:"ShutdownTimeout"`
	}
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) (Config, error) {
	var config Config

	// Set defaults
	config.MongoDB.BatchSize = 1000
	config.MongoDB.QueryTimeout = "60s"
	config.MongoDB.DataRetentionPeriod = "720h" // 30 days
	config.MongoDB.ConnectionPoolSize = 100
	config.MongoDB.ReconnectionAttempts = 5

	config.Elasticsearch.BulkSize = 1000
	config.Elasticsearch.RetryBackoff = "1s"
	config.Elasticsearch.MaxRetries = 3
	config.Elasticsearch.IndexRefreshInterval = "30s"

	config.App.WorkerCount = 10
	config.App.QueueSize = 100000
	config.App.ProcessingInterval = "5m"
	config.App.ShutdownTimeout = "30s"
	config.App.MetricsPort = 9090
	config.App.HealthCheckPort = 8080
	config.App.LogPath = "./logs/app.log"

	// Load from file if specified
	if configPath != "" {
		file, err := os.Open(configPath)
		if err != nil {
			return config, fmt.Errorf("failed to open config file: %w", err)
		}
		defer file.Close()

		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&config); err != nil {
			return config, fmt.Errorf("failed to decode config file: %w", err)
		}
	}

	// Override with environment variables
	if mongoURI := os.Getenv("MONGO_URI"); mongoURI != "" {
		config.MongoDB.URI = mongoURI
	}

	if mongoDb := os.Getenv("MONGO_DB"); mongoDb != "" {
		config.MongoDB.Database = mongoDb
	}

	if mongoColl := os.Getenv("MONGO_COLLECTION"); mongoColl != "" {
		config.MongoDB.Collection = mongoColl
	}

	if esURLs := os.Getenv("ES_URLS"); esURLs != "" {
		var urls []string
		if err := json.Unmarshal([]byte(esURLs), &urls); err == nil {
			config.Elasticsearch.URLs = urls
		}
	}

	if esIndex := os.Getenv("ES_INDEX"); esIndex != "" {
		config.Elasticsearch.IndexName = esIndex
	}

	// Validate required fields
	if config.MongoDB.URI == "" {
		return config, fmt.Errorf("MongoDB URI is required")
	}

	if config.MongoDB.Database == "" {
		return config, fmt.Errorf("MongoDB database name is required")
	}

	if config.MongoDB.Collection == "" {
		return config, fmt.Errorf("MongoDB collection name is required")
	}

	if len(config.Elasticsearch.URLs) == 0 {
		return config, fmt.Errorf("at least one Elasticsearch URL is required")
	}

	if config.Elasticsearch.IndexName == "" {
		return config, fmt.Errorf("Elasticsearch index name is required")
	}

	return config, nil
}

// GetMongoQueryTimeout parses the QueryTimeout string to a time.Duration
func (c *Config) GetMongoQueryTimeout() (time.Duration, error) {
	return time.ParseDuration(c.MongoDB.QueryTimeout)
}

// GetMongoRetentionPeriod parses the DataRetentionPeriod string to a time.Duration
func (c *Config) GetMongoRetentionPeriod() (time.Duration, error) {
	return time.ParseDuration(c.MongoDB.DataRetentionPeriod)
}

// GetESRetryBackoff parses the RetryBackoff string to a time.Duration
func (c *Config) GetESRetryBackoff() (time.Duration, error) {
	return time.ParseDuration(c.Elasticsearch.RetryBackoff)
}

// GetProcessingInterval parses the ProcessingInterval string to a time.Duration
func (c *Config) GetProcessingInterval() (time.Duration, error) {
	return time.ParseDuration(c.App.ProcessingInterval)
}

// GetShutdownTimeout parses the ShutdownTimeout string to a time.Duration
func (c *Config) GetShutdownTimeout() (time.Duration, error) {
	return time.ParseDuration(c.App.ShutdownTimeout)
}
