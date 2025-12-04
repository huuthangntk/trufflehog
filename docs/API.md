# TruffleHog REST API Documentation

## Overview

The TruffleHog REST API provides a programmatic interface to scan repositories for secrets without requiring the binary. It supports webhook notifications for asynchronous scan completion.

## Base URL

```
http://localhost:8080/api/v1
```

## Authentication

Currently, the API does not require authentication. For production deployments, consider adding API key authentication or OAuth2.

## Endpoints

### 1. Initiate Scan

Starts a new repository scan.

**Endpoint:** `POST /api/v1/scan`

**Request Body:**
```json
{
  "repo_url": "https://github.com/username/repository.git",
  "webhook_url": "https://your-webhook-endpoint.com/webhook",
  "verify": true,
  "include_only": ["OpenAI", "AWS", "Perplexity"]
}
```

**Parameters:**
- `repo_url` (required): Git repository URL to scan
- `webhook_url` (optional): URL to receive webhook notification when scan completes
- `verify` (optional): Whether to verify found secrets (default: false)
- `include_only` (optional): Array of detector names to limit scanning

**Response:** `202 Accepted`
```json
{
  "scan_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "pending",
  "message": "Scan initiated successfully",
  "created_at": "2024-12-04T10:30:00Z"
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/v1/scan \
  -H "Content-Type: application/json" \
  -d '{
    "repo_url": "https://github.com/example/repo.git",
    "webhook_url": "https://webhook.site/unique-id",
    "verify": true
  }'
```

### 2. Get Scan Status

Retrieves the status and results of a specific scan.

**Endpoint:** `GET /api/v1/scan/status?scan_id={scan_id}`

**Parameters:**
- `scan_id` (required): The scan ID returned from the initiate scan endpoint

**Response:** `200 OK`
```json
{
  "scan_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "completed",
  "repo_url": "https://github.com/example/repo.git",
  "started_at": "2024-12-04T10:30:00Z",
  "completed_at": "2024-12-04T10:35:00Z",
  "total_secrets": 5,
  "verified": 3,
  "unverified": 2,
  "secrets": [
    {
      "detector_type": "OpenAI",
      "detector_name": "",
      "verified": true,
      "redacted": "sk-...FJ47",
      "extra_data": {
        "id": "user-123",
        "total_orgs": "2"
      },
      "source_name": "example/repo",
      "source_type": "git"
    }
  ]
}
```

**Status Values:**
- `pending`: Scan is queued
- `running`: Scan is in progress
- `completed`: Scan finished successfully
- `failed`: Scan encountered an error

**Example:**
```bash
curl http://localhost:8080/api/v1/scan/status?scan_id=550e8400-e29b-41d4-a716-446655440000
```

### 3. List All Scans

Retrieves a list of all scans.

**Endpoint:** `GET /api/v1/scans`

**Response:** `200 OK`
```json
{
  "scans": [
    {
      "scan_id": "550e8400-e29b-41d4-a716-446655440000",
      "status": "completed",
      "repo_url": "https://github.com/example/repo.git",
      "started_at": "2024-12-04T10:30:00Z",
      "completed_at": "2024-12-04T10:35:00Z",
      "total_secrets": 5,
      "verified": 3,
      "unverified": 2
    }
  ],
  "total": 1
}
```

**Example:**
```bash
curl http://localhost:8080/api/v1/scans
```

### 4. Health Check

Checks if the API server is running.

**Endpoint:** `GET /health`

**Response:** `200 OK`
```json
{
  "status": "healthy"
}
```

**Example:**
```bash
curl http://localhost:8080/health
```

## Webhook Notifications

When a `webhook_url` is provided in the scan request, the API will send a POST request to that URL when the scan completes.

### Webhook Payload

```json
{
  "event": "scan.completed",
  "scan_result": {
    "scan_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "completed",
    "repo_url": "https://github.com/example/repo.git",
    "started_at": "2024-12-04T10:30:00Z",
    "completed_at": "2024-12-04T10:35:00Z",
    "total_secrets": 5,
    "verified": 3,
    "unverified": 2,
    "secrets": [...]
  },
  "timestamp": "2024-12-04T10:35:00Z"
}
```

### Webhook Headers

- `Content-Type: application/json`
- `User-Agent: TruffleHog-API/1.0`
- `X-TruffleHog-Event: scan.completed`

