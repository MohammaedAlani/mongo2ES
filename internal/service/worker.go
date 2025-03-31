package service

import (
	"context"
	"fmt"
	"github.com/MohammaedAlani/Mongo2ES/internal/models"
	"github.com/MohammaedAlani/Mongo2ES/pkg/elasticsearch"
	"github.com/MohammaedAlani/Mongo2ES/pkg/metrics"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Worker represents a data processing worker
type Worker struct {
	id       int
	queue    chan models.Document
	esClient *elasticsearch.Client
	logger   *zap.SugaredLogger
	metrics  *metrics.MetricsCollector
	wg       *sync.WaitGroup
	bulkSize int
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewWorker creates a new worker
func NewWorker(
	id int,
	queue chan models.Document,
	esClient *elasticsearch.Client,
	logger *zap.SugaredLogger,
	metrics *metrics.MetricsCollector,
	wg *sync.WaitGroup,
	bulkSize int,
	ctx context.Context,
) *Worker {
	workerCtx, cancel := context.WithCancel(ctx)
	return &Worker{
		id:       id,
		queue:    queue,
		esClient: esClient,
		logger:   logger.With("worker", id),
		metrics:  metrics,
		wg:       wg,
		bulkSize: bulkSize,
		ctx:      workerCtx,
		cancel:   cancel,
	}
}

// Start begins worker processing
func (w *Worker) Start() {
	defer w.wg.Done()

	w.logger.Infof("Worker %d started", w.id)

	// Create a buffer for bulk processing
	buffer := make([]models.Document, 0, w.bulkSize)

	for {
		select {
		case doc, ok := <-w.queue:
			if !ok {
				// Channel closed, process remaining documents and exit
				if len(buffer) > 0 {
					w.processBulk(buffer)
				}
				w.logger.Infof("Worker %d stopped (channel closed)", w.id)
				return
			}

			// Update queue size metric
			w.metrics.QueueSize.Dec()

			// Add document to buffer
			buffer = append(buffer, doc)

			// Process buffer when it reaches the bulk size
			if len(buffer) >= w.bulkSize {
				if err := w.processBulk(buffer); err != nil {
					w.logger.Errorf("Error processing bulk: %v", err)
				}
				buffer = make([]models.Document, 0, w.bulkSize)
			}

		case <-w.ctx.Done():
			// Context cancelled, process remaining documents and exit
			if len(buffer) > 0 {
				w.processBulk(buffer)
			}
			w.logger.Infof("Worker %d stopped (context cancelled)", w.id)
			return
		}
	}
}

// processBulk sends a batch of documents to Elasticsearch
func (w *Worker) processBulk(docs []models.Document) error {
	if len(docs) == 0 {
		return nil
	}

	startTime := time.Now()
	w.logger.Infof("Processing bulk of %d documents", len(docs))

	// Set worker utilization metric
	w.metrics.WorkerUtilization.WithLabelValues(strconv.Itoa(w.id)).Set(100.0)
	defer w.metrics.WorkerUtilization.WithLabelValues(strconv.Itoa(w.id)).Set(0.0)

	// Send bulk request to Elasticsearch
	ctx, cancel := context.WithTimeout(w.ctx, 30*time.Second)
	defer cancel()

	response, err := w.esClient.BulkIndex(ctx, docs)
	if err != nil {
		for range docs {
			w.metrics.FailedDocs.Inc()
		}
		return fmt.Errorf("bulk indexing failed: %w", err)
	}

	// Process response and update metrics
	if response.Errors {
		// Some items failed, count them
		var successCount, failCount int

		for _, item := range response.Items {
			// Status codes 2xx indicate success
			if item.Index.Status >= 200 && item.Index.Status < 300 {
				successCount++
				w.metrics.ProcessedDocs.Inc()
			} else {
				failCount++
				w.metrics.FailedDocs.Inc()
				w.logger.Warnf("Document indexing failed: %s - %s",
					item.Index.Error.Type,
					item.Index.Error.Reason)
			}
		}

		w.logger.Infof("Bulk processing completed: %d successful, %d failed",
			successCount, failCount)
	} else {
		// All items succeeded
		for range docs {
			w.metrics.ProcessedDocs.Inc()
		}
	}

	duration := time.Since(startTime)
	w.metrics.ElasticBulkDuration.Observe(duration.Seconds())

	w.logger.Infof("Bulk processing completed in %v", duration)
	return nil
}

// Stop gracefully stops the worker
func (w *Worker) Stop() {
	w.logger.Infof("Stopping worker %d", w.id)
	w.cancel()
}
