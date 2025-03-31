# MongoDB to Elasticsearch Migration Tool

A high-performance, production-ready Go application for migrating data from MongoDB to Elasticsearch with configurable batch processing, parallel workers, and comprehensive monitoring.

## Features

- **High Performance**: Multiple worker goroutines process documents in parallel with configurable batch sizes
- **Flexible Configuration**: Extensive configuration options for both MongoDB and Elasticsearch connections
- **Real-time Monitoring**: Prometheus metrics and health check endpoints for observability
- **Data Transformation**: Customizable document transformation to match existing Elasticsearch schemas
- **Data Lifecycle Management**: Optional removal of processed documents from MongoDB
- **Robust Error Handling**: Comprehensive error handling with detailed logging
- **Graceful Shutdown**: Proper resource cleanup and graceful termination

## Architecture

The application follows a clean architecture with separation of concerns:

- **Service Layer**: Handles the migration process, worker coordination, and lifecycle management
- **Worker Pool**: Processes documents in parallel with configurable concurrency
- **Database Clients**: MongoDB and Elasticsearch clients with connection management
- **Configuration**: Extensive configuration with defaults and environment variable overrides
- **Logging**: Structured logging with rotation and multi-destination output
- **Metrics**: Prometheus metrics for monitoring performance and operations

## Prerequisites

- Go 1.16 or later
- MongoDB 4.x or later
- Elasticsearch 7.x or later

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/yourusername/mongo-to-es.git
cd mongo-to-es

cp config/config.json.example config/config.json
# Then edit config/config.json with your credentials

# Build the binary
./build.sh
```

### Using Docker

```bash
# Build the Docker image
docker build -t mongo-to-es .

# Run the container
docker run -d \
  -v $(pwd)/config:/app/config \
  -v $(pwd)/logs:/app/logs \
  -p 9090:9090 \
  -p 8080:8080 \
  --name mongo-to-es \
  mongo-to-es
```

## Configuration

Configuration is loaded from `config/config.json` by default. You can specify a different path using the `-config` flag.

### Sample Configuration

```json
{
  "MongoDB": {
    "URI": "mongodb://username:password@localhost:27017/logs?authSource=admin",
    "Database": "logs",
    "Collection": "application_logs",
    "BatchSize": 1000,
    "QueryTimeout": "60s",
    "DataRetentionPeriod": "720h",
    "RemoveProcessedDocs": false,
    "ConnectionPoolSize": 100,
    "ReconnectionAttempts": 5
  },
  "Elasticsearch": {
    "URLs": ["http://localhost:9200"],
    "IndexName": "application-logs-%{+yyyy.MM.dd}",
    "Username": "elastic",
    "Password": "password",
    "BulkSize": 1000,
    "RetryBackoff": "1s",
    "MaxRetries": 3,
    "IndexRefreshInterval": "30s"
  },
  "App": {
    "WorkerCount": 10,
    "QueueSize": 100000,
    "ProcessingInterval": "5m",
    "LogPath": "./logs/app.log",
    "MetricsPort": 9090,
    "HealthCheckPort": 8080,
    "ShutdownTimeout": "30s"
  }
}
```

### Environment Variables

Key configuration values can be overridden with environment variables:

- `MONGO_URI`: MongoDB connection URI
- `MONGO_DB`: MongoDB database name
- `MONGO_COLLECTION`: MongoDB collection name
- `ES_URLS`: JSON array of Elasticsearch URLs
- `ES_INDEX`: Elasticsearch index name pattern

## Usage

### Running the Application

```bash
# Using default config path
./mongo-to-elastic

# Using custom config path
./mongo-to-elastic -config /path/to/config.json
```

### Monitoring

The application provides several endpoints for monitoring:

- **Health Check**: `http://localhost:8080/health`
- **Statistics**: `http://localhost:8080/stats`
- **Prometheus Metrics**: `http://localhost:9090/metrics`

### Deploying as a Systemd Service

Create a systemd service file at `/etc/systemd/system/mongo-to-elastic.service`:

```
[Unit]
Description=MongoDB to Elasticsearch Migration Service
After=network.target

[Service]
Type=simple
WorkingDirectory=/path/to/application
ExecStart=/path/to/application/mongo-to-elastic
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start the service:

```bash
systemctl daemon-reload
systemctl enable mongo-to-elastic
systemctl start mongo-to-elastic
```

## Performance Tuning

For optimal performance, consider the following configuration adjustments:

1. **Worker Count**: Set based on available CPU cores (typically cores × 1.5)
2. **Batch Size**: 1000-5000 documents per batch is usually a good balance
3. **Queue Size**: Set to at least (Worker Count × Batch Size × 2) to prevent blocking
4. **MongoDB Connection Pool**: Should exceed worker count by 20-50%
5. **Processing Interval**: Adjust based on data volume (5min for most cases)

## Comparison with Logstash

This tool offers several advantages over Logstash for MongoDB to Elasticsearch migration:

1. **Performance**: Purpose-built for MongoDB to Elasticsearch with optimized worker pools
2. **Resource Usage**: Lower memory and CPU footprint
3. **Monitoring**: Detailed Prometheus metrics for performance analysis
4. **Control**: Fine-grained control over document processing and error handling
5. **Configuration**: Extensive configuration options specific to MongoDB and Elasticsearch
6. **Data Lifecycle**: Optional removal of processed documents

## Troubleshooting

### Common Issues

1. **Connection Errors**: Verify MongoDB and Elasticsearch connection settings
2. **Index Mapping Conflicts**: Check document structure consistency
3. **Performance Issues**: Adjust worker count, batch sizes, and connection pool settings
4. **Missing Documents**: Verify timestamp filtering and retention period settings

### Logs

Check the application logs for detailed error information:

```bash
tail -f logs/app.log
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.