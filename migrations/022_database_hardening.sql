-- Migration: 022_database_hardening
-- Description: Database hardening - pgcrypto extension, indexes, and timeouts
-- Fixes: Codex audit findings for database layer

-- =============================================================================
-- Step 1: Enable pgcrypto extension for gen_random_uuid() compatibility
-- =============================================================================
-- Note: gen_random_uuid() is built-in for PostgreSQL 13+, but pgcrypto
-- is required for PostgreSQL 12 and earlier. Adding it ensures compatibility.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- =============================================================================
-- Step 2: Add retention index for document cleanup queries
-- =============================================================================
-- This index supports the GetExpired query:
--   WHERE d.retention_until < NOW() AND d.tenant_id = $1
--   ORDER BY d.retention_until

CREATE INDEX IF NOT EXISTS idx_documents_retention
    ON documents(tenant_id, retention_until)
    WHERE retention_until IS NOT NULL;

-- =============================================================================
-- Step 3: Add index for document status queries
-- =============================================================================
-- Supports filtering by status (common in list queries)

CREATE INDEX IF NOT EXISTS idx_documents_tenant_status
    ON documents(tenant_id, status, received_at DESC);

-- =============================================================================
-- Step 4: Set statement timeout defaults for safety
-- =============================================================================
-- These settings prevent runaway queries from consuming resources
-- Can be overridden per-connection or per-transaction if needed

-- Set default statement timeout to 30 seconds for the app role
-- Note: This only affects the current database, not system-wide
DO $$
BEGIN
    -- Only set if not already configured (to avoid overwriting custom settings)
    IF current_setting('statement_timeout', true) = '0' OR
       current_setting('statement_timeout', true) IS NULL THEN
        ALTER DATABASE CURRENT SET statement_timeout = '30s';
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        -- Ignore errors (might not have permission)
        RAISE NOTICE 'Could not set statement_timeout: %', SQLERRM;
END $$;

-- Set lock_timeout to prevent indefinite lock waits
DO $$
BEGIN
    IF current_setting('lock_timeout', true) = '0' OR
       current_setting('lock_timeout', true) IS NULL THEN
        ALTER DATABASE CURRENT SET lock_timeout = '10s';
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Could not set lock_timeout: %', SQLERRM;
END $$;

-- Set idle_in_transaction_session_timeout to clean up abandoned transactions
DO $$
BEGIN
    IF current_setting('idle_in_transaction_session_timeout', true) = '0' OR
       current_setting('idle_in_transaction_session_timeout', true) IS NULL THEN
        ALTER DATABASE CURRENT SET idle_in_transaction_session_timeout = '60s';
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'Could not set idle_in_transaction_session_timeout: %', SQLERRM;
END $$;

-- =============================================================================
-- Step 5: Add missing unique constraints for business keys
-- =============================================================================

-- Ensure unique external_id per account (prevents duplicate document imports)
-- Note: This constraint may already exist, using IF NOT EXISTS pattern
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'uq_documents_account_external_id'
    ) THEN
        ALTER TABLE documents
            ADD CONSTRAINT uq_documents_account_external_id
            UNIQUE (account_id, external_id);
    END IF;
EXCEPTION
    WHEN duplicate_object THEN
        -- Constraint already exists
        NULL;
END $$;

-- =============================================================================
-- Comments
-- =============================================================================

COMMENT ON INDEX idx_documents_retention IS 'Supports retention/cleanup queries for expired documents';
COMMENT ON INDEX idx_documents_tenant_status IS 'Supports document listing filtered by status';
