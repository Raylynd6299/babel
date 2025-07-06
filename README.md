# 🌍 Polyfy - Babel Backend

<div align="center">

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-2496ED?style=for-the-badge&logo=docker&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-316192?style=for-the-badge&logo=postgresql&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-DC382D?style=for-the-badge&logo=redis&logoColor=white)
![Swagger](https://img.shields.io/badge/Swagger-85EA2D?style=for-the-badge&logo=swagger&logoColor=black)

**Microservices-based language learning platform backend**

[🚀 Quick Start](#-quick-start) • [📚 Documentation](#-api-documentation) • [🏗️ Architecture](#️-architecture) • [🔧 Development](#-development)

</div>

---

## 📋 Table of Contents

- [🌟 Overview](#-overview)
- [🏗️ Architecture](#️-architecture)
- [🚀 Quick Start](#-quick-start)
- [📚 API Documentation](#-api-documentation)
- [🔧 Development](#-development)
- [🐳 Docker Services](#-docker-services)
- [📁 Project Structure](#-project-structure)
- [🔌 Service Endpoints](#-service-endpoints)
- [🛠️ Troubleshooting](#️-troubleshooting)

---

## 🌟 Overview

**Polyfy Babel** is a comprehensive microservices-based backend for a language learning platform. Built with Go and following clean architecture principles, it provides a scalable, maintainable, and robust foundation for language learning applications.

### ✨ Key Features

- 🎯 **7 Specialized Microservices** - Each handling specific domain logic
- 🔐 **JWT Authentication** - Secure user authentication and authorization
- 📊 **Real-time Analytics** - Progress tracking and learning insights
- 🎮 **Gamification** - Points, achievements, and leaderboards
- 👥 **Social Features** - User interactions, groups, and mentorship
- 🔊 **Phonetic Training** - Advanced pronunciation practice
- 📚 **Smart Vocabulary** - SRS-based vocabulary learning
- 🎬 **Rich Content** - Multi-media learning materials
- 📈 **Progress Tracking** - Detailed learning analytics

---

## 🏗️ Architecture

### Microservices Overview

```mermaid
graph TB
    Client[Client Applications] --> Gateway[API Gateway :8080]
    
    Gateway --> Auth[Auth Service :8001]
    Gateway --> Content[Content Service :8002]
    Gateway --> Progress[Progress Service :8003]
    Gateway --> Vocab[Vocabulary Service :8004]
    Gateway --> Phonetic[Phonetic Service :8005]
    Gateway --> Social[Social Service :8006]
    
    Auth --> DB[(PostgreSQL)]
    Content --> DB
    Progress --> DB
    Vocab --> DB
    Phonetic --> DB
    Social --> DB
    
    Progress --> InfluxDB[(InfluxDB)]
    Content --> MinIO[(MinIO)]
    
    Auth --> Redis[(Redis)]
    Vocab --> Redis
    Social --> Redis
```

### Service Responsibilities

| Service | Port | Purpose | Key Features |
|---------|------|---------|--------------|
| **API Gateway** | 8080 | Request routing & authentication | Rate limiting, CORS, JWT validation |
| **Auth Service** | 8001 | User authentication & management | Registration, login, JWT tokens, user profiles |
| **Content Service** | 8002 | Learning content management | Videos, podcasts, articles, episodes, ratings |
| **Progress Service** | 8003 | Learning analytics & tracking | Study sessions, streaks, statistics, calendar |
| **Vocabulary Service** | 8004 | Vocabulary & SRS system | Word management, spaced repetition, reviews |
| **Phonetic Service** | 8005 | Pronunciation training | Phoneme practice, pronunciation exercises |
| **Social Service** | 8006 | Social features & community | Profiles, follows, groups, activities, mentorship |

---

## 🚀 Quick Start

### Prerequisites

- **Docker** & **Docker Compose** installed
- **Go 1.21+** (for local development)
- **Git** for version control

### 1. Clone & Setup

```bash
# Clone the repository
git clone <repository-url>
cd babel

# Copy environment configuration
cp script/.env.example script/.env
```

### 2. Start All Services

```bash
# Navigate to scripts directory
cd script

# Start all services with Docker Compose
docker-compose up -d

# Check services status
docker-compose ps
```

### 3. Verify Installation

```bash
# Check API Gateway health
curl http://localhost:8080/health

# View service logs
docker-compose logs -f auth-service
```

### 4. Access Documentation

- **API Gateway**: http://localhost:8080/swagger/index.html
- **Auth Service**: http://localhost:8001/swagger/index.html
- **Content Service**: http://localhost:8002/swagger/index.html
- **Progress Service**: http://localhost:8003/swagger/index.html
- **Vocabulary Service**: http://localhost:8004/swagger/index.html
- **Phonetic Service**: http://localhost:8005/swagger/index.html
- **Social Service**: http://localhost:8006/swagger/index.html

---

## 📚 API Documentation

All services provide comprehensive Swagger/OpenAPI documentation with interactive testing capabilities.

### 🔐 Authentication

Most endpoints require JWT authentication. Include the token in the Authorization header:

```bash
Authorization: Bearer <your-jwt-token>
```

### 📖 Service Documentation

Each service provides detailed API documentation accessible via Swagger UI:

| Service | Swagger URL | Endpoints | Status |
|---------|-------------|-----------|--------|
| Auth | http://localhost:8001/swagger/ | 6 endpoints | ✅ Complete |
| Content | http://localhost:8002/swagger/ | 12 endpoints | ✅ Complete |
| Progress | http://localhost:8003/swagger/ | 12 endpoints | ✅ Complete |
| Vocabulary | http://localhost:8004/swagger/ | 23 endpoints | ✅ Complete |
| Phonetic | http://localhost:8005/swagger/ | 16 endpoints | ✅ Complete |
| Social | http://localhost:8006/swagger/ | 32 endpoints | ✅ Complete |

---

## 🔧 Development

### Local Development Setup

```bash
# Start only infrastructure services
docker-compose -f docker-compose.dev.yml up -d postgres redis influxdb minio

# Run a service locally
cd cmd/auth-service
go run main.go

# Run tests
go test ./...

# Generate Swagger docs
swag init --generalInfo main.go --dir .,../../internal/auth --output ./docs
```

### Environment Variables

Key environment variables (see `script/.env`):

```env
# Database
DATABASE_URL=postgres://postgres:password@postgres:5432/polyfy
REDIS_URL=redis://redis:6379

# JWT
JWT_SECRET=your-secret-key

# Service URLs
AUTH_SERVICE_URL=http://auth-service:8001
CONTENT_SERVICE_URL=http://content-service:8002
# ... other services
```

### Code Generation

```bash
# Generate Swagger documentation for all services
make swagger

# Build all services
make build

# Run all tests
make test
```

---

## 🐳 Docker Services

### Infrastructure Services

| Service | Port | Purpose | Health Check |
|---------|------|---------|--------------|
| **PostgreSQL** | 5432 | Primary database | `pg_isready` |
| **Redis** | 6379 | Caching & sessions | `redis-cli ping` |
| **InfluxDB** | 8086 | Time-series metrics | HTTP health endpoint |
| **MinIO** | 9000/9001 | File storage | S3 API health |

### Application Services

All application services include:
- Health check endpoints (`/health`)
- Swagger documentation (`/swagger/`)
- Structured logging
- Graceful shutdown
- Auto-restart on failure

### Docker Commands

```bash
# View all services
docker-compose ps

# View logs
docker-compose logs -f [service-name]

# Restart a service
docker-compose restart [service-name]

# Scale a service
docker-compose up -d --scale auth-service=3

# Stop all services
docker-compose down

# Clean up (remove volumes)
docker-compose down -v
```

---

## 📁 Project Structure

```
babel/
├── 📁 cmd/                          # Service entry points
│   ├── 🔐 auth-service/             # Authentication service
│   ├── 📚 content-service/          # Content management
│   ├── 📊 progress-service/         # Learning analytics
│   ├── 📖 vocabulary-service/       # Vocabulary & SRS
│   ├── 🔊 phonetic-service/         # Pronunciation training
│   ├── 👥 social-service/           # Social features
│   └── 🌐 api-gateway/              # API Gateway
├── 📁 internal/                     # Private application code
│   ├── 🔐 auth/                     # Auth domain logic
│   ├── 📚 content/                  # Content domain logic
│   ├── 📊 progress/                 # Progress domain logic
│   ├── 📖 vocabulary/               # Vocabulary domain logic
│   ├── 🔊 phonetic/                 # Phonetic domain logic
│   ├── 👥 social/                   # Social domain logic
│   └── 🔧 shared/                   # Shared utilities
│       ├── config/                  # Configuration management
│       ├── database/                # Database connections
│       └── middleware/              # HTTP middleware
├── 📁 pkg/                          # Public library code
│   ├── jwt/                         # JWT utilities
│   ├── logger/                      # Logging utilities
│   └── gateway/                     # Gateway utilities
├── 📁 docker/                       # Docker configurations
├── 📁 script/                       # Deployment scripts
│   ├── .env                         # Environment variables
│   ├── docker-compose.yml          # Production compose
│   └── docker-compose.dev.yml      # Development compose
├── 📁 migrations/                   # Database migrations
├── 📄 go.mod                        # Go module definition
└── 📄 README.md                     # This file
```

---

## 🔌 Service Endpoints

### 🔐 Auth Service (Port 8001)
```
POST   /api/v1/auth/register         # User registration
POST   /api/v1/auth/login            # User login
POST   /api/v1/auth/refresh          # Token refresh
POST   /api/v1/auth/logout           # User logout
POST   /api/v1/auth/forgot-password  # Password reset request
POST   /api/v1/auth/reset-password   # Password reset
```

### 📚 Content Service (Port 8002)
```
GET    /api/v1/content               # List content
POST   /api/v1/content               # Create content
GET    /api/v1/content/{id}          # Get content details
PUT    /api/v1/content/{id}          # Update content
DELETE /api/v1/content/{id}          # Delete content
POST   /api/v1/content/{id}/rate     # Rate content
GET    /api/v1/content/{id}/episodes # Get episodes
POST   /api/v1/content/{id}/episodes # Create episode
GET    /api/v1/content/recommendations # Get recommendations
GET    /api/v1/content/languages     # Get supported languages
```

### 📊 Progress Service (Port 8003)
```
POST   /api/v1/progress/input        # Log study session
GET    /api/v1/progress/stats        # Get progress stats
GET    /api/v1/progress/analytics    # Get detailed analytics
GET    /api/v1/progress/recent       # Get recent activities
GET    /api/v1/progress/calendar     # Get calendar data
```

### 📖 Vocabulary Service (Port 8004)
```
POST   /api/v1/vocabulary            # Add vocabulary
GET    /api/v1/vocabulary            # Get user vocabulary
PUT    /api/v1/vocabulary/{id}       # Update vocabulary
DELETE /api/v1/vocabulary/{id}       # Delete vocabulary
GET    /api/v1/vocabulary/reviews    # Get words for review
POST   /api/v1/vocabulary/reviews    # Submit review
GET    /api/v1/vocabulary/stats      # Get vocabulary stats
GET    /api/v1/vocabulary/search     # Search vocabulary
POST   /api/v1/vocabulary/import     # Import vocabulary
GET    /api/v1/vocabulary/export     # Export vocabulary
```

### 🔊 Phonetic Service (Port 8005)
```
GET    /api/v1/phonetic/languages/{id}/phonemes # Get phonemes
GET    /api/v1/phonetic/progress     # Get phonetic progress
POST   /api/v1/phonetic/practice     # Submit practice session
GET    /api/v1/phonetic/exercises    # Get exercises
```

### 👥 Social Service (Port 8006)
```
GET    /api/v1/social/profile        # Get my profile
PUT    /api/v1/social/profile        # Update profile
POST   /api/v1/social/follow         # Follow user
DELETE /api/v1/social/follow/{id}    # Unfollow user
GET    /api/v1/social/feed           # Get activity feed
POST   /api/v1/social/activities     # Create activity
GET    /api/v1/social/groups         # Get study groups
POST   /api/v1/social/groups         # Create group
```

---

## 🛠️ Troubleshooting

### Common Issues

#### 🔴 Service Won't Start
```bash
# Check logs
docker-compose logs [service-name]

# Check if port is in use
netstat -tulpn | grep [port]

# Restart service
docker-compose restart [service-name]
```

#### 🔴 Database Connection Issues
```bash
# Check PostgreSQL status
docker-compose exec postgres pg_isready

# Check database logs
docker-compose logs postgres

# Reset database
docker-compose down -v
docker-compose up -d postgres
```

#### 🔴 Swagger UI Not Loading
```bash
# Regenerate Swagger docs
cd cmd/[service-name]
swag init --generalInfo main.go --dir .,../../internal/[service] --output ./docs

# Check service logs
docker-compose logs [service-name]
```

#### 🔴 JWT Authentication Errors
```bash
# Check JWT secret in environment
echo $JWT_SECRET

# Verify token format
curl -H "Authorization: Bearer <token>" http://localhost:8001/api/v1/users/me
```

### Performance Monitoring

```bash
# Monitor resource usage
docker stats

# Check service health
curl http://localhost:8080/health
curl http://localhost:8001/health
# ... for each service

# View detailed logs
docker-compose logs -f --tail=100
```

### Development Tips

1. **Hot Reload**: Use `air` for Go hot reload during development
2. **Database Migrations**: Run migrations manually during development
3. **Testing**: Use separate test database for integration tests
4. **Debugging**: Enable debug mode with `GIN_MODE=debug`

---

## 📞 Support

For issues, questions, or contributions:

1. Check the [troubleshooting section](#️-troubleshooting)
2. Review service logs: `docker-compose logs [service-name]`
3. Verify environment configuration in `script/.env`
4. Test individual services via Swagger UI

---

<div align="center">

**Built with ❤️ for language learners worldwide**

[🔝 Back to Top](#-polyfy---babel-backend)

</div>
