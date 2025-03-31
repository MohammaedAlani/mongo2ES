package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// MetricsCollector holds Prometheus metrics
type MetricsCollector struct {
	ProcessedDocs       prometheus.Counter
	FailedDocs          prometheus.Counter
	ProcessingDuration  prometheus.Histogram
	QueueSize           prometheus.Gauge
	LastRunTimestamp    prometheus.Gauge
	MongoQueryDuration  prometheus.Histogram
	ElasticBulkDuration prometheus.Histogram
	WorkerUtilization   prometheus.GaugeVec
}

// NewMetricsCollector initializes a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	m := &MetricsCollector{
		ProcessedDocs: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "mongo_to_es_processed_docs_total",
			Help: "Total number of documents processed",
		}),
		FailedDocs: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "mongo_to_es_failed_docs_total",
			Help: "Total number of documents that failed processing",
		}),
		ProcessingDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "mongo_to_es_processing_duration_seconds",
			Help:    "Time spent processing documents",
			Buckets: prometheus.DefBuckets,
		}),
		QueueSize: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "mongo_to_es_queue_size",
			Help: "Current number of documents in processing queue",
		}),
		LastRunTimestamp: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "mongo_to_es_last_run_timestamp",
			Help: "Timestamp of the last successful processing run",
		}),
		MongoQueryDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "mongo_to_es_mongo_query_duration_seconds",
			Help:    "Time spent querying MongoDB",
			Buckets: prometheus.DefBuckets,
		}),
		ElasticBulkDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "mongo_to_es_elastic_bulk_duration_seconds",
			Help:    "Time spent on Elasticsearch bulk operations",
			Buckets: prometheus.DefBuckets,
		}),
		WorkerUtilization: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "mongo_to_es_worker_utilization",
				Help: "Worker utilization percentage",
			},
			[]string{"worker_id"},
		),
	}

	// Register metrics
	prometheus.MustRegister(m.ProcessedDocs)
	prometheus.MustRegister(m.FailedDocs)
	prometheus.MustRegister(m.ProcessingDuration)
	prometheus.MustRegister(m.QueueSize)
	prometheus.MustRegister(m.LastRunTimestamp)
	prometheus.MustRegister(m.MongoQueryDuration)
	prometheus.MustRegister(m.ElasticBulkDuration)
	prometheus.MustRegister(m.WorkerUtilization)

	return m
}
