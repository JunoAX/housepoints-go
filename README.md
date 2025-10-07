# HousePoints Go - Multi-Tenant Family Management System

[![Build Status](https://github.com/JunoAX/housepoints-go/workflows/CI/badge.svg)](https://github.com/JunoAX/housepoints-go/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/JunoAX/housepoints-go)](https://goreportcard.com/report/github.com/JunoAX/housepoints-go)

Modern, scalable Go backend for HousePoints - designed to support 10,000+ families.

## 🎯 Project Goals

- **Multi-tenant architecture**: Isolated data for each family
- **High performance**: 3-4x better resource efficiency than Python
- **Scalability**: Single server → 10,000 families
- **Cost-effective**: 80% reduction in hosting costs

## 🏗️ Architecture

### Single Server Start (0-500 families)
- Monolithic Go service
- PostgreSQL with Row-Level Security
- Redis for caching/sessions
- Kubernetes HPA for auto-scaling

### Scaling Path
1. **Stage 1**: Single pod (1-500 families)
2. **Stage 2**: Load balanced (500-2,000 families)
3. **Stage 3**: Service split (2,000-5,000 families)
4. **Stage 4**: Microservices (5,000-10,000 families)

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- Docker & Docker Compose
- PostgreSQL 14+
- Redis 7+

### Local Development

```bash
# Clone repository
git clone https://github.com/JunoAX/housepoints-go
cd housepoints-go

# Install dependencies
go mod download

# Setup environment
cp .env.example .env

# Start dependencies
docker-compose up -d

# Run migrations
make migrate-up

# Start server
make run

# Server runs on http://localhost:8080
```

### Testing

```bash
# Run all tests
make test

# Run integration tests
make test-integration

# Run with coverage
make test-coverage
```

## 📂 Project Structure

```
housepoints-go/
├── cmd/
│   └── server/              # Application entry point
├── internal/
│   ├── api/                 # HTTP handlers
│   ├── domain/              # Business logic
│   ├── repository/          # Data access
│   └── config/              # Configuration
├── pkg/                     # Reusable packages
├── migrations/              # Database migrations
├── deployments/             # Kubernetes manifests
└── docs/                    # Documentation
```

## 🔄 Migration from Python

See [docs/MIGRATION.md](docs/MIGRATION.md) for detailed migration strategy.

**Current Status**: Phase 0 - Foundation
- ✅ Repository setup
- ✅ Project structure
- ⏳ Core services implementation
- ⏳ Multi-tenant database schema

## 🚢 Deployment

### Kubernetes

```bash
# Build and push image
make docker-build
make docker-push

# Deploy to production
kubectl apply -k deployments/kubernetes/overlays/production
```

## 📊 Performance Targets

| Metric | Target | Current |
|--------|--------|---------|
| API Latency (p95) | < 100ms | TBD |
| Throughput | 10k req/s | TBD |
| Memory per family | < 5MB | TBD |
| Concurrent families | 500 per pod | TBD |

## 🛠️ Development

### Available Make Commands

```bash
make run              # Run server locally
make test             # Run all tests
make build            # Build binary
make docker-build     # Build Docker image
make migrate-up       # Run migrations
make migrate-down     # Rollback migrations
make lint             # Run linters
make fmt              # Format code
```

## 📝 Documentation

- [Architecture Overview](docs/ARCHITECTURE.md)
- [Migration Guide](docs/MIGRATION.md)
- [API Documentation](docs/API.md)
- [Database Schema](docs/DATABASE.md)
- [Deployment Guide](docs/DEPLOYMENT.md)

## 🔗 Links

- [Python Backend](https://github.com/JunoAX/housepoints) - Original implementation
- [Migration Plan](https://github.com/JunoAX/housepoints/blob/main/docs/GO_MIGRATION_PLAN.md)

## 📜 License

MIT License - See LICENSE file for details.
