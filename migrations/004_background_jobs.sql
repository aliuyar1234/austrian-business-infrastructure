-- Migration: 004_background_jobs
-- Description: Background jobs, schedules, webhooks, and job history
-- Spec: 008-background-jobs-automation

-- Job status enum-like check
-- Status: pending, running, completed, failed, dead

-- Priority levels
-- high = 10, normal = 5, low = 1

-- Sync interval enum-like check
-- Options: hourly, 4hourly, daily, weekly, disabled

-- =============================================================================
-- JOBS TABLE - Main job queue
-- =============================================================================

CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    type VARCHAR(100) NOT NULL,
    payload JSONB DEFAULT '{}',
    priority INTEGER DEFAULT 5 CHECK (priority BETWEEN 1 AND 10),
    status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed', 'dead')),
    max_retries INTEGER DEFAULT 3,
    retry_count INTEGER DEFAULT 0,
    last_error TEXT,
    run_at TIMESTAMPTZ DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    timeout_seconds INTEGER DEFAULT 1800, -- 30 minutes default
    worker_id VARCHAR(255),
    idempotency_key VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for job queue operations
CREATE INDEX idx_jobs_pending ON jobs(run_at, priority DESC) WHERE status = 'pending';
CREATE INDEX idx_jobs_tenant ON jobs(tenant_id);
CREATE INDEX idx_jobs_type ON jobs(type);
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_worker ON jobs(worker_id) WHERE status = 'running';
CREATE INDEX idx_jobs_idempotency ON jobs(idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX idx_jobs_created ON jobs(created_at DESC);

-- =============================================================================
-- SCHEDULES TABLE - Cron schedules for recurring jobs
-- =============================================================================

CREATE TABLE schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    job_type VARCHAR(100) NOT NULL,
    job_payload JSONB DEFAULT '{}',
    cron_expression VARCHAR(100), -- Standard cron format
    interval VARCHAR(50), -- Alternative: hourly, 4hourly, daily, weekly
    enabled BOOLEAN DEFAULT TRUE,
    timezone VARCHAR(100) DEFAULT 'UTC',
    last_run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ,
    run_count INTEGER DEFAULT 0,
    fail_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

CREATE INDEX idx_schedules_tenant ON schedules(tenant_id);
CREATE INDEX idx_schedules_next_run ON schedules(next_run_at) WHERE enabled = TRUE;
CREATE INDEX idx_schedules_job_type ON schedules(job_type);

-- =============================================================================
-- JOB_HISTORY TABLE - Historical job executions
-- =============================================================================

CREATE TABLE job_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    job_id UUID, -- May be NULL if job was deleted
    schedule_id UUID REFERENCES schedules(id) ON DELETE SET NULL,
    type VARCHAR(100) NOT NULL,
    payload JSONB DEFAULT '{}',
    status VARCHAR(50) NOT NULL CHECK (status IN ('completed', 'failed')),
    result JSONB DEFAULT '{}',
    error_message TEXT,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ NOT NULL,
    duration_ms INTEGER GENERATED ALWAYS AS (
        EXTRACT(EPOCH FROM (completed_at - started_at)) * 1000
    ) STORED,
    worker_id VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_job_history_tenant ON job_history(tenant_id);
CREATE INDEX idx_job_history_type ON job_history(type);
CREATE INDEX idx_job_history_status ON job_history(status);
CREATE INDEX idx_job_history_started ON job_history(started_at DESC);
CREATE INDEX idx_job_history_schedule ON job_history(schedule_id);

-- =============================================================================
-- DEAD_LETTERS TABLE - Jobs that failed permanently
-- =============================================================================

CREATE TABLE dead_letters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    original_job_id UUID,
    type VARCHAR(100) NOT NULL,
    payload JSONB DEFAULT '{}',
    errors JSONB DEFAULT '[]', -- Array of error messages from each attempt
    max_retries INTEGER,
    total_attempts INTEGER,
    first_attempted_at TIMESTAMPTZ,
    last_attempted_at TIMESTAMPTZ,
    acknowledged BOOLEAN DEFAULT FALSE,
    acknowledged_by UUID REFERENCES users(id),
    acknowledged_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_dead_letters_tenant ON dead_letters(tenant_id);
CREATE INDEX idx_dead_letters_type ON dead_letters(type);
CREATE INDEX idx_dead_letters_unacked ON dead_letters(created_at) WHERE acknowledged = FALSE;

-- =============================================================================
-- WEBHOOKS TABLE - Webhook configurations
-- =============================================================================

CREATE TABLE webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    url VARCHAR(2048) NOT NULL,
    secret VARCHAR(255) NOT NULL, -- For HMAC signature
    events TEXT[] DEFAULT '{}', -- Array of event types: new_document, deadline_warning, fb_change, sync_complete
    enabled BOOLEAN DEFAULT TRUE,
    timeout_seconds INTEGER DEFAULT 30,
    max_retries INTEGER DEFAULT 3,
    headers JSONB DEFAULT '{}', -- Custom headers to send
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

CREATE INDEX idx_webhooks_tenant ON webhooks(tenant_id);
CREATE INDEX idx_webhooks_enabled ON webhooks(tenant_id) WHERE enabled = TRUE;

-- =============================================================================
-- WEBHOOK_DELIVERIES TABLE - Webhook call history
-- =============================================================================

CREATE TABLE webhook_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id UUID NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'success', 'failed')),
    response_status INTEGER,
    response_body TEXT,
    response_headers JSONB,
    attempt_count INTEGER DEFAULT 0,
    last_error TEXT,
    next_retry_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_webhook_deliveries_webhook ON webhook_deliveries(webhook_id);
