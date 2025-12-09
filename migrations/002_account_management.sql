-- Migration: 002_account_management
-- Description: Account management for FinanzOnline, ELDA, Firmenbuch credentials

-- Accounts table: stores encrypted credentials for external services
CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('finanzonline', 'elda', 'firmenbuch')),
    credentials BYTEA NOT NULL,
    credentials_iv BYTEA NOT NULL,
    status VARCHAR(50) DEFAULT 'unverified' CHECK (status IN ('unverified', 'verified', 'error', 'suspended')),
    last_verified_at TIMESTAMPTZ,
    last_sync_at TIMESTAMPTZ,
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);

-- Indexes for accounts
CREATE INDEX IF NOT EXISTS idx_accounts_tenant ON accounts(tenant_id);
CREATE INDEX IF NOT EXISTS idx_accounts_type ON accounts(type);
CREATE INDEX IF NOT EXISTS idx_accounts_status ON accounts(status);
CREATE INDEX IF NOT EXISTS idx_accounts_deleted ON accounts(deleted_at) WHERE deleted_at IS NULL;

-- Connection tests: history of credential verification attempts
CREATE TABLE IF NOT EXISTS connection_tests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    success BOOLEAN NOT NULL,
    duration_ms INTEGER,
    error_code VARCHAR(50),
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for connection tests
CREATE INDEX IF NOT EXISTS idx_connection_tests_account ON connection_tests(account_id);
CREATE INDEX IF NOT EXISTS idx_connection_tests_time ON connection_tests(created_at DESC);

-- Tags: user-defined labels for organizing accounts
CREATE TABLE IF NOT EXISTS tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    color VARCHAR(7),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, name)
);

-- Account-tag junction table
CREATE TABLE IF NOT EXISTS account_tags (
    account_id UUID REFERENCES accounts(id) ON DELETE CASCADE,
    tag_id UUID REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (account_id, tag_id)
);

-- Indexes for tags
CREATE INDEX IF NOT EXISTS idx_tags_tenant ON tags(tenant_id);
CREATE INDEX IF NOT EXISTS idx_account_tags_account ON account_tags(account_id);
CREATE INDEX IF NOT EXISTS idx_account_tags_tag ON account_tags(tag_id);

-- Import jobs: bulk account import tracking
CREATE TABLE IF NOT EXISTS import_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    total_rows INTEGER,
    processed_rows INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    error_count INTEGER DEFAULT 0,
    errors JSONB DEFAULT '[]',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMPTZ
);

-- Indexes for import jobs
CREATE INDEX IF NOT EXISTS idx_import_jobs_tenant ON import_jobs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_import_jobs_user ON import_jobs(user_id);
CREATE INDEX IF NOT EXISTS idx_import_jobs_status ON import_jobs(status);

-- Trigger for accounts updated_at
CREATE OR REPLACE FUNCTION update_accounts_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS accounts_updated_at ON accounts;
CREATE TRIGGER accounts_updated_at
    BEFORE UPDATE ON accounts
    FOR EACH ROW
    EXECUTE FUNCTION update_accounts_updated_at();
