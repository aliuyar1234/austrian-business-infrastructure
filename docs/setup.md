# Setup & Installation

## Prerequisites

- Go 1.21+
- PostgreSQL 15+
- Redis 7+
- Node.js 18+ (for frontend)
- Docker & Docker Compose (optional)

## Quick Start with Docker

The fastest way to get started:

```bash
# Clone the repository
git clone https://github.com/aliuyar1234/austrian-business-infrastructure.git
cd austrian-business-infrastructure

# Copy environment template
cp .env.example .env

# Edit .env with your settings (see Configuration guide)
nano .env

# Start all services
docker-compose up -d
```

The application will be available at:
- Frontend: http://localhost:3000
- API: http://localhost:8080
- Portal: http://localhost:3001

## Manual Installation

### 1. Database Setup

```bash
# Create PostgreSQL database
createdb austrian_business

# Run migrations
go run migrations/migrate.go up
```

### 2. Redis Setup

```bash
# Start Redis (or use Docker)
redis-server
```

### 3. Backend

```bash
# Install Go dependencies
go mod download

# Build the server
go build -o bin/server ./cmd/server

# Build the worker
go build -o bin/worker ./cmd/worker

# Run the server
./bin/server

# Run the worker (separate terminal)
./bin/worker
```

### 4. Frontend

```bash
cd frontend

# Install dependencies
npm install

# Development mode
npm run dev

# Production build
npm run build
npm run preview
```

## Verification

Check that everything is running:

```bash
# API health check
curl http://localhost:8080/api/v1/health

# Expected response:
# {"status":"healthy","version":"1.0.0"}
```

## Next Steps

1. [Configure environment variables](configuration.md)
2. Set up your first [FinanzOnline account](modules/finanzonline.md)
3. Explore the [API Reference](api-reference.md)
