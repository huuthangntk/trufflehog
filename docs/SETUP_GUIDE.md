# TruffleHog API Setup Guide

This guide will help you set up and run the TruffleHog REST API service.

## Prerequisites

- Go 1.24 or later (for building from source)
- Docker and Docker Compose (for containerized deployment)
- Git

## Installation Methods

### Method 1: Docker Compose (Recommended)

The easiest way to get started is using Docker Compose:

```bash
# Clone the repository
git clone https://github.com/huuthangntk/trufflehog.git
cd trufflehog

# Start the API server
docker-compose up -d

# Check if it's running
curl http://localhost:8080/health
```

The API will be available at `http://localhost:8080`.

### Method 2: Docker

Build and run using Docker directly:

```bash
# Build the image
docker build -f Dockerfile.api -t trufflehog-api .

# Run the container
docker run -d -p 8080:8080 --name trufflehog-api trufflehog-api

# Check logs
docker logs -f trufflehog-api

# Stop the container
docker stop trufflehog-api
```

### Method 3: Build from Source

Build and run directly with Go:

```bash
# Clone the repository
git clone https://github.com/huuthangntk/trufflehog.git
cd trufflehog

# Build the API server
go build -o trufflehog-api ./cmd/api

# Run the server
./trufflehog-api -addr :8080
```

## Configuration

### Command-Line Options

```bash
./trufflehog-api -addr :8080
```

Options:
- `-addr`: Server address (default: `:8080`)

### Environment Variables

You can also configure using environment variables:

```bash
export API_PORT=8080
./trufflehog-api
```

## Verification

After starting the server, verify it's working:

```bash
# Health check
curl http://localhost:8080/health

# Expected response:
# {"status":"healthy"}
```

## Quick Start Example

### 1. Start a Scan

```bash
curl -X POST http://localhost:8080/api/v1/scan \
  -H "Content-Type: application/json" \
  -d '{
    "repo_url": "https://github.com/example/test-repo.git",
    "verify": true
  }'
```

Response:
```json
{
  "scan_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "pending",
  "message": "Scan initiated successfully",
  "created_at": "2024-12-04T10:30:00Z"
}
```

### 2. Check Scan Status

```bash
curl "http://localhost:8080/api/v1/scan/status?scan_id=550e8400-e29b-41d4-a716-446655440000"
```

### 3. List All Scans

```bash
curl http://localhost:8080/api/v1/scans
```

## Using the Client Libraries

### Python Client

```bash
# Install dependencies
pip install requests

# Run the example
python examples/api-client.py
```

### Node.js Client

```bash
# Install dependencies
npm install axios

# Run the example
node examples/api-client.js
```

## Setting Up Webhooks

### Option 1: Use webhook.site (Testing)

