<div align="center">

# Austrian Business Infrastructure

**The first open-source toolkit for Austrian government API integrations**

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-AGPL--3.0-ed1c24?style=flat-square)](LICENSE)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-14+-336791?style=flat-square&logo=postgresql&logoColor=white)](https://www.postgresql.org/)

</div>

---

## Problems this solves

**Tax Advisors / Accounting**
> "I manage 30 FinanzOnline accounts for our subsidiaries. Every day I log into each one, 2FA for every single account, just to check for new documents. Takes hours."

→ `./fo databox --all` — 30 accounts in 12 seconds.

**HR Departments**
> "Employee registration, deregistration, annual tax forms — all manual through ELDA. With 50+ employees per month, it becomes a full-time job."

→ `./fo elda anmeldung --file employees.json` — Batch processing.

**Compliance Teams**
> "We need company registry extracts regularly for due diligence. Every time it's manual search and download."

→ `./fo fb search "GmbH" --extract` — Automated, monitored, notified.

**Finance / Treasury**
> "Building SEPA payment files by hand is error-prone. Reconciling bank statements is even worse."

→ `./fo sepa pain001 payments.csv` — Generated, validated, done.

**Management**
> "What funding programs exist for us? AWS, FFG, EU? No idea where to start."

→ 74 Austrian funding programs searchable, with eligibility matching.

---

**This is the first open-source solution for Austrian government APIs.**

---

## How it works

![Solution](docs/solution.png)

Your application talks to this library. This library talks to the government APIs. You never touch SOAP.

**Use it as:**
- **Go SDK** — Import into your Go code
- **CLI** — Automate from terminal or scripts
- **REST API** — Integrate from any language
- **MCP Server** — Connect AI assistants directly
- **SaaS Platform** — Full multi-tenant web application

---

## What's included

| Module | Capabilities |
|--------|--------------|
| **FinanzOnline** | Session handling, databox polling, UVA/ZM submission, UID validation |
| **ELDA** | Employee registration (Anmeldung/Abmeldung), L16, mBGM |
| **Firmenbuch** | Company search, extracts, watchlist monitoring |
| **E-Rechnung** | XRechnung/ZUGFeRD generation, EN16931 compliant |
| **SEPA** | Payment files (pain.001, pain.008), bank statements (camt.053) |
| **Digital Signatures** | Qualified electronic signatures via A-Trust, ID Austria |
| **Document Analysis** | OCR + LLM-based classification and data extraction |
| **Förderungsradar** | 74 Austrian funding programs with eligibility matching |

---

## Architecture

![Architecture](docs/architecture.png)

<details>
<summary><strong>Project structure</strong></summary>

```
cmd/
├── fo/          CLI tool
├── server/      REST API server
└── worker/      Background jobs

internal/        69 packages
├── fonws/       FinanzOnline WebService
├── elda/        ELDA client
├── firmenbuch/  Firmenbuch client
├── erechnung/   Invoice generation
├── sepa/        SEPA handling
├── signature/   Digital signatures
├── foerderung/  74 funding programs
├── matcher/     LLM matching
├── analysis/    Document analysis
├── security/    RLS, rate limiting
├── auth/        JWT, 2FA
└── ...

frontend/        SvelteKit dashboard
portal/          Client portal
migrations/      22 database migrations
```

</details>

---

## Quick Start

**Build the CLI**
```bash
go build -o fo ./cmd/fo
```

**Add an account and check databox**
```bash
./fo account add --name "Muster GmbH" --tid 123456789 --benid USER01
./fo databox list "Muster GmbH"
```

**Submit a UVA**
```bash
./fo uva submit --input uva.json --account "Muster GmbH"
```

**Deploy self-hosted**
```bash
./scripts/generate-secrets.sh > .env
echo "DOMAIN=your-domain.com" >> .env
docker compose -f docker-compose.selfhost.yml up -d
```

---

## CLI Commands

```
fo account     Manage FinanzOnline/ELDA/Firmenbuch accounts
fo databox     Poll FinanzOnline databox
fo uva         Submit Umsatzsteuervoranmeldung
fo zm          Submit Zusammenfassende Meldung
fo elda        Employee registration
fo fb          Firmenbuch search and extracts
fo erechnung   Generate XRechnung/ZUGFeRD
fo sepa        Generate SEPA payment files
fo sign        Digital signatures
fo foerderung  Search funding programs
fo analyze     Document analysis
fo mcp         MCP server for AI assistants
```

---

## REST API

Available at `/api/v1/`:

```
POST   /auth/login           Authentication
GET    /accounts             List accounts
POST   /accounts/:id/sync    Trigger sync
GET    /documents            List documents
POST   /uva/submit           Submit UVA
POST   /sepa/pain001         Generate SEPA
GET    /foerderung/match     Match funding
WS     /ws                   Real-time updates
```

---

## MCP Server

Connect AI assistants to Austrian government APIs:

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

---

## Security

This handles tax filings and government credentials. Security is non-negotiable.

| Layer | Implementation |
|-------|----------------|
| Authentication | ES256 JWT, TOTP 2FA |
| Authorization | Row-Level Security (PostgreSQL) |
| IDOR Protection | AccountVerifier pattern |
| Encryption | AES-256-GCM at rest |
| CI/CD | GitHub Actions pinned to SHA |
| Scanning | gosec, govulncheck, Trivy |
| Compliance | DSGVO, OWASP Top 10, eIDAS |

---

## Numbers

| 69 | 9 | 74 | 22 |
|:--:|:--:|:--:|:--:|
| Go packages | Government APIs | Funding programs | DB migrations |

---

## Requirements

| Component | Version |
|-----------|---------|
| Go | 1.24+ |
| PostgreSQL | 14+ |
| Redis | 7+ |
| Node.js | 20+ (frontend) |

**Credentials:**
- **FinanzOnline**: TID, BENID, PIN — [finanzonline.bmf.gv.at](https://finanzonline.bmf.gv.at)
- **ELDA**: Certificate from ÖGK
- **Firmenbuch**: API key (optional)
- **A-Trust**: Signing credentials (optional)

---

## Contributing

```bash
git checkout -b feature/your-feature
go test ./...
go build ./...
```

---

## License

AGPL-3.0 — [LICENSE](LICENSE)

---

<div align="center">

[Documentation](docs/) · [Issues](https://github.com/aliuyar1234/austrian-business-infrastructure/issues)

</div>
