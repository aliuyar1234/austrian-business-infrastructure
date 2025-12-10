-- Digital Signature (ID Austria) Migration
-- Creates tables for signature requests, signers, fields, batches, verification, and templates

-- ============================================
-- SIGNATURE REQUESTS TABLE
-- ============================================

CREATE TABLE IF NOT EXISTS signature_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,

    -- Request Info
    name VARCHAR(255),
    message TEXT,
    expires_at TIMESTAMPTZ NOT NULL,

    -- Status
    status VARCHAR(50) DEFAULT 'pending',  -- pending, in_progress, completed, expired, cancelled
    completed_at TIMESTAMPTZ,

    -- Workflow
    is_sequential BOOLEAN DEFAULT TRUE,  -- Sequential vs parallel signing
    current_signer_index INTEGER DEFAULT 0,

    -- Result
    signed_document_id UUID REFERENCES documents(id),

    -- Metadata
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT signature_requests_status_check CHECK (status IN ('pending', 'in_progress', 'completed', 'expired', 'cancelled'))
);

CREATE INDEX IF NOT EXISTS idx_sig_requests_tenant ON signature_requests(tenant_id);
CREATE INDEX IF NOT EXISTS idx_sig_requests_document ON signature_requests(document_id);
CREATE INDEX IF NOT EXISTS idx_sig_requests_status ON signature_requests(status);
CREATE INDEX IF NOT EXISTS idx_sig_requests_expires ON signature_requests(expires_at) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_sig_requests_created_by ON signature_requests(created_by);

-- ============================================
-- SIGNERS TABLE
-- ============================================

CREATE TABLE IF NOT EXISTS signers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    signature_request_id UUID NOT NULL REFERENCES signature_requests(id) ON DELETE CASCADE,

    -- Signer Info
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    order_index INTEGER NOT NULL DEFAULT 0,

    -- Token
    signing_token VARCHAR(255) NOT NULL UNIQUE,
    token_expires_at TIMESTAMPTZ NOT NULL,
    token_used BOOLEAN DEFAULT FALSE,

    -- Status
    status VARCHAR(50) DEFAULT 'pending',  -- pending, notified, signing, signed, expired
    notified_at TIMESTAMPTZ,
    signed_at TIMESTAMPTZ,

    -- Signature Data (after signing)
    certificate_subject VARCHAR(500),
    certificate_serial VARCHAR(100),
    certificate_issuer VARCHAR(500),
    signature_value TEXT,  -- Base64 encoded

    -- ID Austria Data
    idaustria_subject VARCHAR(255),
    idaustria_bpk VARCHAR(255),  -- bereichsspezifisches Personenkennzeichen (hashed)

    -- Reminders
    reminder_count INTEGER DEFAULT 0,
    last_reminder_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT signers_status_check CHECK (status IN ('pending', 'notified', 'signing', 'signed', 'expired'))
);

CREATE INDEX IF NOT EXISTS idx_signers_request ON signers(signature_request_id);
CREATE INDEX IF NOT EXISTS idx_signers_token ON signers(signing_token) WHERE NOT token_used;
CREATE INDEX IF NOT EXISTS idx_signers_status ON signers(status);
CREATE INDEX IF NOT EXISTS idx_signers_email ON signers(email);

-- ============================================
-- SIGNATURE FIELDS TABLE
-- ============================================