CREATE INDEX idx_webhook_deliveries_tenant ON webhook_deliveries(tenant_id);
CREATE INDEX idx_webhook_deliveries_pending ON webhook_deliveries(next_retry_at) WHERE status = 'pending';
CREATE INDEX idx_webhook_deliveries_created ON webhook_deliveries(created_at DESC);

-- =============================================================================
-- WATCHLIST TABLE - Firmenbuch watchlist entries
-- =============================================================================

CREATE TABLE watchlist (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    account_id UUID REFERENCES accounts(id) ON DELETE SET NULL,
    company_number VARCHAR(50) NOT NULL, -- FN number e.g., "12345d"
    company_name VARCHAR(500),
    last_snapshot JSONB, -- Last fetched FB data for comparison
    last_checked_at TIMESTAMPTZ,
    last_changed_at TIMESTAMPTZ,
    check_enabled BOOLEAN DEFAULT TRUE,
    notify_on_change BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, company_number)
);

CREATE INDEX idx_watchlist_tenant ON watchlist(tenant_id);
CREATE INDEX idx_watchlist_enabled ON watchlist(tenant_id) WHERE check_enabled = TRUE;
CREATE INDEX idx_watchlist_company ON watchlist(company_number);

-- =============================================================================
-- ALTER ACCOUNTS TABLE - Add sync configuration
-- =============================================================================

ALTER TABLE accounts ADD COLUMN IF NOT EXISTS sync_interval VARCHAR(50) DEFAULT '4hourly'
    CHECK (sync_interval IN ('hourly', '4hourly', 'daily', 'weekly', 'disabled'));
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS auto_sync_enabled BOOLEAN DEFAULT TRUE;
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS last_sync_at TIMESTAMPTZ;
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS next_sync_at TIMESTAMPTZ;

-- =============================================================================
-- ALTER DOCUMENTS TABLE - Add deadline field
-- =============================================================================

ALTER TABLE documents ADD COLUMN IF NOT EXISTS deadline TIMESTAMPTZ;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS reminder_7d_sent_at TIMESTAMPTZ;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS reminder_3d_sent_at TIMESTAMPTZ;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS reminder_1d_sent_at TIMESTAMPTZ;

CREATE INDEX idx_documents_deadline ON documents(deadline) WHERE deadline IS NOT NULL;

-- =============================================================================
-- ALTER NOTIFICATION_PREFERENCES TABLE - Add tenant_id and more fields
-- =============================================================================

ALTER TABLE notification_preferences ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;
ALTER TABLE notification_preferences ADD COLUMN IF NOT EXISTS email_mode VARCHAR(50) DEFAULT 'immediate'
    CHECK (email_mode IN ('immediate', 'digest', 'off'));
ALTER TABLE notification_preferences ADD COLUMN IF NOT EXISTS account_ids UUID[] DEFAULT '{}';

-- Drop old unique constraint and create new one
DROP INDEX IF EXISTS notification_preferences_user_id_key;
CREATE UNIQUE INDEX idx_notification_prefs_user_tenant ON notification_preferences(user_id, tenant_id);

-- =============================================================================
-- TENANT SETTINGS - Add job configuration
-- =============================================================================

ALTER TABLE tenants ADD COLUMN IF NOT EXISTS job_settings JSONB DEFAULT '{
    "auto_sync_enabled": true,
    "default_sync_interval": "4hourly",
    "reminder_days": [7, 3, 1],
    "webhooks_enabled": true,
    "audit_retention_days": 90
}';

-- =============================================================================
-- UPDATE TRIGGERS
-- =============================================================================

CREATE TRIGGER update_jobs_updated_at
    BEFORE UPDATE ON jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_schedules_updated_at
    BEFORE UPDATE ON schedules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_webhooks_updated_at
    BEFORE UPDATE ON webhooks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_watchlist_updated_at
    BEFORE UPDATE ON watchlist
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- =============================================================================
-- COMMENTS
-- =============================================================================

COMMENT ON TABLE jobs IS 'Background job queue with PostgreSQL-based locking';
COMMENT ON COLUMN jobs.priority IS 'Job priority: 10=high, 5=normal, 1=low';
COMMENT ON COLUMN jobs.idempotency_key IS 'Unique key to prevent duplicate job execution';
COMMENT ON COLUMN jobs.worker_id IS 'ID of worker process currently processing this job';

COMMENT ON TABLE schedules IS 'Cron-style schedules for recurring jobs';
COMMENT ON COLUMN schedules.cron_expression IS 'Standard 5-field cron expression';
COMMENT ON COLUMN schedules.interval IS 'Alternative to cron: hourly, 4hourly, daily, weekly';

COMMENT ON TABLE job_history IS 'Historical record of completed job executions';
COMMENT ON COLUMN job_history.duration_ms IS 'Auto-calculated duration in milliseconds';

COMMENT ON TABLE dead_letters IS 'Jobs that exhausted all retries for manual review';
COMMENT ON COLUMN dead_letters.errors IS 'JSON array of error messages from each attempt';

COMMENT ON TABLE webhooks IS 'Webhook configurations for event notifications';
COMMENT ON COLUMN webhooks.secret IS 'Shared secret for HMAC-SHA256 signature';
COMMENT ON COLUMN webhooks.events IS 'Array of event types this webhook subscribes to';

COMMENT ON TABLE webhook_deliveries IS 'Webhook call history and retry tracking';
COMMENT ON COLUMN webhook_deliveries.next_retry_at IS 'Time for next retry attempt (exponential backoff)';

COMMENT ON TABLE watchlist IS 'Firmenbuch entries being monitored for changes';
COMMENT ON COLUMN watchlist.last_snapshot IS 'Last fetched FB data as JSON for diff comparison';
