# TruffleHog 🐷🔑

<p align="center">
  <img src="https://raw.githubusercontent.com/trufflesecurity/trufflehog/main/assets/trufflehog-logo.png" alt="TruffleHog Logo" width="200"/>
</p>

<p align="center">
  <strong>Find, verify, and analyze leaked credentials in seconds</strong>
</p>

<p align="center">
  <a href="#features">Features</a> •
  <a href="#rest-api">REST API</a> •
  <a href="#installation">Installation</a> •
  <a href="#usage">Usage</a> •
  <a href="#custom-detectors">Custom Detectors</a> •
  <a href="#docker">Docker</a> •
  <a href="#documentation">Documentation</a>
</p>

---

## 🌟 What's New in This Fork

This fork extends the original TruffleHog with:

- ✅ **Production-Ready REST API** with JWT authentication
- ✅ **9 Custom AI Service Detectors** (Exa AI, FireCrawl, Perplexity, OpenRouter, Google Gemini, Google Veo, HeyGen, MidJourney, Runway ML)
- ✅ **Interactive Swagger Documentation** at `/swagger/`
- ✅ **Async Job Processing** with Redis queue
- ✅ **Docker Compose** for easy deployment
- ✅ **Webhook Notifications** for scan events
- ✅ **PostgreSQL Storage** for results
- ✅ **851 Total Detectors** (9 custom + 842 built-in)

**🔗 Live Demo:** Deploy your own instance with Docker  
**📚 API Docs:** Available at `/swagger/` when running

---

## Features

TruffleHog scans for secrets in:
- Git repositories (GitHub, GitLab, Bitbucket, etc.)
- Filesystems and directories
- S3 buckets
- Docker images
- CI/CD logs and more

**Key Capabilities:**
- 🔍 **851+ Secret Detectors** including custom AI services
- ✅ **Secret Verification** - Validates if credentials are active
- 🚀 **Fast Scanning** - Processes 10GB+ repos in minutes
- 🎯 **Low False Positives** - Smart filtering and validation
- 🔌 **Multiple Integrations** - GitHub Actions, GitLab CI, pre-commit hooks
- 🌐 **REST API** - Programmatic access via HTTP endpoints

---

## REST API

### Quick Start with API

```bash
# 1. Login to get JWT token
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.token')

# 2. Create a scan job
JOB=$(curl -s -X POST http://localhost:8080/api/v1/scan \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"repo_url":"https://github.com/user/repo"}')

# 3. Check scan status
JOB_ID=$(echo $JOB | jq -r '.job_id')
curl -s http://localhost:8080/api/v1/scan/$JOB_ID \
  -H "Authorization: Bearer $TOKEN" | jq .
```

### API Endpoints

#### Public Endpoints
- `GET /` - API information
- `GET /health` - Health check
- `GET /swagger/` - Interactive API documentation
- `GET /api/v1/detectors` - List all 851 detectors
- `POST /api/v1/auth/login` - User authentication
- `POST /api/v1/auth/register` - User registration

#### Protected Endpoints (Require JWT)
- `POST /api/v1/scan` - Create scan job
- `GET /api/v1/scan/:jobId` - Get scan status and results
- `DELETE /api/v1/scan/:jobId` - Cancel scan
- `POST /api/v1/webhooks` - Create webhook
- `GET /api/v1/webhooks` - List webhooks

**📖 Full API Documentation:** See [README_API.md](README_API.md) or visit `/swagger/` when running

---

## Installation

### Using Docker (Recommended)

```bash
# Clone the repository
git clone https://github.com/huuthangntk/trufflehog.git
cd trufflehog

# Start all services (API, PostgreSQL, Redis)
docker-compose up -d

# Check services
docker-compose ps

# View logs
docker-compose logs -f api
```

The API will be available at `http://localhost:8080`

### Using Go

```bash
# Install TruffleHog CLI
go install github.com/trufflesecurity/trufflehog/v3@latest

# Or build from source
git clone https://github.com/huuthangntk/trufflehog.git
cd trufflehog
go build
./trufflehog --version
```

### Using Pre-built Binaries

