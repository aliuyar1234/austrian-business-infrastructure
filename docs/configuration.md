# Configuration

All configuration is done via environment variables. Copy `.env.example` to `.env` and adjust as needed.

## Core Settings

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `APP_ENV` | Environment (`development`, `production`) | `development` | No |
| `PORT` | API server port | `8080` | No |
| `FRONTEND_URL` | Frontend URL for CORS | `http://localhost:3000` | Yes |

## Database

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DATABASE_URL` | PostgreSQL connection string | - | Yes |
| `DB_MAX_CONNECTIONS` | Connection pool size | `25` | No |
| `DB_MAX_IDLE_TIME` | Idle connection timeout | `15m` | No |

Example:
```bash
DATABASE_URL=postgres://user:password@localhost:5432/austrian_business?sslmode=disable
```

## Redis

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `REDIS_URL` | Redis connection string | `redis://localhost:6379` | Yes |
| `REDIS_PASSWORD` | Redis password | - | No |

## Authentication

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `JWT_SECRET` | JWT signing secret (min 32 chars) | - | Yes |
| `JWT_ACCESS_EXPIRY` | Access token lifetime | `15m` | No |
| `JWT_REFRESH_EXPIRY` | Refresh token lifetime | `7d` | No |

## Encryption

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `MASTER_KEY` | AES-256 master key (32 bytes, base64) | - | Yes |
| `KEY_DERIVATION_SALT` | HKDF salt for key derivation | - | Yes |

Generate a master key:
```bash
openssl rand -base64 32
```

## Storage

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `STORAGE_TYPE` | Storage backend (`local`, `s3`) | `local` | No |
| `STORAGE_PATH` | Local storage path | `./data` | No |
| `S3_BUCKET` | S3 bucket name | - | If S3 |
| `S3_REGION` | S3 region | - | If S3 |
| `S3_ACCESS_KEY` | S3 access key | - | If S3 |
| `S3_SECRET_KEY` | S3 secret key | - | If S3 |

## FinanzOnline

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `FO_WEBSERVICE_URL` | FinanzOnline API URL | Production URL | No |
| `FO_SESSION_TIMEOUT` | Session timeout | `30m` | No |

## ELDA

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `ELDA_ENDPOINT` | ELDA service endpoint | Production URL | No |
| `ELDA_CERTIFICATE_PATH` | Path to client certificate | - | For prod |

## AI Integration (Optional)

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `CLAUDE_API_KEY` | Anthropic API key | - | No |
| `CLAUDE_MODEL` | Model to use | `claude-sonnet-4-20250514` | No |
| `CLAUDE_MAX_TOKENS` | Max response tokens | `4096` | No |

## Email (Optional)

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `SMTP_HOST` | SMTP server host | - | No |
| `SMTP_PORT` | SMTP server port | `587` | No |
| `SMTP_USER` | SMTP username | - | No |
| `SMTP_PASSWORD` | SMTP password | - | No |
| `SMTP_FROM` | From address | - | No |

## Example .env File

```bash
# Core
APP_ENV=production
PORT=8080
FRONTEND_URL=https://app.example.com

# Database
DATABASE_URL=postgres://user:pass@localhost:5432/austrian_business?sslmode=require

# Redis
REDIS_URL=redis://localhost:6379

# Security
JWT_SECRET=your-super-secret-jwt-key-minimum-32-chars
MASTER_KEY=base64-encoded-32-byte-key
KEY_DERIVATION_SALT=unique-salt-for-your-deployment

# Storage
STORAGE_TYPE=s3
S3_BUCKET=my-documents
S3_REGION=eu-central-1
S3_ACCESS_KEY=AKIA...
S3_SECRET_KEY=...
```
