<div align="center">

# Austrian Business Infrastructure

**Enterprise-grade Go toolkit for Austrian government and business API integrations**

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-AGPL--3.0-blue.svg)](LICENSE)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-336791?style=flat&logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7+-DC382D?style=flat&logo=redis&logoColor=white)](https://redis.io/)

[Features](#features) • [Quick Start](#quick-start) • [Documentation](#documentation) • [Security](#security) • [Contributing](#contributing)

</div>

---

## Overview

A production-ready platform for integrating with Austrian government services and business APIs. Built for tax advisors, accountants, and enterprises handling Austrian tax filings, employee registrations, and financial documents.

```
┌─────────────────────────────────────────────────────────────────┐
│                    Austrian Business Infrastructure              │
├─────────────────────────────────────────────────────────────────┤
│  CLI Tool    │  REST API    │  Client Portal  │  MCP Server    │
├─────────────────────────────────────────────────────────────────┤
│  FinanzOnline  │  ELDA  │  Firmenbuch  │  E-Rechnung  │  SEPA  │
└─────────────────────────────────────────────────────────────────┘
```

## Features

| Module | Description |
|--------|-------------|
| **FinanzOnline** | Session management, databox polling, UVA/ZM tax submissions |
| **ELDA** | Employee registration/deregistration (Anmeldung/Abmeldung) |
| **Firmenbuch** | Company registry search, extracts, watchlist monitoring |
| **E-Rechnung** | XRechnung/ZUGFeRD invoice generation (EN16931) |
| **SEPA** | pain.001/pain.008 generation, camt.053 parsing, IBAN/BIC validation |

### Platform Capabilities

- **Multi-tenant SaaS** — Isolated tenant data with row-level security
- **CLI + API + Portal** — Multiple interfaces for different use cases
- **MCP Integration** — AI-ready with Model Context Protocol server
- **Enterprise Security** — ES256 JWT, httpOnly cookies, CSP, rate limiting

## Quick Start

### Prerequisites

- Go 1.24+
- PostgreSQL 15+
- Redis 7+
- Node.js 20+ (frontend only)

### Installation

```bash
# Clone repository
git clone https://github.com/your-org/austrian-business-infrastructure.git
cd austrian-business-infrastructure

# Build CLI
go build -o fo ./cmd/fo

# Build server
go build -o server ./cmd/server
```

### CLI Usage

```bash
# Add FinanzOnline account
./fo account add --name "Muster GmbH" --tid 123456789 --benid USER01

# Check databox
./fo session login "Muster GmbH"
./fo databox list "Muster GmbH"

# Submit UVA
./fo uva submit --input uva.json --account "Muster GmbH"

# ELDA employee registration
./fo elda anmeldung --employee-file employee.json --account "My Company"

# Firmenbuch search
./fo fb search "Muster GmbH" --limit 10

# SEPA payment file
./fo sepa pain001 payments.csv --debtor-iban AT611904300234573201 -o payments.xml
```

### Server Mode (Development)

```bash
# Start dependencies
docker compose up -d postgres redis

# Configure environment
cp .env.example .env
# Edit .env with your settings

# Start server
go run ./cmd/server
```

## Self-Hosted Deployment

Production-ready deployment with automatic HTTPS via Caddy.

### Quick Deploy

```bash
# 1. Generate secure secrets
./scripts/generate-secrets.sh > .env

# 2. Configure your domain
echo "DOMAIN=your-domain.com" >> .env
echo "ACME_EMAIL=admin@your-domain.com" >> .env

# 3. Deploy
docker compose -f docker-compose.selfhost.yml up -d
```

### What's Included

- **Automatic TLS** — Caddy handles Let's Encrypt certificates
- **Auto Migrations** — Database schema updates on container restart
- **Security Hardening** — Read-only filesystem, resource limits, no exposed DB ports
- **Backup/Restore** — Scripts for data management

### Updates

```bash
docker compose -f docker-compose.selfhost.yml pull
docker compose -f docker-compose.selfhost.yml up -d
```

### Backup & Restore

```bash
# Backup (saves to ./backups/)
./scripts/backup.sh

# Backup to S3
./scripts/backup.sh s3://your-bucket/backups

# Restore
./scripts/restore.sh backups/abp_backup_20240101_120000.tar.gz
```

### Configuration Files

| File | Purpose |
|------|---------|
| `docker-compose.selfhost.yml` | Production with Caddy (recommended) |
| `docker-compose.prod.yml` | Production with Traefik (advanced) |
| `docker-compose.yml` | Local development only |
| `Caddyfile` | Reverse proxy with security headers |

## Documentation

### API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/auth/login` | Authenticate user |
| `POST` | `/api/v1/auth/refresh` | Refresh access token |
| `GET` | `/api/v1/auth/me` | Current user info |
| `WS` | `/api/v1/ws` | Real-time updates |

### CLI Commands

```
fo account     Manage FinanzOnline/ELDA accounts
fo session     Session management
fo databox     FinanzOnline databox operations
fo uva         VAT advance return (Umsatzsteuervoranmeldung)
fo zm          Summary declaration (Zusammenfassende Meldung)
fo elda        Social insurance operations
fo fb          Company registry (Firmenbuch)
fo erechnung   E-invoice generation
fo sepa        SEPA payment files
fo mcp         MCP server for AI integration
fo dashboard   Multi-service status overview
```

### MCP Server

For AI integration with Claude Desktop:

```json
{
  "mcpServers": {
    "austrian-business": {
      "command": "./fo",
      "args": ["mcp", "serve", "--stdio"]
    }
  }
}
```

## Security

Built with enterprise security requirements in mind.

### Authentication

| Feature | Implementation |
|---------|---------------|
| Token Signing | ES256 (ECDSA P-256) |
| Access Tokens | 15-minute expiry, memory-only storage |
| Refresh Tokens | httpOnly, Secure, SameSite=Strict cookies |
| 2FA | TOTP with encrypted secret storage |
| WebSocket | First-message authentication pattern |

### Infrastructure

- **Rate Limiting** — Per-IP and per-user limits, fail-closed for auth endpoints
- **Token Revocation** — Redis-backed blacklist with user/tenant-level revocation
- **Security Headers** — CSP, X-Frame-Options, HSTS, X-Content-Type-Options
- **Audit Logging** — Structured security event logging
- **Secrets Management** — Provider abstraction (env, file, Vault-ready)
- **CI/CD Security** — Automated gosec, govulncheck, and Trivy container scanning

### Data Protection

- Row-level security for tenant isolation
- AES-256-GCM encryption for credentials at rest
- No PII in JWT claims
- DSGVO/GDPR compliant data handling

## Project Structure

```
cmd/
├── fo/                 # CLI application
├── server/             # HTTP API server
└── worker/             # Background job processor

internal/
├── api/                # HTTP middleware, routing
├── auth/               # JWT, sessions, rate limiting
├── audit/              # Security event logging
├── crypto/             # Encryption, key management
├── elda/               # ELDA client
├── fonws/              # FinanzOnline WebService
├── fb/                 # Firmenbuch client
├── erechnung/          # E-invoice generation
├── sepa/               # SEPA file handling
├── mcp/                # MCP server
└── tenant/             # Multi-tenant support

frontend/               # SvelteKit admin dashboard
portal/                 # SvelteKit client portal
```

## Testing

```bash
# All tests
go test ./...

# Unit tests with coverage
go test ./tests/unit/... -cover

# Specific module
go test ./tests/unit/... -run TestJWT -v
```

## Compliance

| Standard | Status |
|----------|--------|
| DSGVO/GDPR | Data protection measures implemented |
| EN16931 | E-invoice format compliance |
| FinanzOnline API | Official WebService integration |
| ELDA | Austrian social insurance reporting |
| SEPA | ISO 20022 payment standards |

## Requirements

### Production

| Component | Version |
|-----------|---------|
| Go | 1.24+ |
| PostgreSQL | 15+ |
| Redis | 7+ |

### Credentials

- FinanzOnline WebService credentials (TID, BENID, PIN)
- ELDA certificate and credentials
- Firmenbuch API key (optional)

## Environment Variables

```bash
# Required
DATABASE_URL=postgres://user:pass@host/db
REDIS_URL=redis://localhost:6379
JWT_ECDSA_KEY_FILE=/path/to/private.pem

# Optional
APP_ENV=production
ALLOWED_ORIGINS=https://app.example.com
LOG_LEVEL=info
```

## Contributing

Contributions are welcome.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/improvement`)
3. Commit changes (`git commit -am 'Add feature'`)
4. Push to branch (`git push origin feature/improvement`)
5. Open a Pull Request

## License

This project is licensed under the **GNU Affero General Public License v3.0 (AGPL-3.0)**.

See [LICENSE](LICENSE) for details.

---

<div align="center">

**[Documentation](docs/)**

</div>