Download the latest release from the [releases page](https://github.com/trufflesecurity/trufflehog/releases).

---

## Usage

### CLI Usage

#### Scan a Git Repository

```bash
# Scan a GitHub repository
trufflehog git https://github.com/user/repo

# Scan with verification
trufflehog git https://github.com/user/repo --only-verified

# Scan specific branch
trufflehog git https://github.com/user/repo --branch main

# Output JSON
trufflehog git https://github.com/user/repo --json
```

#### Scan Filesystem

```bash
# Scan current directory
trufflehog filesystem .

# Scan specific path
trufflehog filesystem /path/to/scan

# Exclude paths
trufflehog filesystem . --exclude-paths exclude-patterns.txt
```

#### Scan S3 Bucket

```bash
# Scan specific bucket
trufflehog s3 --bucket=my-bucket

# Scan with IAM role
trufflehog s3 --bucket=my-bucket --role-arn=arn:aws:iam::123456789012:role/MyRole
```

#### Scan Docker Image

```bash
trufflehog docker --image trufflesecurity/secrets
```

### API Usage

#### JavaScript/Node.js Example

```javascript
const axios = require('axios');

const API_BASE = 'http://localhost:8080';

// Login
const loginResponse = await axios.post(`${API_BASE}/api/v1/auth/login`, {
  username: 'admin',
  password: 'admin123'
});
const token = loginResponse.data.token;

// Create scan
const scanResponse = await axios.post(
  `${API_BASE}/api/v1/scan`,
  {
    repo_url: 'https://github.com/user/repo',
    only_verified: true
  },
  {
    headers: { Authorization: `Bearer ${token}` }
  }
);
const jobId = scanResponse.data.job_id;

// Check status
const statusResponse = await axios.get(
  `${API_BASE}/api/v1/scan/${jobId}`,
  {
    headers: { Authorization: `Bearer ${token}` }
  }
);
console.log('Scan status:', statusResponse.data);
```

#### Python Example

```python
import requests

API_BASE = 'http://localhost:8080'

# Login
login_response = requests.post(
    f'{API_BASE}/api/v1/auth/login',
    json={'username': 'admin', 'password': 'admin123'}
)
token = login_response.json()['token']

# Create scan
scan_response = requests.post(
    f'{API_BASE}/api/v1/scan',
    json={'repo_url': 'https://github.com/user/repo', 'only_verified': True},
    headers={'Authorization': f'Bearer {token}'}
)
job_id = scan_response.json()['job_id']

# Check status
status_response = requests.get(
    f'{API_BASE}/api/v1/scan/{job_id}',
    headers={'Authorization': f'Bearer {token}'}
)
print('Scan status:', status_response.json())
```

---

## Custom Detectors

This fork includes 9 additional custom AI service detectors:

| Service | Detector | Verification |
|---------|----------|--------------|
| **Exa AI** | ✅ | ✅ |
| **FireCrawl** | ✅ | ✅ |
| **Perplexity** | ✅ | ✅ |
| **OpenRouter** | ✅ | ✅ |
| **Google Gemini** | ✅ | ✅ |
| **Google Veo** | ✅ | ✅ |
| **HeyGen** | ✅ | ✅ |
| **MidJourney** | ✅ | ⚠️ (No public API) |
| **Runway ML** | ✅ | ✅ |

**Total Detectors:** 851 (9 custom + 842 built-in)

### List All Detectors

```bash
# Via CLI
trufflehog dev

# Via API
curl http://localhost:8080/api/v1/detectors | jq '.detectors[] | select(.name | contains("exa"))'
```

---

## Docker

### Docker Compose

The `docker-compose.yml` includes:
- **TruffleHog API** (port 8080)
- **PostgreSQL** (for storing scan results)
- **Redis** (for job queue)

```bash
# Start services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down

# Rebuild after changes
docker-compose up -d --build
```

### Environment Variables

Create a `.env` file:

```bash
# API Configuration
API_PORT=8080
API_WORKERS=4

# Database
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_DB=trufflehog
POSTGRES_USER=trufflehog
POSTGRES_PASSWORD=your_secure_password

# Redis
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=your_redis_password

# JWT
JWT_SECRET=your_jwt_secret_min_32_chars

# Scanning
MAX_CONCURRENT_SCANS=5
SCAN_TIMEOUT=3600
```

---

## GitHub Actions

Use TruffleHog in your CI/CD pipeline:

```yaml
name: Secret Scan

on: [push, pull_request]

jobs:
  trufflehog:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0
      
      - name: TruffleHog Secret Scan
        uses: trufflesecurity/trufflehog@main
        with:
          extra_args: --only-verified --results=verified,unknown
```

---

## Pre-commit Hook

Prevent secrets from being committed:

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/trufflesecurity/trufflehog
    rev: main
    hooks:
      - id: trufflehog
        name: TruffleHog
        entry: trufflehog filesystem
        language: system
        pass_filenames: false
        args: ['--only-verified', '.']
```

Install: `pre-commit install`

---

## Documentation

- **API Documentation:** [README_API.md](README_API.md)
- **Interactive API Docs:** Available at `/swagger/` when running the API
- **Original TruffleHog Docs:** [https://github.com/trufflesecurity/trufflehog](https://github.com/trufflesecurity/trufflehog)

---

## Architecture

### REST API Architecture

```
┌─────────────┐     ┌──────────────┐     ┌────────────┐
│   Nginx     │────▶│   API Server │────▶│ PostgreSQL │
│ (SSL/HTTPS) │     │  (Port 8080) │     │            │
└─────────────┘     └──────────────┘     └────────────┘
                           │
                           ▼
                    ┌────────────┐     ┌────────────┐
                    │   Redis    │────▶│  Workers   │
                    │   Queue    │     │  (Scans)   │
                    └────────────┘     └────────────┘
```

---

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Adding New Detectors

See the [detector development guide](https://github.com/trufflesecurity/trufflehog/blob/main/docs/adding_detectors.md).

---

## Security

TruffleHog is a security tool. If you find a security vulnerability, please report it responsibly:

- **Email:** security@trufflesec.com
- **GitHub Security Advisory:** Use the Security tab

**Default Credentials:**
- Username: `admin`
- Password: `admin123`

⚠️ **IMPORTANT:** Change the default admin password immediately after deployment!

---

## License

This project is licensed under the **AGPL-3.0 License** - see the [LICENSE](LICENSE) file for details.

---

## Acknowledgments

- Original TruffleHog: [TruffleSecurity](https://github.com/trufflesecurity/trufflehog)
- Built with ❤️ using Go, Fiber, PostgreSQL, and Redis
- Custom AI detectors for modern security scanning

---

## Support

- **Issues:** [GitHub Issues](https://github.com/huuthangntk/trufflehog/issues)
- **Discussions:** [GitHub Discussions](https://github.com/trufflesecurity/trufflehog/discussions)
- **API Support:** Check the `/swagger/` endpoint when running the API

---

<p align="center">
  Made with 🐷 by the TruffleHog community
</p>

