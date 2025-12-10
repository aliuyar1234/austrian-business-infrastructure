<div align="center">

# Austrian Business Infrastructure

### The Open-Source Backend for Austrian Government APIs

**Stop fighting SOAP. Start building.**

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-AGPL--3.0-ed1c24?style=for-the-badge)](LICENSE)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-14+-336791?style=for-the-badge&logo=postgresql&logoColor=white)](https://www.postgresql.org/)

[Getting Started](#quick-start) • [Architecture](#architecture) • [Documentation](#documentation) • [Security](#security)

</div>

---

## The Problem

Every Austrian business fights the same battle: **FinanzOnline**, **ELDA**, **Firmenbuch** — government APIs with SOAP interfaces, XML schemas from 2005, and zero open-source tooling.

**600,000+ Austrian companies.** All doing the same manual work. All paying for proprietary solutions.

Until now.

---

## The Solution

```
┌────────────────────────────────────────────────────────────────────────────┐
│                                                                            │
│   YOUR APP / ERP / BUCHHALTUNG                                            │
│                                                                            │
└──────────────────────────────────┬─────────────────────────────────────────┘
                                   │
                                   ▼
┌────────────────────────────────────────────────────────────────────────────┐
│                                                                            │
│   AUSTRIAN BUSINESS INFRASTRUCTURE                                        │
│                                                                            │
│   Go SDK  •  CLI Tools  •  REST API  •  MCP Server  •  SaaS Platform     │
│                                                                            │
└──────────────────────────────────┬─────────────────────────────────────────┘
                                   │
          ┌────────────────────────┼────────────────────────┐
          │                        │                        │
          ▼                        ▼                        ▼
   ┌─────────────┐          ┌─────────────┐          ┌─────────────┐
   │     BMF     │          │   ÖGK/SVS   │          │   Justiz    │
   │ FinanzOnline│          │    ELDA     │          │ Firmenbuch  │
   └─────────────┘          └─────────────┘          └─────────────┘
```

**One library. All Austrian government APIs. Production-ready.**

---

## What's Inside

<table>
<tr>
<td width="50%">

### Government Integrations

| Module | What it does |
|--------|--------------|
| **FinanzOnline** | Tax filings, databox, UVA, UID validation |
| **ELDA** | Employee registration, L16, social insurance |
| **Firmenbuch** | Company search, extracts, watchlists |
| **E-Rechnung** | XRechnung/ZUGFeRD (EN16931) |
| **SEPA** | pain.001, pain.008, camt.053 |

</td>
<td width="50%">

### Platform Features

| Feature | Description |
|---------|-------------|
| **Digital Signatures** | A-Trust + ID Austria (QES/eIDAS) |
| **AI Analysis** | OCR + LLM document classification |
| **Förderungsradar** | 74 funding programs + eligibility matching |
| **Multi-Tenancy** | Row-level security, tenant isolation |
| **Client Portal** | White-label portal for your clients |

</td>
</tr>
</table>

---

## Architecture

<div align="center">

![Austrian Business Infrastructure Architecture](docs/architecture.png)

*Enterprise-grade architecture with parallel service modules, background job processing, and multi-layer security*

</div>

---

## By The Numbers

<div align="center">

| | | | |
|:---:|:---:|:---:|:---:|
| **69** | **9** | **74** | **22** |
| Go Packages | Government APIs | Funding Programs | DB Migrations |

</div>

---

## Quick Start

### Option 1: CLI (Fastest)

```bash
# Build
go build -o fo ./cmd/fo

# Add a FinanzOnline account
./fo account add --name "Muster GmbH" --tid 123456789 --benid USER01

# Check 30 accounts in 12 seconds instead of 2.5 hours
./fo databox --all
```

### Option 2: Self-Hosted Platform

```bash
# Generate secrets
./scripts/generate-secrets.sh > .env

# Configure domain
echo "DOMAIN=your-domain.com" >> .env

# Deploy (includes auto-TLS via Caddy)
docker compose -f docker-compose.selfhost.yml up -d
```

### Option 3: Development

```bash
docker compose up -d postgres redis
cp .env.example .env
go run ./cmd/server
```

---

## CLI Commands

```bash
fo account     # Manage FinanzOnline/ELDA accounts
fo databox     # Poll databox across all accounts
fo uva         # Submit Umsatzsteuervoranmeldung
fo zm          # Submit Zusammenfassende Meldung
fo elda        # ELDA Anmeldung/Abmeldung
fo fb          # Firmenbuch search & extracts
fo erechnung   # Generate XRechnung/ZUGFeRD
fo sepa        # Generate SEPA payment files
fo sign        # Digital signatures (A-Trust/ID Austria)
fo foerderung  # Search 74 Austrian funding programs
fo analyze     # AI document analysis
fo mcp         # MCP server for AI integration (Claude, etc.)
```

---

## Security

This isn't a hobby project. It's built for production.

| Layer | Implementation |
|-------|----------------|
| **Authentication** | ES256 JWT (ECDSA P-256), TOTP 2FA |
| **Authorization** | Row-Level Security (RLS) at database level |
| **IDOR Protection** | AccountVerifier pattern on all write operations |
| **Secrets** | AES-256-GCM encryption at rest |
| **CI/CD** | All GitHub Actions pinned to SHA hashes |
| **Containers** | Images pinned to specific versions |
| **Scanning** | gosec, govulncheck, Trivy on every push |
| **Compliance** | DSGVO/GDPR, OWASP Top 10, eIDAS |

---

## MCP Server (AI Integration)

Connect your AI assistant directly to Austrian government APIs:

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

*"Check my FinanzOnline databox and summarize any new documents"* — now possible.

---

## API Endpoints

Full REST API for integration with your existing systems:

```
POST   /api/v1/auth/login          # JWT authentication
GET    /api/v1/accounts            # List all accounts
POST   /api/v1/accounts/:id/sync   # Trigger databox sync
GET    /api/v1/documents           # List documents
POST   /api/v1/uva/submit          # Submit UVA
POST   /api/v1/sepa/pain001        # Generate SEPA file
GET    /api/v1/foerderung/match    # Match funding programs
WS     /api/v1/ws                  # Real-time updates
```

---

## Project Structure

```
cmd/
├── fo/                 # CLI application
├── server/             # HTTP API server
└── worker/             # Background job processor

internal/               # 69 packages including:
├── fonws/              # FinanzOnline WebService client
├── elda/               # ELDA client (11 files)
├── firmenbuch/         # Firmenbuch client
├── erechnung/          # E-invoice generation
├── sepa/               # SEPA file handling
├── signature/          # A-Trust/ID Austria
├── foerderung/         # 74 funding programs
├── matcher/            # LLM eligibility matching
├── analysis/           # AI document analysis
├── security/           # RLS, rate limiting, IDOR protection
└── ...

frontend/               # SvelteKit admin dashboard
portal/                 # SvelteKit client portal
migrations/             # 22 PostgreSQL migrations
```

---

## Requirements

| Component | Version |
|-----------|---------|
| Go | 1.24+ |
| PostgreSQL | 14+ |
| Redis | 7+ |
| Node.js | 20+ (frontend only) |

### Credentials You'll Need

- **FinanzOnline**: WebService TID, BENID, PIN ([Apply here](https://finanzonline.bmf.gv.at))
- **ELDA**: Certificate + credentials from ÖGK
- **Firmenbuch**: API key (optional, for automated queries)
- **A-Trust**: Signing credentials (for digital signatures)

---

## Why This Exists

> "In 5 years, every Austrian business automation runs on this library — or a fork of it."

The Austrian government has APIs. Good ones, actually. But:
- SOAP/XML interfaces nobody wants to touch
- Zero open-source implementations
- Every company builds the same wrappers
- Proprietary solutions cost €€€€

**We built the missing infrastructure layer.**

Open source wins infrastructure. Always.

---

## Contributing

```bash
# Fork & clone
git clone https://github.com/YOUR_USERNAME/austrian-business-infrastructure.git

# Create branch
git checkout -b feature/your-feature

# Make changes, then
go test ./...
go build ./...

# Submit PR
```

---

## License

**AGPL-3.0** — Use it freely. If you modify it for a hosted service, share your changes.

See [LICENSE](LICENSE) for details.

---

<div align="center">

**Built for Austrian businesses. Open source forever.**

[Documentation](docs/) • [Report Issue](https://github.com/aliuyar1234/austrian-business-infrastructure/issues)

</div>
