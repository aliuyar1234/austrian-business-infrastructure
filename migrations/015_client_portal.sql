-- Client Portal Migration
-- Creates tables for client management, uploads, shares, approvals, messaging, and branding

-- ============================================
-- EXTEND USERS TABLE FOR CLIENT ROLE
-- ============================================

-- Add client_id and is_client columns to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS client_id UUID;
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_client BOOLEAN DEFAULT FALSE;

-- Update role constraint to include 'client' role
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('owner', 'admin', 'member', 'viewer', 'client'));

CREATE INDEX IF NOT EXISTS idx_users_client ON users(client_id) WHERE is_client = TRUE;

-- ============================================
-- CLIENTS TABLE
-- ============================================

CREATE TABLE IF NOT EXISTS clients (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id),  -- After activation

    -- Profile
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    company_name VARCHAR(255),
    phone VARCHAR(50),

    -- Status
    status VARCHAR(50) DEFAULT 'invited',  -- invited, active, inactive
    invited_at TIMESTAMPTZ,
    activated_at TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,

    -- Settings
    notification_email BOOLEAN DEFAULT TRUE,
    notification_portal BOOLEAN DEFAULT TRUE,
    language VARCHAR(10) DEFAULT 'de',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT clients_status_check CHECK (status IN ('invited', 'active', 'inactive')),
    CONSTRAINT clients_email_tenant_unique UNIQUE (tenant_id, email)
);

CREATE INDEX idx_clients_tenant ON clients(tenant_id);
CREATE INDEX idx_clients_user ON clients(user_id);
CREATE INDEX idx_clients_status ON clients(status);
CREATE INDEX idx_clients_email ON clients(email);

-- Add foreign key for users.client_id after clients table exists
ALTER TABLE users ADD CONSTRAINT fk_users_client FOREIGN KEY (client_id) REFERENCES clients(id) ON DELETE SET NULL;

-- ============================================
-- CLIENT ACCOUNT ACCESS
-- ============================================

CREATE TABLE IF NOT EXISTS client_account_access (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,

    -- Permissions
    can_upload BOOLEAN DEFAULT TRUE,
    can_view_documents BOOLEAN DEFAULT TRUE,
    can_approve BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT client_account_access_unique UNIQUE (client_id, account_id)
);

CREATE INDEX idx_client_access_client ON client_account_access(client_id);
CREATE INDEX idx_client_access_account ON client_account_access(account_id);

-- ============================================
-- CLIENT INVITATIONS
-- ============================================

CREATE TABLE IF NOT EXISTS client_invitations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,

    -- Token
    token VARCHAR(255) NOT NULL UNIQUE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,

    -- Status
    used BOOLEAN DEFAULT FALSE,
    used_at TIMESTAMPTZ,

    -- Metadata
    invited_by UUID REFERENCES users(id),
    email_sent_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_invitations_token ON client_invitations(token) WHERE NOT used;
CREATE INDEX idx_invitations_token_hash ON client_invitations(token_hash) WHERE NOT used;
CREATE INDEX idx_invitations_client ON client_invitations(client_id);

-- ============================================
-- CLIENT UPLOADS
-- ============================================

CREATE TABLE IF NOT EXISTS client_uploads (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id UUID NOT NULL REFERENCES clients(id),
    account_id UUID NOT NULL REFERENCES accounts(id),

    -- File Info
    filename VARCHAR(500) NOT NULL,
    storage_path VARCHAR(500) NOT NULL,
    file_size INTEGER NOT NULL,
    mime_type VARCHAR(100),
    content_hash VARCHAR(64),

    -- Metadata
    category VARCHAR(100),  -- rechnung, beleg, vertrag, kontoauszug, sonstiges
    note TEXT,
    upload_date TIMESTAMPTZ DEFAULT NOW(),

    -- Processing
    status VARCHAR(50) DEFAULT 'new',  -- new, processed, archived
    processed_by UUID REFERENCES users(id),
    processed_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT client_uploads_status_check CHECK (status IN ('new', 'processed', 'archived')),
    CONSTRAINT client_uploads_category_check CHECK (
        category IS NULL OR category IN ('rechnung', 'beleg', 'vertrag', 'kontoauszug', 'sonstiges')
    )
);

