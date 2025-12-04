# New Features

## New API Key Detectors

This fork adds support for detecting and verifying the following API keys:

### 1. Perplexity AI
- **Pattern**: `pplx-[a-zA-Z0-9]{32,}`
- **Verification**: Tests against Perplexity API `/models` endpoint
- **Use Case**: Conversational AI and search capabilities

### 2. ElevenLabs
- **Pattern**: 32-character hexadecimal strings
- **Verification**: Tests against ElevenLabs API `/v1/user` endpoint
- **Use Case**: AI voice generation and text-to-speech services

### 3. OpenRouter
- **Pattern**: `sk-or-v1-[a-zA-Z0-9]{64,}`
- **Verification**: Tests against OpenRouter API `/api/v1/auth/key` endpoint
- **Use Case**: Unified access to multiple AI models

### 4. RunwayML
- **Pattern**: `rw_[a-zA-Z0-9]{32,}`
- **Verification**: Tests against RunwayML API `/v1/user` endpoint
- **Use Case**: AI-powered creative tools for video generation

### 5. Firecrawl
- **Pattern**: `fc-[a-zA-Z0-9]{32,}`
- **Verification**: Tests against Firecrawl API crawl status endpoint
- **Use Case**: Web scraping and crawling services

### 6. Exa Search
- **Pattern**: UUID format (8-4-4-4-12 hexadecimal)
- **Verification**: Tests against Exa API `/search` endpoint
- **Use Case**: AI-powered semantic search engine

## REST API Service

A new REST API service has been added to allow using TruffleHog without the binary. This enables integration with any programming language or platform.

### Key Features

1. **Asynchronous Scanning**: Submit repository URLs for scanning and receive results via webhook
2. **Webhook Notifications**: Get notified when scans complete
3. **Status Tracking**: Query scan status and results at any time
4. **Multi-language Support**: Use from any language that can make HTTP requests
5. **Docker Support**: Easy deployment with Docker and Docker Compose

### Quick Start

#### Using Docker Compose
```bash
docker-compose up -d
```

#### Using Go
```bash
go run ./cmd/api/main.go -addr :8080
```

### API Endpoints

- `POST /api/v1/scan` - Initiate a new scan
- `GET /api/v1/scan/status` - Get scan status and results
- `GET /api/v1/scans` - List all scans
- `GET /health` - Health check

### Example Usage

```bash
# Start a scan
curl -X POST http://localhost:8080/api/v1/scan \
  -H "Content-Type: application/json" \
  -d '{
    "repo_url": "https://github.com/example/repo.git",
    "webhook_url": "https://your-webhook.com/endpoint",
    "verify": true
  }'

# Check scan status
curl http://localhost:8080/api/v1/scan/status?scan_id=YOUR_SCAN_ID
```

## Webhook Integration

When a scan completes, the API sends a POST request to your webhook URL with the complete results:

```json
{
  "event": "scan.completed",
  "scan_result": {
    "scan_id": "...",
    "status": "completed",
    "total_secrets": 5,
    "verified": 3,
    "unverified": 2,
    "secrets": [...]
  },
  "timestamp": "2024-12-04T10:35:00Z"
}
```

## Architecture

```
┌─────────────┐
│   Client    │
│ (Any Lang)  │
└──────┬──────┘
       │ HTTP POST /api/v1/scan
       ▼
┌─────────────────┐
│  TruffleHog API │
│     Server      │
└────────┬────────┘
         │
         ├─► Scan Repository (async)
         │
         └─► Send Webhook Notification
                    │
                    ▼
            ┌──────────────┐
            │   Your App   │
            │   Webhook    │
            └──────────────┘
```

## Use Cases

1. **CI/CD Integration**: Scan repositories as part of your build pipeline
2. **Scheduled Scans**: Set up cron jobs to regularly scan repositories
3. **Multi-Repository Monitoring**: Scan multiple repositories and aggregate results
4. **Custom Workflows**: Build custom secret detection workflows in any language
5. **Webhook-based Alerting**: Get real-time notifications when secrets are found

## Configuration

The API server can be configured using command-line flags:

```bash
./trufflehog-api -addr :8080
```

### Environment Variables

- `API_PORT`: Port to listen on (default: 8080)

## Security Best Practices

1. **Use HTTPS**: Always use TLS/SSL in production
2. **Add Authentication**: Implement API key or OAuth2 authentication
3. **Verify Webhooks**: Implement webhook signature verification
4. **Rate Limiting**: Add rate limiting to prevent abuse
5. **Input Validation**: Validate all repository URLs
6. **Network Security**: Use firewalls and network policies

## Deployment Options

### Docker
```bash
docker build -f Dockerfile.api -t trufflehog-api .
docker run -p 8080:8080 trufflehog-api
```

### Kubernetes
```yaml
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
```

### Cloud Platforms

- **AWS**: Deploy using ECS, EKS, or Lambda
- **Google Cloud**: Deploy using Cloud Run, GKE, or App Engine
- **Azure**: Deploy using Container Instances, AKS, or App Service

## Performance Considerations

1. **Concurrent Scans**: The API handles multiple scans concurrently
2. **Resource Limits**: Set appropriate memory and CPU limits
3. **Scan Timeout**: Configure timeouts for long-running scans
4. **Result Storage**: Consider using a database for persistent storage
5. **Caching**: Implement caching for frequently scanned repositories

## Monitoring and Logging

- Health check endpoint: `/health`
- Structured logging for all operations
- Metrics for scan duration and success rates
- Webhook delivery status tracking

## Future Enhancements

Potential improvements for future versions:

1. Database backend for persistent storage
2. Authentication and authorization
3. Rate limiting and quotas
4. Scan scheduling and recurring scans
5. Result filtering and search
6. Email notifications
7. Slack/Discord integrations
8. Custom detector configuration
9. Scan history and analytics
10. Multi-tenancy support

## Documentation

- [API Documentation](./API.md) - Complete API reference
- [Detector Documentation](../pkg/detectors/README.md) - Detector implementation guide

## Contributing

Contributions are welcome! Please see the main CONTRIBUTING.md for guidelines.

## License

This project maintains the same license as the original TruffleHog project.
