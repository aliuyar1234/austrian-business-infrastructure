-- Migration: 003_databox_documents
-- Description: Databox documents, sync jobs, and notification preferences
-- Spec: 007-databox-documents-api

-- Documents table - stores fetched databox documents
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    external_id VARCHAR(255),
    type VARCHAR(100) NOT NULL,
    title VARCHAR(500),
    sender VARCHAR(255),
    received_at TIMESTAMPTZ,
    content_hash VARCHAR(64),
    storage_path VARCHAR(500),
    file_size INTEGER,
    mime_type VARCHAR(100),
    status VARCHAR(50) DEFAULT 'new' CHECK (status IN ('new', 'read', 'archived')),
    archived_at TIMESTAMPTZ,
    retention_until TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(account_id, external_id)
);

-- Indexes for document queries
CREATE INDEX idx_documents_account ON documents(account_id);
CREATE INDEX idx_documents_status ON documents(status);
CREATE INDEX idx_documents_received ON documents(received_at DESC);
CREATE INDEX idx_documents_type ON documents(type);
CREATE INDEX idx_documents_content_hash ON documents(content_hash);
CREATE INDEX idx_documents_archived ON documents(archived_at) WHERE archived_at IS NOT NULL;

-- Full-text search index for title and metadata
CREATE INDEX idx_documents_search ON documents USING GIN (
    to_tsvector('german', COALESCE(title, '') || ' ' || COALESCE(sender, ''))
);

-- Sync jobs table - tracks databox synchronization
CREATE TABLE sync_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    account_id UUID REFERENCES accounts(id) ON DELETE SET NULL,
    status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    job_type VARCHAR(50) DEFAULT 'single' CHECK (job_type IN ('single', 'all')),
    documents_found INTEGER DEFAULT 0,
    documents_new INTEGER DEFAULT 0,
    documents_skipped INTEGER DEFAULT 0,
    error_message TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_sync_jobs_account ON sync_jobs(account_id);
CREATE INDEX idx_sync_jobs_tenant ON sync_jobs(tenant_id);
CREATE INDEX idx_sync_jobs_status ON sync_jobs(status);
CREATE INDEX idx_sync_jobs_created ON sync_jobs(created_at DESC);

-- Notification preferences per user
CREATE TABLE notification_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    document_types TEXT[] DEFAULT '{}',
    email_enabled BOOLEAN DEFAULT TRUE,
    email_digest BOOLEAN DEFAULT FALSE,
    digest_time TIME DEFAULT '08:00:00',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id)
);

CREATE INDEX idx_notification_prefs_user ON notification_preferences(user_id);

-- Notification queue for background processing
CREATE TABLE notification_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    notification_type VARCHAR(50) DEFAULT 'email',
    status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'failed')),
    retry_count INTEGER DEFAULT 0,
    error_message TEXT,
    scheduled_at TIMESTAMPTZ DEFAULT NOW(),
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_notification_queue_status ON notification_queue(status);
CREATE INDEX idx_notification_queue_scheduled ON notification_queue(scheduled_at) WHERE status = 'pending';

-- Update updated_at triggers
CREATE TRIGGER update_documents_updated_at
    BEFORE UPDATE ON documents
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_notification_prefs_updated_at
    BEFORE UPDATE ON notification_preferences
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- Comments
COMMENT ON TABLE documents IS 'Databox documents fetched from FinanzOnline and other services';
COMMENT ON COLUMN documents.external_id IS 'External ID from source system (e.g., FO databox ID)';
COMMENT ON COLUMN documents.type IS 'Document type: bescheid, ersuchen, mitteilung, mahnung, sonstige';
COMMENT ON COLUMN documents.content_hash IS 'SHA-256 hash for deduplication';
COMMENT ON COLUMN documents.storage_path IS 'Path in storage system (local or S3)';

COMMENT ON TABLE sync_jobs IS 'Tracks databox synchronization jobs';
COMMENT ON COLUMN sync_jobs.job_type IS 'single = one account, all = all accounts for tenant';

COMMENT ON TABLE notification_preferences IS 'User preferences for document notifications';
COMMENT ON COLUMN notification_preferences.document_types IS 'Array of document types to notify about';
COMMENT ON COLUMN notification_preferences.digest_time IS 'Time to send daily digest (if enabled)';
