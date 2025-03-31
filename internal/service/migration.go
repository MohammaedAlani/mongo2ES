package service

import (
	"context"
	"fmt"
	"github.com/MohammaedAlani/Mongo2ES/internal/config"
	"github.com/MohammaedAlani/Mongo2ES/internal/models"
	"github.com/MohammaedAlani/Mongo2ES/pkg/elasticsearch"
	"github.com/MohammaedAlani/Mongo2ES/pkg/logger"
	"github.com/MohammaedAlani/Mongo2ES/pkg/metrics"
	"github.com/MohammaedAlani/Mongo2ES/pkg/mongodb"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// MigrationService handles the migration of data from MongoDB to Elasticsearch
type MigrationService struct {
	config        config.Config
	mongoClient   *mongodb.Client
	esClient      *elasticsearch.Client
	logger        *zap.SugaredLogger
	metrics       *metrics.MetricsCollector
	workers       []*Worker
	queue         chan models.Document
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
	startTime     time.Time
	httpServer    *http.Server
	metricsServer *http.Server
}

// NewMigrationService creates a new migration service
func NewMigrationService(cfg config.Config) (*MigrationService, error) {
	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize logger
	log, err := logger.InitLogger(cfg.App.LogPath)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Initialize metrics
	metricsCollector := metrics.NewMetricsCollector()

	// Initialize MongoDB client
	mongoClient, err := mongodb.NewClient(ctx, cfg, log)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create MongoDB client: %w", err)
	}

	// Initialize Elasticsearch client
	esClient, err := elasticsearch.NewClient(cfg, log)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}

	// Ensure Elasticsearch index exists
	if err := esClient.EnsureIndex(ctx); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to ensure Elasticsearch index: %w", err)
	}

	// Create service
	service := &MigrationService{
		config:      cfg,
		mongoClient: mongoClient,
		esClient:    esClient,
		logger:      log,
		metrics:     metricsCollector,
		ctx:         ctx,
		cancel:      cancel,
		startTime:   time.Now(),
		queue:       make(chan models.Document, cfg.App.QueueSize),
	}

	return service, nil
}

// Start begins the migration process
func (s *MigrationService) Start() error {
	s.logger.Info("Starting MongoDB to Elasticsearch migration service")

	// Start HTTP server for metrics
	s.startMetricsServer()

	// Start HTTP server for health checks
	s.startHealthServer()

	// Initialize and start workers
	s.startWorkers()

	// Process data at intervals
	interval, err := s.config.GetProcessingInterval()
	if err != nil {
		return fmt.Errorf("invalid processing interval: %w", err)
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.processData()
		case <-s.ctx.Done():
			s.logger.Info("Service context cancelled, shutting down")
			return nil
		}
	}
}

// startMetricsServer starts the HTTP server for Prometheus metrics
func (s *MigrationService) startMetricsServer() {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	s.metricsServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.App.MetricsPort),
		Handler: mux,
	}

	go func() {
		s.logger.Infof("Starting metrics server on port %d", s.config.App.MetricsPort)
		if err := s.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Errorf("Metrics server failed: %v", err)
		}
	}()
}