### Webhook Example Handler (Node.js/Express)

```javascript
app.post('/webhook', (req, res) => {
  const { event, scan_result, timestamp } = req.body;
  
  if (event === 'scan.completed') {
    console.log(`Scan ${scan_result.scan_id} completed`);
    console.log(`Found ${scan_result.total_secrets} secrets`);
    console.log(`Verified: ${scan_result.verified}, Unverified: ${scan_result.unverified}`);
    
    // Process the results
    scan_result.secrets.forEach(secret => {
      if (secret.verified) {
        console.log(`⚠️  Verified ${secret.detector_type} secret found!`);
      }
    });
  }
  
  res.status(200).send('OK');
});
```

## Deployment

### Using Docker Compose

```bash
docker-compose up -d
```

### Using Docker

```bash
docker build -f Dockerfile.api -t trufflehog-api .
docker run -p 8080:8080 trufflehog-api
```

### Using Go

```bash
go run ./cmd/api/main.go -addr :8080
```

## New Detectors

This version includes support for the following new API key detectors:

1. **Perplexity** - AI search and conversational AI
2. **ElevenLabs** - AI voice generation and text-to-speech
3. **OpenRouter** - Unified access to multiple AI models
4. **RunwayML** - AI-powered creative tools for video
5. **Firecrawl** - Web scraping and crawling services
6. **Exa** - AI-powered semantic search

## Error Handling

All endpoints return appropriate HTTP status codes:

- `200 OK`: Request successful
- `202 Accepted`: Scan initiated
- `400 Bad Request`: Invalid request parameters
- `404 Not Found`: Resource not found
- `405 Method Not Allowed`: Invalid HTTP method
- `500 Internal Server Error`: Server error

Error responses include a message:
```json
{
  "error": "Description of the error"
}
```

## Rate Limiting

Currently, there is no rate limiting. For production use, consider implementing rate limiting based on your requirements.

## Security Considerations

1. **Authentication**: Add API key or OAuth2 authentication
2. **HTTPS**: Use TLS/SSL in production
3. **Webhook Verification**: Implement webhook signature verification
4. **Input Validation**: Validate all repository URLs
5. **Resource Limits**: Set limits on concurrent scans
6. **Secrets Storage**: Never store raw secrets, only redacted versions

## Examples

### Python Client

```python
import requests
import time

# Initiate scan
response = requests.post('http://localhost:8080/api/v1/scan', json={
    'repo_url': 'https://github.com/example/repo.git',
    'webhook_url': 'https://webhook.site/unique-id',
    'verify': True
})

scan_id = response.json()['scan_id']
print(f"Scan initiated: {scan_id}")

# Poll for results
while True:
    response = requests.get(f'http://localhost:8080/api/v1/scan/status?scan_id={scan_id}')
    result = response.json()
    
    if result['status'] in ['completed', 'failed']:
        print(f"Scan {result['status']}")
        print(f"Total secrets: {result['total_secrets']}")
        print(f"Verified: {result['verified']}")
        break
    
    time.sleep(5)
```

### JavaScript/Node.js Client

```javascript
const axios = require('axios');

async function scanRepository(repoUrl, webhookUrl) {
  try {
    // Initiate scan
    const response = await axios.post('http://localhost:8080/api/v1/scan', {
      repo_url: repoUrl,
      webhook_url: webhookUrl,
      verify: true
    });
    
    const scanId = response.data.scan_id;
    console.log(`Scan initiated: ${scanId}`);
    
    // Poll for results
    while (true) {
      const statusResponse = await axios.get(
        `http://localhost:8080/api/v1/scan/status?scan_id=${scanId}`
      );
      
      const result = statusResponse.data;
      
      if (result.status === 'completed' || result.status === 'failed') {
        console.log(`Scan ${result.status}`);
        console.log(`Total secrets: ${result.total_secrets}`);
        console.log(`Verified: ${result.verified}`);
        break;
      }
      
      await new Promise(resolve => setTimeout(resolve, 5000));
    }
  } catch (error) {
    console.error('Error:', error.message);
  }
}

scanRepository('https://github.com/example/repo.git', 'https://webhook.site/unique-id');
```

## Support

For issues and questions, please open an issue on the GitHub repository.