CREATE TABLE IF NOT EXISTS signature_fields (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    signature_request_id UUID NOT NULL REFERENCES signature_requests(id) ON DELETE CASCADE,
    signer_id UUID REFERENCES signers(id) ON DELETE SET NULL,

    -- Position (PDF points, origin at bottom-left)
    page INTEGER NOT NULL DEFAULT 1,
    x DECIMAL(10,2) NOT NULL,  -- From left (points)
    y DECIMAL(10,2) NOT NULL,  -- From bottom (points)
    width DECIMAL(10,2) NOT NULL DEFAULT 200,
    height DECIMAL(10,2) NOT NULL DEFAULT 50,

    -- Appearance
    show_name BOOLEAN DEFAULT TRUE,
    show_date BOOLEAN DEFAULT TRUE,
    show_reason BOOLEAN DEFAULT FALSE,
    reason TEXT,

    -- Background
    background_image_url VARCHAR(500),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sig_fields_request ON signature_fields(signature_request_id);
CREATE INDEX IF NOT EXISTS idx_sig_fields_signer ON signature_fields(signer_id);

-- ============================================
-- SIGNATURE BATCHES TABLE
-- ============================================

CREATE TABLE IF NOT EXISTS signature_batches (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Batch Info
    name VARCHAR(255),
    total_documents INTEGER NOT NULL DEFAULT 0,

    -- Status
    status VARCHAR(50) DEFAULT 'pending',  -- pending, signing, completed, partial_failure, cancelled
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    -- Results
    signed_count INTEGER DEFAULT 0,
    failed_count INTEGER DEFAULT 0,

    -- Signer (self-signing batches)
    signer_user_id UUID REFERENCES users(id),
    idaustria_session_id UUID,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT signature_batches_status_check CHECK (status IN ('pending', 'signing', 'completed', 'partial_failure', 'cancelled'))
);

CREATE INDEX IF NOT EXISTS idx_sig_batches_tenant ON signature_batches(tenant_id);
CREATE INDEX IF NOT EXISTS idx_sig_batches_status ON signature_batches(status);
CREATE INDEX IF NOT EXISTS idx_sig_batches_signer ON signature_batches(signer_user_id);

-- ============================================
-- SIGNATURE BATCH ITEMS TABLE
-- ============================================

CREATE TABLE IF NOT EXISTS signature_batch_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    batch_id UUID NOT NULL REFERENCES signature_batches(id) ON DELETE CASCADE,
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,

    -- Status
    status VARCHAR(50) DEFAULT 'pending',  -- pending, signing, signed, failed
    signed_at TIMESTAMPTZ,
    error_message TEXT,

    -- Result
    signed_document_id UUID REFERENCES documents(id),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT signature_batch_items_status_check CHECK (status IN ('pending', 'signing', 'signed', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_batch_items_batch ON signature_batch_items(batch_id);
CREATE INDEX IF NOT EXISTS idx_batch_items_document ON signature_batch_items(document_id);
CREATE INDEX IF NOT EXISTS idx_batch_items_status ON signature_batch_items(status);

-- ============================================
-- SIGNATURE VERIFICATIONS TABLE
-- ============================================

CREATE TABLE IF NOT EXISTS signature_verifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Document
    document_id UUID REFERENCES documents(id),
    document_hash VARCHAR(64) NOT NULL,
    original_filename VARCHAR(255),

    -- Results
    is_valid BOOLEAN NOT NULL,
    verification_status VARCHAR(50) NOT NULL,  -- valid, invalid, indeterminate, unknown

    -- Signature Info (detailed breakdown in JSONB)
    signatures JSONB DEFAULT '[]'::jsonb,
    signature_count INTEGER DEFAULT 0,

    -- Timestamp
    verified_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    verified_by UUID REFERENCES users(id),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT sig_verifications_status_check CHECK (verification_status IN ('valid', 'invalid', 'indeterminate', 'unknown'))
);

CREATE INDEX IF NOT EXISTS idx_verifications_tenant ON signature_verifications(tenant_id);
CREATE INDEX IF NOT EXISTS idx_verifications_document ON signature_verifications(document_id);
CREATE INDEX IF NOT EXISTS idx_verifications_hash ON signature_verifications(document_hash);

-- ============================================
-- SIGNATURE TEMPLATES TABLE
-- ============================================

CREATE TABLE IF NOT EXISTS signature_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Template Info
    name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Configuration
    is_sequential BOOLEAN DEFAULT TRUE,
    default_expiry_days INTEGER DEFAULT 14,
    default_message TEXT,

    -- Signer Templates (JSONB array)
    -- [{role: "gf1", name: "Geschäftsführer", email: "", order: 1}]
    signer_templates JSONB DEFAULT '[]'::jsonb,

    -- Field Templates (JSONB array)
    -- [{page: 1, x: 100, y: 100, width: 200, height: 50, signer_role: "gf1"}]
    field_templates JSONB DEFAULT '[]'::jsonb,

    -- Usage tracking
    use_count INTEGER DEFAULT 0,
    last_used_at TIMESTAMPTZ,

    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sig_templates_tenant ON signature_templates(tenant_id);
CREATE INDEX IF NOT EXISTS idx_sig_templates_name ON signature_templates(tenant_id, name);

-- ============================================
-- ID AUSTRIA SESSIONS TABLE
-- ============================================

CREATE TABLE IF NOT EXISTS idaustria_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- OIDC State
    state VARCHAR(255) NOT NULL UNIQUE,
    nonce VARCHAR(255) NOT NULL,
    code_verifier VARCHAR(255),  -- PKCE

    -- Context (either signer_id OR batch_id)
    signer_id UUID REFERENCES signers(id) ON DELETE CASCADE,
    batch_id UUID REFERENCES signature_batches(id) ON DELETE CASCADE,
    redirect_after VARCHAR(500),

    -- Tokens (encrypted at rest)
    access_token_encrypted TEXT,
    id_token TEXT,
    refresh_token_encrypted TEXT,
    token_expires_at TIMESTAMPTZ,

    -- User Info from ID Austria
    subject VARCHAR(255),  -- sub claim
    name VARCHAR(255),
    bpk_hash VARCHAR(64),  -- SHA-256 of BPK for privacy

    -- Status
    status VARCHAR(50) DEFAULT 'pending',  -- pending, authenticated, used, expired, failed
    error_message TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT idaustria_sessions_status_check CHECK (status IN ('pending', 'authenticated', 'used', 'expired', 'failed')),
    CONSTRAINT idaustria_sessions_context_check CHECK (
        (signer_id IS NOT NULL AND batch_id IS NULL) OR
        (signer_id IS NULL AND batch_id IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_idaustria_state ON idaustria_sessions(state);
CREATE INDEX IF NOT EXISTS idx_idaustria_signer ON idaustria_sessions(signer_id);
CREATE INDEX IF NOT EXISTS idx_idaustria_batch ON idaustria_sessions(batch_id);
CREATE INDEX IF NOT EXISTS idx_idaustria_status ON idaustria_sessions(status);

-- ============================================
-- SIGNATURE AUDIT LOGS TABLE
-- ============================================

CREATE TABLE IF NOT EXISTS signature_audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,

    -- Reference (at least one should be set)
    signature_request_id UUID REFERENCES signature_requests(id) ON DELETE SET NULL,
    signer_id UUID REFERENCES signers(id) ON DELETE SET NULL,
    batch_id UUID REFERENCES signature_batches(id) ON DELETE SET NULL,
    verification_id UUID REFERENCES signature_verifications(id) ON DELETE SET NULL,

    -- Event
    event_type VARCHAR(100) NOT NULL,
    -- Events: request_created, request_cancelled, request_expired,
    --         signer_notified, signer_reminded, signing_started,
    --         signing_completed, signing_failed, batch_started,
    --         batch_completed, verification_performed

    -- Details
    details JSONB DEFAULT '{}'::jsonb,

    -- Actor
    actor_type VARCHAR(50),  -- user, system, signer
    actor_id VARCHAR(255),
    actor_ip VARCHAR(45),
    actor_user_agent TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sig_audit_tenant ON signature_audit_logs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_sig_audit_request ON signature_audit_logs(signature_request_id);
CREATE INDEX IF NOT EXISTS idx_sig_audit_signer ON signature_audit_logs(signer_id);
CREATE INDEX IF NOT EXISTS idx_sig_audit_batch ON signature_audit_logs(batch_id);
CREATE INDEX IF NOT EXISTS idx_sig_audit_type ON signature_audit_logs(event_type);
CREATE INDEX IF NOT EXISTS idx_sig_audit_date ON signature_audit_logs(created_at);

-- ============================================
-- SIGNATURE USAGE TABLE (for billing)
-- ============================================

CREATE TABLE IF NOT EXISTS signature_usage (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Reference
    signature_request_id UUID REFERENCES signature_requests(id) ON DELETE SET NULL,
    batch_id UUID REFERENCES signature_batches(id) ON DELETE SET NULL,

    -- Usage
    signature_count INTEGER NOT NULL DEFAULT 1,
    cost_cents INTEGER,  -- In Euro-Cents (for tracking A-Trust costs)

    -- Period
    usage_date DATE NOT NULL DEFAULT CURRENT_DATE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sig_usage_tenant ON signature_usage(tenant_id);
CREATE INDEX IF NOT EXISTS idx_sig_usage_tenant_date ON signature_usage(tenant_id, usage_date);
CREATE INDEX IF NOT EXISTS idx_sig_usage_date ON signature_usage(usage_date);

-- ============================================
-- VIEWS
-- ============================================

-- Pending Signatures View
CREATE OR REPLACE VIEW v_pending_signatures AS
SELECT
    sr.id AS request_id,
    sr.tenant_id,
    sr.name AS request_name,
    sr.status AS request_status,
    d.title AS document_title,
    s.id AS signer_id,
    s.email AS signer_email,
    s.name AS signer_name,
    s.status AS signer_status,
    s.order_index,
    sr.expires_at,
    EXTRACT(DAY FROM sr.expires_at - NOW()) AS days_until_expiry,
    sr.is_sequential,
    sr.current_signer_index,
    sr.created_at AS request_created_at
FROM signature_requests sr
JOIN documents d ON sr.document_id = d.id
JOIN signers s ON s.signature_request_id = sr.id
WHERE sr.status IN ('pending', 'in_progress')
  AND s.status IN ('pending', 'notified', 'signing')
ORDER BY sr.expires_at ASC;

-- Signature Statistics View
CREATE OR REPLACE VIEW v_signature_stats AS
SELECT
    tenant_id,
    DATE_TRUNC('month', created_at) AS month,
    COUNT(*) AS total_requests,
    COUNT(*) FILTER (WHERE status = 'completed') AS completed_count,
    COUNT(*) FILTER (WHERE status = 'expired') AS expired_count,
    COUNT(*) FILTER (WHERE status = 'cancelled') AS cancelled_count,
    COUNT(*) FILTER (WHERE status IN ('pending', 'in_progress')) AS pending_count,
    AVG(EXTRACT(EPOCH FROM (completed_at - created_at)) / 3600)
        FILTER (WHERE status = 'completed') AS avg_hours_to_complete
FROM signature_requests
GROUP BY tenant_id, DATE_TRUNC('month', created_at);

-- Overdue Signatures View
CREATE OR REPLACE VIEW v_overdue_signatures AS
SELECT
    sr.id AS request_id,
    sr.tenant_id,
    sr.name AS request_name,
    sr.expires_at,
    EXTRACT(DAY FROM NOW() - sr.expires_at) AS days_overdue,
    COUNT(s.id) FILTER (WHERE s.status IN ('pending', 'notified')) AS unsigned_signers
FROM signature_requests sr
JOIN signers s ON s.signature_request_id = sr.id
WHERE sr.status = 'pending'
  AND sr.expires_at < NOW()
GROUP BY sr.id, sr.tenant_id, sr.name, sr.expires_at;

-- ============================================
-- FUNCTIONS
-- ============================================

-- Function to update signature request status based on signers
CREATE OR REPLACE FUNCTION update_signature_request_status()
RETURNS TRIGGER AS $$
DECLARE
    v_all_signed BOOLEAN;
    v_any_signed BOOLEAN;
    v_request_id UUID;
BEGIN
    v_request_id := COALESCE(NEW.signature_request_id, OLD.signature_request_id);

    -- Check if all signers have signed
    SELECT
        bool_and(status = 'signed'),
        bool_or(status = 'signed')
    INTO v_all_signed, v_any_signed
    FROM signers
    WHERE signature_request_id = v_request_id;

    -- Update request status
    IF v_all_signed THEN
        UPDATE signature_requests
        SET status = 'completed',
            completed_at = NOW(),
            updated_at = NOW()
        WHERE id = v_request_id;
    ELSIF v_any_signed THEN
        UPDATE signature_requests
        SET status = 'in_progress',
            updated_at = NOW()
        WHERE id = v_request_id AND status = 'pending';
    END IF;

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Trigger for signer status changes
DROP TRIGGER IF EXISTS trg_update_signature_request_status ON signers;
CREATE TRIGGER trg_update_signature_request_status
    AFTER UPDATE OF status ON signers
    FOR EACH ROW
    EXECUTE FUNCTION update_signature_request_status();

-- Function to update batch status based on items
CREATE OR REPLACE FUNCTION update_signature_batch_status()
RETURNS TRIGGER AS $$
DECLARE
    v_total INTEGER;
    v_signed INTEGER;
    v_failed INTEGER;
    v_batch_id UUID;
BEGIN
    v_batch_id := COALESCE(NEW.batch_id, OLD.batch_id);

    -- Count items by status
    SELECT
        COUNT(*),
        COUNT(*) FILTER (WHERE status = 'signed'),
        COUNT(*) FILTER (WHERE status = 'failed')
    INTO v_total, v_signed, v_failed
    FROM signature_batch_items
    WHERE batch_id = v_batch_id;

    -- Update batch counts
    UPDATE signature_batches
    SET signed_count = v_signed,
        failed_count = v_failed
    WHERE id = v_batch_id;

    -- Update batch status if all processed
    IF (v_signed + v_failed) = v_total THEN
        UPDATE signature_batches
        SET status = CASE
            WHEN v_failed = 0 THEN 'completed'
            WHEN v_signed = 0 THEN 'partial_failure'
            ELSE 'partial_failure'
        END,
        completed_at = NOW()
        WHERE id = v_batch_id;
    END IF;

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Trigger for batch item status changes
DROP TRIGGER IF EXISTS trg_update_signature_batch_status ON signature_batch_items;
CREATE TRIGGER trg_update_signature_batch_status
    AFTER UPDATE OF status ON signature_batch_items
    FOR EACH ROW
    EXECUTE FUNCTION update_signature_batch_status();

-- ============================================
-- RLS POLICIES
-- ============================================

-- Enable RLS on all signature tables
ALTER TABLE signature_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE signers ENABLE ROW LEVEL SECURITY;
ALTER TABLE signature_fields ENABLE ROW LEVEL SECURITY;
ALTER TABLE signature_batches ENABLE ROW LEVEL SECURITY;
ALTER TABLE signature_batch_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE signature_verifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE signature_templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE signature_usage ENABLE ROW LEVEL SECURITY;
ALTER TABLE signature_audit_logs ENABLE ROW LEVEL SECURITY;

-- RLS policy for signature_requests
CREATE POLICY signature_requests_tenant_isolation ON signature_requests
    FOR ALL
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

-- RLS policy for signers (through request)
CREATE POLICY signers_tenant_isolation ON signers
    FOR ALL
    USING (
        signature_request_id IN (
            SELECT id FROM signature_requests
            WHERE tenant_id = current_setting('app.tenant_id', true)::uuid
        )
    );

-- RLS policy for signature_fields (through request)
CREATE POLICY signature_fields_tenant_isolation ON signature_fields
    FOR ALL
    USING (
        signature_request_id IN (
            SELECT id FROM signature_requests
            WHERE tenant_id = current_setting('app.tenant_id', true)::uuid
        )
    );

-- RLS policy for signature_batches
CREATE POLICY signature_batches_tenant_isolation ON signature_batches
    FOR ALL
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

-- RLS policy for signature_batch_items (through batch)
CREATE POLICY signature_batch_items_tenant_isolation ON signature_batch_items
    FOR ALL
    USING (
        batch_id IN (
            SELECT id FROM signature_batches
            WHERE tenant_id = current_setting('app.tenant_id', true)::uuid
        )
    );

-- RLS policy for signature_verifications
CREATE POLICY signature_verifications_tenant_isolation ON signature_verifications
    FOR ALL
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

-- RLS policy for signature_templates
CREATE POLICY signature_templates_tenant_isolation ON signature_templates
    FOR ALL
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

-- RLS policy for signature_usage
CREATE POLICY signature_usage_tenant_isolation ON signature_usage
    FOR ALL
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

-- RLS policy for signature_audit_logs
CREATE POLICY signature_audit_logs_tenant_isolation ON signature_audit_logs
    FOR ALL
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

-- Note: idaustria_sessions does not have tenant_id directly,
-- access is controlled through the signing flow
