#!/bin/bash
# =============================================================================
# Generate secure secrets for Austrian Business Platform
# =============================================================================
# Usage: ./scripts/generate-secrets.sh > .env
#        or append to existing: ./scripts/generate-secrets.sh >> .env
# =============================================================================

set -e

echo "# Generated secrets - $(date)"
echo "# DO NOT COMMIT THIS FILE TO VERSION CONTROL"
echo ""

# Generate JWT secret (64 hex chars = 32 bytes)
JWT_SECRET=$(openssl rand -hex 32)
echo "JWT_SECRET=$JWT_SECRET"

# Generate encryption key (32 chars for AES-256)
ENCRYPTION_KEY=$(openssl rand -hex 16)
echo "ENCRYPTION_KEY=$ENCRYPTION_KEY"

# Generate database password
POSTGRES_PASSWORD=$(openssl rand -base64 24 | tr -dc 'a-zA-Z0-9' | head -c 32)
echo "POSTGRES_PASSWORD=$POSTGRES_PASSWORD"

# Generate Redis password
REDIS_PASSWORD=$(openssl rand -base64 24 | tr -dc 'a-zA-Z0-9' | head -c 32)
echo "REDIS_PASSWORD=$REDIS_PASSWORD"

echo ""
echo "# Database configuration"
echo "POSTGRES_USER=abp"
echo "POSTGRES_DB=abp"

echo ""
echo "# Required: Set your domain"
echo "# DOMAIN=your-domain.com"
echo "# ACME_EMAIL=admin@your-domain.com"

echo ""
echo "# ============================================="
echo "# Secrets generated successfully!"
echo "# Next steps:"
echo "#   1. Set DOMAIN and ACME_EMAIL above"
echo "#   2. Save this output to .env file"
echo "#   3. Run: docker compose -f docker-compose.selfhost.yml up -d"
echo "# ============================================="