// startHealthServer starts the HTTP server for health checks
func (s *MigrationService) startHealthServer() {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Stats endpoint
	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement service statistics
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"running"}`))
	})

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.App.HealthCheckPort),
		Handler: mux,
	}

	go func() {
		s.logger.Infof("Starting health check server on port %d", s.config.App.HealthCheckPort)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Errorf("Health check server failed: %v", err)
		}
	}()
}

// startWorkers initializes and starts worker goroutines
func (s *MigrationService) startWorkers() {
	s.workers = make([]*Worker, s.config.App.WorkerCount)

	for i := 0; i < s.config.App.WorkerCount; i++ {
		s.wg.Add(1)
		worker := NewWorker(
			i,
			s.queue,
			s.esClient,
			s.logger,
			s.metrics,
			&s.wg,
			s.config.Elasticsearch.BulkSize,
			s.ctx,
		)
		s.workers[i] = worker
		go worker.Start()
	}

	s.logger.Infof("Started %d workers", s.config.App.WorkerCount)
}

// processData fetches data from MongoDB and puts it in the queue
func (s *MigrationService) processData() {
	startTime := time.Now()
	s.logger.Info("Starting data processing cycle")

	// Create MongoDB query context with timeout
	timeout, err := s.config.GetMongoQueryTimeout()
	if err != nil {
		s.logger.Errorf("Invalid query timeout: %v", err)
		return
	}
	queryCtx, cancel := context.WithTimeout(s.ctx, timeout)
	defer cancel()

	// Query MongoDB for documents older than retention period
	retentionPeriod, err := s.config.GetMongoRetentionPeriod()
	if err != nil {
		s.logger.Errorf("Invalid retention period: %v", err)
		return
	}
	cutoffTime := time.Now().Add(-retentionPeriod)

	// Assuming documents have a timestamp field named "timestamp"
	filter := bson.M{"timestamp": bson.M{"$lt": cutoffTime}}

	// Setup find options
	findOptions := options.Find().
		SetBatchSize(int32(s.config.MongoDB.BatchSize)).
		SetNoCursorTimeout(true)

	// Execute query
	collection := s.mongoClient.GetCollection()
	cursor, err := collection.Find(queryCtx, filter, findOptions)
	if err != nil {
		s.logger.Errorf("Failed to query MongoDB: %v", err)
		s.metrics.FailedDocs.Inc()
		return
	}
	defer cursor.Close(queryCtx)

	// Process results
	var processedCount int
	var docsToRemove []primitive.ObjectID

	for cursor.Next(queryCtx) {
		var doc models.Document
		if err := cursor.Decode(&doc); err != nil {
			s.logger.Errorf("Failed to decode document: %v", err)
			s.metrics.FailedDocs.Inc()
			continue
		}

		// Store document ID for later removal if configured
		if s.config.MongoDB.RemoveProcessedDocs {
			if id, ok := doc["_id"].(primitive.ObjectID); ok {
				docsToRemove = append(docsToRemove, id)
			}
		}

		// Put document in queue
		select {
		case s.queue <- doc:
			processedCount++
			s.metrics.QueueSize.Inc()
		case <-queryCtx.Done():
			s.logger.Warn("Query context cancelled while processing documents")
			return
		}
	}

	if err := cursor.Err(); err != nil {
		s.logger.Errorf("Cursor error: %v", err)
	}

	// Remove processed documents if configured
	if s.config.MongoDB.RemoveProcessedDocs && len(docsToRemove) > 0 {
		s.removeProcessedDocuments(queryCtx, docsToRemove)
	}

	// Update metrics
	s.metrics.LastRunTimestamp.Set(float64(time.Now().Unix()))
	s.metrics.MongoQueryDuration.Observe(time.Since(startTime).Seconds())

	duration := time.Since(startTime)
	s.logger.Infof("Processing cycle completed: %d documents queued in %v", processedCount, duration)
}

// removeProcessedDocuments removes documents that have been processed
func (s *MigrationService) removeProcessedDocuments(ctx context.Context, ids []primitive.ObjectID) {
	if len(ids) == 0 {
		return
	}

	// Create delete context with timeout
	timeout, err := s.config.GetMongoQueryTimeout()
	if err != nil {
		s.logger.Errorf("Invalid query timeout: %v", err)
		return
	}
	deleteCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create filter for IDs to remove
	filter := bson.M{"_id": bson.M{"$in": ids}}

	// Execute deletion
	collection := s.mongoClient.GetCollection()
	result, err := collection.DeleteMany(deleteCtx, filter)
	if err != nil {
		s.logger.Errorf("Failed to remove processed documents: %v", err)
		return
	}

	s.logger.Infof("Removed %d processed documents", result.DeletedCount)
}

// Shutdown gracefully shuts down the service
func (s *MigrationService) Shutdown() error {
	s.logger.Info("Shutting down service...")

	// Signal cancellation to all goroutines
	s.cancel()

	// Create a context with timeout for shutdown
	timeout, err := s.config.GetShutdownTimeout()
	if err != nil {
		s.logger.Errorf("Invalid shutdown timeout: %v", err)
		timeout = 30 * time.Second // Fallback to default
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Shutdown HTTP servers
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			s.logger.Errorf("Error shutting down health check server: %v", err)
		}
	}

	if s.metricsServer != nil {
		if err := s.metricsServer.Shutdown(shutdownCtx); err != nil {
			s.logger.Errorf("Error shutting down metrics server: %v", err)
		}
	}

	// Close the queue to signal workers to finish
	close(s.queue)

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("All workers completed successfully")
	case <-shutdownCtx.Done():
		s.logger.Warn("Shutdown timed out, some workers may not have completed")
	}

	// Disconnect from MongoDB
	if err := s.mongoClient.Disconnect(shutdownCtx); err != nil {
		s.logger.Errorf("Error disconnecting from MongoDB: %v", err)
	}

	s.logger.Info("Service shutdown complete")
	return nil
}