CREATE INDEX idx_uploads_client ON client_uploads(client_id);
CREATE INDEX idx_uploads_account ON client_uploads(account_id);
CREATE INDEX idx_uploads_status ON client_uploads(status);
CREATE INDEX idx_uploads_date ON client_uploads(upload_date DESC);

-- ============================================
-- DOCUMENT SHARES
-- ============================================

CREATE TABLE IF NOT EXISTS document_shares (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,

    -- Share Info
    shared_by UUID REFERENCES users(id),
    shared_at TIMESTAMPTZ DEFAULT NOW(),

    -- Access
    can_download BOOLEAN DEFAULT TRUE,
    expires_at TIMESTAMPTZ,  -- Optional expiry

    -- Tracking
    first_viewed_at TIMESTAMPTZ,
    view_count INTEGER DEFAULT 0,

    CONSTRAINT document_shares_unique UNIQUE (document_id, client_id)
);

CREATE INDEX idx_shares_document ON document_shares(document_id);
CREATE INDEX idx_shares_client ON document_shares(client_id);

-- ============================================
-- APPROVAL REQUESTS
-- ============================================

CREATE TABLE IF NOT EXISTS approval_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID NOT NULL REFERENCES documents(id),
    client_id UUID NOT NULL REFERENCES clients(id),

    -- Request
    requested_by UUID REFERENCES users(id),
    requested_at TIMESTAMPTZ DEFAULT NOW(),
    message TEXT,

    -- Response
    status VARCHAR(50) DEFAULT 'pending',  -- pending, approved, rejected, revision_requested
    responded_at TIMESTAMPTZ,
    response_comment TEXT,

    -- Tracking
    reminder_sent_at TIMESTAMPTZ,
    reminder_count INTEGER DEFAULT 0,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT approval_requests_status_check CHECK (status IN ('pending', 'approved', 'rejected', 'revision_requested'))
);

CREATE INDEX idx_approvals_document ON approval_requests(document_id);
CREATE INDEX idx_approvals_client ON approval_requests(client_id);
CREATE INDEX idx_approvals_status ON approval_requests(status);
CREATE INDEX idx_approvals_pending ON approval_requests(client_id) WHERE status = 'pending';

-- ============================================
-- MESSAGES
-- ============================================

CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    client_id UUID NOT NULL REFERENCES clients(id),

    -- Participants
    sender_type VARCHAR(20) NOT NULL,  -- staff, client
    sender_user_id UUID REFERENCES users(id),

    -- Content
    content TEXT NOT NULL,
    has_attachment BOOLEAN DEFAULT FALSE,

    -- Status
    read_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT messages_sender_type_check CHECK (sender_type IN ('staff', 'client'))
);

CREATE INDEX idx_messages_client ON messages(client_id);
CREATE INDEX idx_messages_tenant ON messages(tenant_id);
CREATE INDEX idx_messages_created ON messages(created_at DESC);
CREATE INDEX idx_messages_unread ON messages(client_id) WHERE read_at IS NULL;

-- ============================================
-- MESSAGE ATTACHMENTS
-- ============================================

CREATE TABLE IF NOT EXISTS message_attachments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,

    filename VARCHAR(500) NOT NULL,
    storage_path VARCHAR(500) NOT NULL,
    file_size INTEGER NOT NULL,
    mime_type VARCHAR(100),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_message_attachments_message ON message_attachments(message_id);

-- ============================================
-- CLIENT TASKS
-- ============================================

CREATE TABLE IF NOT EXISTS client_tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id UUID NOT NULL REFERENCES clients(id),
    account_id UUID REFERENCES accounts(id),

    -- Task
    title VARCHAR(500) NOT NULL,
    description TEXT,
    task_type VARCHAR(100),  -- belege_liefern, unterschrift, freigabe, sonstiges
    deadline DATE,

    -- Status
    status VARCHAR(50) DEFAULT 'open',  -- open, in_progress, completed, cancelled
    completed_at TIMESTAMPTZ,

    -- Linked Resources
    linked_document_id UUID REFERENCES documents(id),
    linked_upload_id UUID REFERENCES client_uploads(id),
    linked_approval_id UUID REFERENCES approval_requests(id),

    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT client_tasks_status_check CHECK (status IN ('open', 'in_progress', 'completed', 'cancelled')),
    CONSTRAINT client_tasks_type_check CHECK (
        task_type IS NULL OR task_type IN ('belege_liefern', 'unterschrift', 'freigabe', 'sonstiges')
    )
);