For testing, use [webhook.site](https://webhook.site):

1. Go to https://webhook.site
2. Copy your unique URL
3. Use it in your scan request:

```bash
curl -X POST http://localhost:8080/api/v1/scan \
  -H "Content-Type: application/json" \
  -d '{
    "repo_url": "https://github.com/example/repo.git",
    "webhook_url": "https://webhook.site/your-unique-id",
    "verify": true
  }'
```

### Option 2: Run Local Webhook Server

```bash
# Install dependencies
npm install express body-parser

# Run the webhook server
node examples/webhook-server.js
```

Then use `http://localhost:3000/webhook` as your webhook URL.

### Option 3: Use ngrok (Local Development)

To receive webhooks on your local machine:

```bash
# Install ngrok
# https://ngrok.com/download

# Start ngrok
ngrok http 3000

# Use the ngrok URL in your scan request
curl -X POST http://localhost:8080/api/v1/scan \
  -H "Content-Type: application/json" \
  -d '{
    "repo_url": "https://github.com/example/repo.git",
    "webhook_url": "https://your-id.ngrok.io/webhook",
    "verify": true
  }'
```

## Production Deployment

### Security Considerations

1. **Use HTTPS**: Always use TLS/SSL in production
2. **Add Authentication**: Implement API key or OAuth2
3. **Rate Limiting**: Add rate limiting to prevent abuse
4. **Input Validation**: Validate all repository URLs
5. **Network Security**: Use firewalls and security groups

### Recommended Architecture

```
Internet
   │
   ▼
Load Balancer (HTTPS)
   │
   ▼
API Gateway (Auth, Rate Limiting)
   │
   ▼
TruffleHog API (Multiple Instances)
   │
   ▼
Message Queue (for async processing)
   │
   ▼
Worker Nodes (scan execution)
```

### Kubernetes Deployment

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: trufflehog-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: trufflehog-api
  template:
    metadata:
      labels:
        app: trufflehog-api
    spec:
      containers:
      - name: trufflehog-api
        image: trufflehog-api:latest
        ports:
        - containerPort: 8080
        env:
        - name: API_PORT
          value: "8080"
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "2000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: trufflehog-api
spec:
  selector:
    app: trufflehog-api
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

Deploy:
```bash
kubectl apply -f deployment.yaml
```

### AWS ECS Deployment

```json
{
  "family": "trufflehog-api",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "1024",
  "memory": "2048",
  "containerDefinitions": [
    {
      "name": "trufflehog-api",
      "image": "trufflehog-api:latest",
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "API_PORT",
          "value": "8080"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/trufflehog-api",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ]
}
```

## Monitoring

### Health Checks

The API provides a health check endpoint:

```bash
curl http://localhost:8080/health
```

### Logging

The API logs all operations. In production, configure log aggregation:

- **ELK Stack**: Elasticsearch, Logstash, Kibana
- **CloudWatch**: AWS CloudWatch Logs
- **Stackdriver**: Google Cloud Logging
- **Datadog**: Application monitoring

### Metrics

Consider implementing metrics for:

- Scan duration
- Success/failure rates
- API response times
- Webhook delivery status
- Resource usage

## Troubleshooting

### API Server Won't Start

```bash
# Check if port is already in use
lsof -i :8080

# Try a different port
./trufflehog-api -addr :8081
```

### Scans Failing

Check the scan status for error messages:

```bash
curl "http://localhost:8080/api/v1/scan/status?scan_id=YOUR_SCAN_ID"
```

Common issues:
- Invalid repository URL
- Repository requires authentication
- Network connectivity issues
- Insufficient resources

### Webhooks Not Received

1. Verify webhook URL is accessible
2. Check webhook server logs
3. Test webhook URL manually:

```bash
curl -X POST https://your-webhook-url.com/webhook \
  -H "Content-Type: application/json" \
  -d '{"test": "data"}'
```

### Docker Issues

```bash
# View logs
docker logs trufflehog-api

# Restart container
docker restart trufflehog-api

# Remove and recreate
docker-compose down
docker-compose up -d
```

## Performance Tuning

### Concurrent Scans

The API handles multiple scans concurrently. Adjust based on your resources:

```go
// In server.go, configure worker pool size
maxConcurrentScans := 10
```

### Resource Limits

Set appropriate limits in Docker:

```yaml
services:
  trufflehog-api:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 4G
        reservations:
          cpus: '1'
          memory: 2G
```

### Scan Timeouts

Configure timeouts for long-running scans:

```go
scanTimeout := 30 * time.Minute
```

## Backup and Recovery

### Scan Results

Consider implementing persistent storage:

1. **Database**: PostgreSQL, MySQL, MongoDB
2. **Object Storage**: S3, GCS, Azure Blob
3. **File System**: NFS, EFS

### Configuration Backup

Backup your configuration files:

```bash
# Backup
tar -czf trufflehog-backup.tar.gz \
  docker-compose.yml \
  Dockerfile.api \
  examples/

# Restore
tar -xzf trufflehog-backup.tar.gz
```

## Upgrading

### Docker

```bash
# Pull latest image
docker-compose pull

# Restart services
docker-compose up -d
```

### From Source

```bash
# Pull latest code
git pull origin main

# Rebuild
go build -o trufflehog-api ./cmd/api

# Restart service
systemctl restart trufflehog-api
```

## Support

For issues and questions:

1. Check the [API Documentation](./API.md)
2. Review [New Features](./NEW_FEATURES.md)
3. Open an issue on GitHub

## Next Steps

- [API Documentation](./API.md) - Complete API reference
- [New Features](./NEW_FEATURES.md) - Learn about new detectors
- [Examples](../examples/) - More code examples
