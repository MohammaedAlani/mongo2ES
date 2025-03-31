package models

// Document represents a document with flexible schema
type Document map[string]interface{}

// BulkResponse represents the Elasticsearch bulk API response
type BulkResponse struct {
	Took   int  `json:"took"`
	Errors bool `json:"errors"`
	Items  []struct {
		Index struct {
			ID     string `json:"_id"`
			Result string `json:"result"`
			Status int    `json:"status"`
			Error  struct {
				Type   string `json:"type"`
				Reason string `json:"reason"`
			} `json:"error,omitempty"`
		} `json:"index"`
	} `json:"items"`
}

// MongoStats contains MongoDB statistics
type MongoStats struct {
	DocumentsProcessed int `json:"documents_processed"`
	DocumentsRemoved   int `json:"documents_removed"`
	QueryTime          int `json:"query_time_ms"`
}

// ElasticStats contains Elasticsearch statistics
type ElasticStats struct {
	DocumentsIndexed int `json:"documents_indexed"`
	IndexingTime     int `json:"indexing_time_ms"`
	BulkRequests     int `json:"bulk_requests"`
	FailedRequests   int `json:"failed_requests"`
}

// WorkerStats contains worker statistics
type WorkerStats struct {
	ID              int `json:"id"`
	DocumentsQueued int `json:"documents_queued"`
	ProcessingTime  int `json:"processing_time_ms"`
}

// ServiceStats contains service statistics
type ServiceStats struct {
	StartTime          string        `json:"start_time"`
	Uptime             string        `json:"uptime"`
	CurrentQueueSize   int           `json:"current_queue_size"`
	TotalDocuments     int           `json:"total_documents_processed"`
	FailedDocuments    int           `json:"failed_documents"`
	AverageProcessTime int           `json:"average_process_time_ms"`
	MongoDB            MongoStats    `json:"mongodb"`
	Elasticsearch      ElasticStats  `json:"elasticsearch"`
	Workers            []WorkerStats `json:"workers"`
}