CREATE INDEX idx_client_tasks_client ON client_tasks(client_id);
CREATE INDEX idx_client_tasks_status ON client_tasks(status);
CREATE INDEX idx_client_tasks_deadline ON client_tasks(deadline) WHERE status = 'open';

-- ============================================
-- CLIENT GROUPS
-- ============================================

CREATE TABLE IF NOT EXISTS client_groups (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),

    name VARCHAR(255) NOT NULL,
    description TEXT,
    color VARCHAR(7),  -- Hex color for UI

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT client_groups_tenant_name_unique UNIQUE (tenant_id, name)
);

CREATE INDEX idx_client_groups_tenant ON client_groups(tenant_id);

-- ============================================
-- CLIENT GROUP MEMBERS
-- ============================================

CREATE TABLE IF NOT EXISTS client_group_members (
    client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES client_groups(id) ON DELETE CASCADE,
    added_at TIMESTAMPTZ DEFAULT NOW(),

    PRIMARY KEY (client_id, group_id)
);

CREATE INDEX idx_group_members_client ON client_group_members(client_id);
CREATE INDEX idx_group_members_group ON client_group_members(group_id);

-- ============================================
-- TENANT BRANDING
-- ============================================

CREATE TABLE IF NOT EXISTS tenant_branding (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),

    -- Logo
    logo_url VARCHAR(500),
    favicon_url VARCHAR(500),

    -- Colors
    primary_color VARCHAR(7) DEFAULT '#0066CC',
    secondary_color VARCHAR(7) DEFAULT '#F0F4F8',
    accent_color VARCHAR(7) DEFAULT '#00AA55',

    -- Text
    portal_title VARCHAR(255),
    welcome_message TEXT,
    footer_text VARCHAR(255),

    -- Feature Flags
    show_branding BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT tenant_branding_tenant_unique UNIQUE (tenant_id)
);

CREATE INDEX idx_branding_tenant ON tenant_branding(tenant_id);

-- ============================================
-- VIEWS
-- ============================================

-- Active Clients View
CREATE OR REPLACE VIEW v_active_clients AS
SELECT
    c.id,
    c.tenant_id,
    c.email,
    c.name,
    c.company_name,
    c.last_login_at,
    COUNT(DISTINCT caa.account_id) AS account_count,
    COUNT(DISTINCT cu.id) FILTER (WHERE cu.status = 'new') AS pending_uploads,
    COUNT(DISTINCT ar.id) FILTER (WHERE ar.status = 'pending') AS pending_approvals
FROM clients c
LEFT JOIN client_account_access caa ON c.id = caa.client_id
LEFT JOIN client_uploads cu ON c.id = cu.client_id
LEFT JOIN approval_requests ar ON c.id = ar.client_id
WHERE c.status = 'active'
GROUP BY c.id;

-- Pending Approvals View
CREATE OR REPLACE VIEW v_pending_approvals AS
SELECT
    ar.id,
    ar.document_id,
    d.title AS document_title,
    ar.client_id,
    c.name AS client_name,
    c.email AS client_email,
    ar.requested_at,
    ar.message,
    t.name AS tenant_name
FROM approval_requests ar
JOIN documents d ON ar.document_id = d.id
JOIN clients c ON ar.client_id = c.id
JOIN accounts a ON d.account_id = a.id
JOIN tenants t ON a.tenant_id = t.id
WHERE ar.status = 'pending'
ORDER BY ar.requested_at ASC;

-- Unread Messages View
CREATE OR REPLACE VIEW v_unread_messages AS
SELECT
    m.client_id,
    c.name AS client_name,
    COUNT(*) AS unread_count,
    MAX(m.created_at) AS last_message_at
FROM messages m
JOIN clients c ON m.client_id = c.id
WHERE m.read_at IS NULL
  AND m.sender_type = 'client'
GROUP BY m.client_id, c.name;

-- ============================================
-- TRIGGERS
-- ============================================

-- Update timestamps
CREATE TRIGGER update_clients_updated_at
    BEFORE UPDATE ON clients
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_client_tasks_updated_at
    BEFORE UPDATE ON client_tasks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_tenant_branding_updated_at
    BEFORE UPDATE ON tenant_branding
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
