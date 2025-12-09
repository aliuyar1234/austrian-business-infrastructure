-- Migration: 020_security_layer.sql
-- Security Layer - Foundational Migration
-- Feature: 020-security-layer
-- Date: 2025-12-08

-- =============================================================================
-- T001: Add 2FA columns to users table
-- =============================================================================

ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_secret BYTEA;
ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_enabled BOOLEAN DEFAULT false;
ALTER TABLE users ADD COLUMN IF NOT EXISTS recovery_codes BYTEA;
ALTER TABLE users ADD COLUMN IF NOT EXISTS recovery_codes_used INTEGER DEFAULT 0;

-- Index for 2FA enabled lookup
CREATE INDEX IF NOT EXISTS idx_users_totp_enabled
    ON users(tenant_id, totp_enabled) WHERE totp_enabled = true;

-- =============================================================================
-- T002: Update audit_logs table (already exists, add RLS support)
-- =============================================================================

-- Note: audit_logs table already exists from previous migrations
-- Add column if not exists for metadata (renamed from details)
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS metadata JSONB;

-- Migrate details to metadata if exists
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'audit_logs' AND column_name = 'details') THEN
        UPDATE audit_logs SET metadata = details WHERE metadata IS NULL AND details IS NOT NULL;
    END IF;
END $$;

-- =============================================================================
-- T003: Create tenant_deletion_requests table
-- =============================================================================

CREATE TABLE IF NOT EXISTS tenant_deletion_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
    requested_by UUID NOT NULL REFERENCES users(id),
    requested_at TIMESTAMP NOT NULL DEFAULT NOW(),
    grace_period_ends TIMESTAMP NOT NULL,
    cancelled_at TIMESTAMP,
    executed_at TIMESTAMP,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    reason VARCHAR(50),
    CONSTRAINT chk_deletion_status CHECK (status IN ('pending', 'cancelled', 'executing', 'executed'))
);

-- Index for pending deletion requests
CREATE INDEX IF NOT EXISTS idx_deletion_pending
    ON tenant_deletion_requests(status, grace_period_ends)
    WHERE status = 'pending';

-- =============================================================================
-- T004: Create key_rotation_log table
-- =============================================================================

CREATE TABLE IF NOT EXISTS key_rotation_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    old_version INTEGER NOT NULL,
    new_version INTEGER NOT NULL,
    credentials_updated INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP,
    status VARCHAR(20) NOT NULL DEFAULT 'in_progress',
    error_message TEXT,
    CONSTRAINT chk_rotation_status CHECK (status IN ('in_progress', 'completed', 'failed'))
);

-- =============================================================================
-- T005: Enable Row Level Security and create policies
-- =============================================================================

-- Enable RLS on tenant tables
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE accounts ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_deletion_requests ENABLE ROW LEVEL SECURITY;

-- Create tenant isolation policies
-- Note: Using app.tenant_id session variable for RLS context

-- Users table policy
DROP POLICY IF EXISTS tenant_isolation_users ON users;
CREATE POLICY tenant_isolation_users ON users
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- Audit log policy (tenant-scoped read, allow all inserts)
DROP POLICY IF EXISTS tenant_isolation_audit_logs ON audit_logs;
CREATE POLICY tenant_isolation_audit_logs ON audit_logs
    FOR SELECT
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS audit_logs_insert ON audit_logs;
CREATE POLICY audit_logs_insert ON audit_logs
    FOR INSERT
    WITH CHECK (true);

-- Accounts table policy
DROP POLICY IF EXISTS tenant_isolation_accounts ON accounts;
CREATE POLICY tenant_isolation_accounts ON accounts
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- Tenant deletion requests policy
DROP POLICY IF EXISTS tenant_isolation_deletion_requests ON tenant_deletion_requests;
CREATE POLICY tenant_isolation_deletion_requests ON tenant_deletion_requests
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- Audit log is append-only: revoke UPDATE and DELETE
REVOKE UPDATE, DELETE ON audit_logs FROM PUBLIC;

-- =============================================================================
-- T006: Create indexes for audit_log queries
-- =============================================================================

CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_time
    ON audit_logs(tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user
    ON audit_logs(tenant_id, user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_logs_action
    ON audit_logs(tenant_id, action, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_logs_resource
    ON audit_logs(tenant_id, resource_type, resource_id, created_at DESC);

-- =============================================================================
-- Additional: Update accounts table for key hierarchy support
-- =============================================================================

ALTER TABLE accounts ADD COLUMN IF NOT EXISTS pin_nonce BYTEA;
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS key_version INTEGER DEFAULT 1;

-- Index for credential lookup
CREATE INDEX IF NOT EXISTS idx_accounts_tenant
    ON accounts(tenant_id);

-- =============================================================================
-- Additional: Documents table RLS (if exists)
-- =============================================================================

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'documents') THEN
        ALTER TABLE documents ENABLE ROW LEVEL SECURITY;

        -- Drop existing policy if exists
        DROP POLICY IF EXISTS tenant_isolation_documents ON documents;

        -- Create tenant isolation policy
        CREATE POLICY tenant_isolation_documents ON documents
            FOR ALL
            USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
    END IF;
END $$;

-- =============================================================================
-- Additional: Other tenant tables RLS
-- =============================================================================

-- Enable RLS on other tenant tables if they exist
DO $$
DECLARE
    tbl TEXT;
    tenant_tables TEXT[] := ARRAY['tags', 'imports', 'sync_jobs', 'invitations', 'api_keys', 'sessions'];
BEGIN
    FOREACH tbl IN ARRAY tenant_tables
    LOOP
        IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = tbl) THEN
            EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', tbl);

            -- Drop existing policy if exists
            EXECUTE format('DROP POLICY IF EXISTS tenant_isolation_%I ON %I', tbl, tbl);

            -- Create tenant isolation policy
            EXECUTE format('CREATE POLICY tenant_isolation_%I ON %I FOR ALL USING (tenant_id = NULLIF(current_setting(''app.tenant_id'', true), '''')::uuid)', tbl, tbl);
        END IF;
    END LOOP;
END $$;

-- =============================================================================
-- Comment: Migration complete
-- =============================================================================
-- This migration:
-- 1. Adds 2FA columns (totp_secret, totp_enabled, recovery_codes) to users
-- 2. Creates audit_log table (append-only)
-- 3. Creates tenant_deletion_requests table (DSGVO Art. 17)
-- 4. Creates key_rotation_log table
-- 5. Enables RLS on all tenant tables
-- 6. Creates indexes for efficient audit log queries
-- 7. Adds key hierarchy columns to accounts table
