-- 019_foerderungsradar.sql
-- Förderungsradar Integration - AI-gestützte Förderungssuche
-- 74 Förderungsprogramme, Rule+LLM Matching, proaktive Benachrichtigungen

-- ============================================
-- FÖRDERUNGEN (Funding Programs)
-- ============================================

CREATE TABLE IF NOT EXISTS foerderungen (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Basic Info
    name VARCHAR(500) NOT NULL,
    short_name VARCHAR(100),
    description TEXT,
    provider VARCHAR(255) NOT NULL,  -- AWS, FFG, WKO, AMS, OeKB, EU, Bundesland

    -- Type
    type VARCHAR(50) NOT NULL,  -- zuschuss, kredit, garantie, beratung, kombination
    funding_rate_min DECIMAL(5,2),  -- e.g., 0.25 = 25%
    funding_rate_max DECIMAL(5,2),
    max_amount INTEGER,  -- in EUR
    min_amount INTEGER,

    -- Target Group
    target_size VARCHAR(50),  -- kmu, startup, grossunternehmen, alle
    target_age VARCHAR(50),   -- gruendung, etabliert, alle
    target_legal_forms TEXT[],  -- GmbH, AG, EPU, OG, KG
    target_industries TEXT[],   -- ÖNACE codes or descriptions
    target_states TEXT[],       -- Bundesländer: Wien, NÖ, OÖ, etc.

    -- Topics & Categories
    topics TEXT[] NOT NULL DEFAULT '{}',  -- innovation, digitalisierung, export, umwelt, etc.
    categories TEXT[] DEFAULT '{}',

    -- Requirements
    requirements TEXT,
    eligibility_criteria JSONB DEFAULT '{}',

    -- Deadlines
    application_deadline DATE,
    deadline_type VARCHAR(50),  -- fixed, rolling, budget_exhausted
    call_start DATE,
    call_end DATE,

    -- Links
    url VARCHAR(500),
    application_url VARCHAR(500),
    guideline_url VARCHAR(500),

    -- Combinations
    combinable_with UUID[] DEFAULT '{}',
    not_combinable_with UUID[] DEFAULT '{}',

    -- Status
    status VARCHAR(50) DEFAULT 'active',  -- active, upcoming, paused, closed
    is_highlighted BOOLEAN DEFAULT FALSE,

    -- Metadata
    source VARCHAR(100),  -- json_import, manual, api_sync
    source_id VARCHAR(100),
    last_updated_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT foerderungen_type_check CHECK (type IN ('zuschuss', 'kredit', 'garantie', 'beratung', 'kombination')),
    CONSTRAINT foerderungen_status_check CHECK (status IN ('active', 'upcoming', 'paused', 'closed')),
    CONSTRAINT foerderungen_target_size_check CHECK (target_size IS NULL OR target_size IN ('kmu', 'startup', 'grossunternehmen', 'alle')),
    CONSTRAINT foerderungen_deadline_type_check CHECK (deadline_type IS NULL OR deadline_type IN ('fixed', 'rolling', 'budget_exhausted'))
);

CREATE INDEX IF NOT EXISTS idx_foerderungen_provider ON foerderungen(provider);
CREATE INDEX IF NOT EXISTS idx_foerderungen_type ON foerderungen(type);
CREATE INDEX IF NOT EXISTS idx_foerderungen_status ON foerderungen(status);
CREATE INDEX IF NOT EXISTS idx_foerderungen_deadline ON foerderungen(application_deadline) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_foerderungen_topics ON foerderungen USING GIN(topics);
CREATE INDEX IF NOT EXISTS idx_foerderungen_states ON foerderungen USING GIN(target_states);

-- ============================================
-- UNTERNEHMENSPROFILE (Company Profiles)
-- ============================================

CREATE TABLE IF NOT EXISTS unternehmensprofile (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    account_id UUID REFERENCES accounts(id) ON DELETE SET NULL,

    -- Company Info
    name VARCHAR(255) NOT NULL,
    legal_form VARCHAR(50),      -- GmbH, AG, EPU, OG, KG, etc.
    founded_year INTEGER,
    state VARCHAR(50),           -- Bundesland
    district VARCHAR(100),

    -- Size
    employees_count INTEGER,
    annual_revenue INTEGER,      -- in EUR
    balance_total INTEGER,       -- Bilanzsumme

    -- Classification
    industry VARCHAR(255),       -- Branche
    onace_codes TEXT[] DEFAULT '{}',  -- ÖNACE-Codes
    is_startup BOOLEAN DEFAULT FALSE,

    -- Project
    project_description TEXT,
    investment_amount INTEGER,   -- Geplante Investition in EUR
    project_topics TEXT[] DEFAULT '{}',

    -- Derived Info (calculated)
    is_kmu BOOLEAN,              -- EU KMU Definition
    company_age_category VARCHAR(50),  -- startup, etabliert

    -- Status
    status VARCHAR(50) DEFAULT 'draft',  -- draft, complete
    derived_from_account BOOLEAN DEFAULT FALSE,
    last_search_at TIMESTAMPTZ,

    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT profile_status_check CHECK (status IN ('draft', 'complete')),
    CONSTRAINT profile_legal_form_check CHECK (legal_form IS NULL OR legal_form IN ('GmbH', 'AG', 'EPU', 'OG', 'KG', 'GesbR', 'eGen', 'Stiftung', 'Verein', 'Sonstige'))
);

CREATE INDEX IF NOT EXISTS idx_profile_tenant ON unternehmensprofile(tenant_id);
CREATE INDEX IF NOT EXISTS idx_profile_account ON unternehmensprofile(account_id);
CREATE INDEX IF NOT EXISTS idx_profile_state ON unternehmensprofile(state);
CREATE INDEX IF NOT EXISTS idx_profile_status ON unternehmensprofile(status);

-- ============================================
-- FÖRDERUNGS-SUCHEN (Search Sessions)
-- ============================================

CREATE TABLE IF NOT EXISTS foerderungs_suchen (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    profile_id UUID NOT NULL REFERENCES unternehmensprofile(id) ON DELETE CASCADE,

    -- Results
    total_foerderungen INTEGER DEFAULT 0,
    total_matches INTEGER DEFAULT 0,
    matches JSONB DEFAULT '[]',  -- Array of FoerderungsMatch

    -- Status
    status VARCHAR(50) DEFAULT 'pending',  -- pending, rule_filtering, llm_analysis, completed, failed
    phase VARCHAR(50),  -- rule_filter, llm_analysis
    progress INTEGER DEFAULT 0,  -- 0-100

    -- Timing
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error_message TEXT,

    -- Usage Tracking
    llm_tokens_used INTEGER DEFAULT 0,
    llm_cost_cents INTEGER DEFAULT 0,

    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT suchen_status_check CHECK (status IN ('pending', 'rule_filtering', 'llm_analysis', 'completed', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_suchen_tenant ON foerderungs_suchen(tenant_id);
CREATE INDEX IF NOT EXISTS idx_suchen_profile ON foerderungs_suchen(profile_id);
CREATE INDEX IF NOT EXISTS idx_suchen_status ON foerderungs_suchen(status);
CREATE INDEX IF NOT EXISTS idx_suchen_created ON foerderungs_suchen(created_at DESC);

-- ============================================
-- PROFIL-MONITORE (Monitoring Settings)
-- ============================================

CREATE TABLE IF NOT EXISTS profil_monitore (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    profile_id UUID NOT NULL REFERENCES unternehmensprofile(id) ON DELETE CASCADE,

    -- Settings
    is_active BOOLEAN DEFAULT TRUE,
    min_score_threshold INTEGER DEFAULT 70,  -- 0-100
    notification_email BOOLEAN DEFAULT TRUE,
    notification_portal BOOLEAN DEFAULT TRUE,
    digest_mode VARCHAR(50) DEFAULT 'immediate',  -- immediate, daily, weekly

    -- Tracking
    last_check_at TIMESTAMPTZ,
    last_notification_at TIMESTAMPTZ,
    matches_found INTEGER DEFAULT 0,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT monitor_profile_unique UNIQUE(profile_id),
    CONSTRAINT monitor_digest_mode_check CHECK (digest_mode IN ('immediate', 'daily', 'weekly')),
    CONSTRAINT monitor_score_check CHECK (min_score_threshold >= 0 AND min_score_threshold <= 100)
);

CREATE INDEX IF NOT EXISTS idx_monitor_tenant ON profil_monitore(tenant_id);
CREATE INDEX IF NOT EXISTS idx_monitor_active ON profil_monitore(is_active) WHERE is_active = TRUE;

-- ============================================
-- MONITOR-BENACHRICHTIGUNGEN (Notifications)
-- ============================================

CREATE TABLE IF NOT EXISTS monitor_notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    monitor_id UUID NOT NULL REFERENCES profil_monitore(id) ON DELETE CASCADE,
    foerderung_id UUID NOT NULL REFERENCES foerderungen(id) ON DELETE CASCADE,

    -- Match Info
    score INTEGER NOT NULL,
    match_summary TEXT,

    -- Delivery
    email_sent BOOLEAN DEFAULT FALSE,
    email_sent_at TIMESTAMPTZ,
    portal_notified BOOLEAN DEFAULT FALSE,

    -- User Action
    viewed_at TIMESTAMPTZ,
    dismissed BOOLEAN DEFAULT FALSE,

    created_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT notification_unique UNIQUE(monitor_id, foerderung_id)
);

CREATE INDEX IF NOT EXISTS idx_notifications_monitor ON monitor_notifications(monitor_id);
CREATE INDEX IF NOT EXISTS idx_notifications_foerderung ON monitor_notifications(foerderung_id);
CREATE INDEX IF NOT EXISTS idx_notifications_pending ON monitor_notifications(email_sent, created_at) WHERE email_sent = FALSE;

-- ============================================
-- FÖRDERUNGS-ANTRÄGE (Applications)
-- ============================================

CREATE TABLE IF NOT EXISTS foerderungs_antraege (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    account_id UUID REFERENCES accounts(id) ON DELETE SET NULL,
    foerderung_id UUID NOT NULL REFERENCES foerderungen(id) ON DELETE RESTRICT,
    profile_id UUID REFERENCES unternehmensprofile(id) ON DELETE SET NULL,
    suche_id UUID REFERENCES foerderungs_suchen(id) ON DELETE SET NULL,

    -- Application
    application_number VARCHAR(100),
    applied_at DATE,
    applied_amount INTEGER,

    -- Status
    status VARCHAR(50) DEFAULT 'planned',
    -- planned, drafting, submitted, in_review, approved, rejected, withdrawn
    status_changed_at TIMESTAMPTZ,

    -- Result
    approved_amount INTEGER,
    rejection_reason TEXT,
    decision_date DATE,

    -- Notes
    notes TEXT,
    documents JSONB DEFAULT '[]',  -- [{name, path, uploaded_at}]

    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT antraege_status_check CHECK (status IN ('planned', 'drafting', 'submitted', 'in_review', 'approved', 'rejected', 'withdrawn'))
);

CREATE INDEX IF NOT EXISTS idx_antraege_tenant ON foerderungs_antraege(tenant_id);
CREATE INDEX IF NOT EXISTS idx_antraege_account ON foerderungs_antraege(account_id);
CREATE INDEX IF NOT EXISTS idx_antraege_foerderung ON foerderungs_antraege(foerderung_id);
CREATE INDEX IF NOT EXISTS idx_antraege_status ON foerderungs_antraege(status);

-- ============================================
-- FÖRDERUNGS-IMPORTE (Import History)
-- ============================================

CREATE TABLE IF NOT EXISTS foerderungs_importe (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Import Info
    source VARCHAR(100) NOT NULL,  -- json_file, aws_api, ffg_api
    filename VARCHAR(255),

    -- Results
    total_records INTEGER DEFAULT 0,
    imported INTEGER DEFAULT 0,
    updated INTEGER DEFAULT 0,
    failed INTEGER DEFAULT 0,
    errors JSONB DEFAULT '[]',

    -- Status
    status VARCHAR(50) DEFAULT 'pending',  -- pending, processing, completed, failed

    imported_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT importe_status_check CHECK (status IN ('pending', 'processing', 'completed', 'failed'))
);

-- ============================================
-- VIEWS
-- ============================================

-- Active Förderungen with deadline status
CREATE OR REPLACE VIEW v_active_foerderungen AS
SELECT
    f.*,
    CASE
        WHEN f.application_deadline IS NULL THEN 'rolling'
        WHEN f.application_deadline > CURRENT_DATE + 30 THEN 'open'
        WHEN f.application_deadline > CURRENT_DATE THEN 'closing_soon'
        ELSE 'closed'
    END AS deadline_status,
    CASE
        WHEN f.application_deadline IS NULL THEN NULL
        ELSE f.application_deadline - CURRENT_DATE
    END AS days_until_deadline
FROM foerderungen f
WHERE f.status = 'active'
ORDER BY
    CASE WHEN f.application_deadline IS NULL THEN 1 ELSE 0 END,
    f.application_deadline ASC;

-- Förderungs-Statistiken per Provider
CREATE OR REPLACE VIEW v_foerderungs_stats AS
SELECT
    f.provider,
    COUNT(*) AS total_count,
    COUNT(*) FILTER (WHERE f.status = 'active') AS active_count,
    ROUND(AVG(f.max_amount)) AS avg_max_amount,
    COUNT(DISTINCT unnest) AS topic_count
FROM foerderungen f
LEFT JOIN LATERAL unnest(f.topics) ON true
GROUP BY f.provider
ORDER BY active_count DESC;

-- Antrags-Erfolgsquote per Tenant
CREATE OR REPLACE VIEW v_antrag_erfolgsquote AS
SELECT
    tenant_id,
    COUNT(*) AS total_applications,
    COUNT(*) FILTER (WHERE status = 'approved') AS approved,
    COUNT(*) FILTER (WHERE status = 'rejected') AS rejected,
    COUNT(*) FILTER (WHERE status IN ('submitted', 'in_review')) AS pending,
    ROUND(
        COUNT(*) FILTER (WHERE status = 'approved')::DECIMAL /
        NULLIF(COUNT(*) FILTER (WHERE status IN ('approved', 'rejected')), 0) * 100,
        1
    ) AS success_rate_pct,
    COALESCE(SUM(approved_amount) FILTER (WHERE status = 'approved'), 0) AS total_approved_amount
FROM foerderungs_antraege
GROUP BY tenant_id;

-- Pending notifications per tenant
CREATE OR REPLACE VIEW v_pending_foerderungs_notifications AS
SELECT
    pm.tenant_id,
    pm.profile_id,
    up.name AS profile_name,
    COUNT(mn.id) AS pending_count,
    MAX(mn.created_at) AS latest_notification_at
FROM profil_monitore pm
JOIN unternehmensprofile up ON up.id = pm.profile_id
LEFT JOIN monitor_notifications mn ON mn.monitor_id = pm.id AND mn.dismissed = FALSE AND mn.viewed_at IS NULL
WHERE pm.is_active = TRUE
GROUP BY pm.tenant_id, pm.profile_id, up.name;

-- ============================================
-- TRIGGERS
-- ============================================

-- Update profile is_kmu flag when size data changes
CREATE OR REPLACE FUNCTION update_profile_kmu_flag()
RETURNS TRIGGER AS $$
BEGIN
    -- EU KMU Definition: <250 employees AND (<€50M revenue OR <€43M balance)
    IF NEW.employees_count IS NOT NULL AND NEW.annual_revenue IS NOT NULL THEN
        NEW.is_kmu := (
            NEW.employees_count < 250 AND
            (NEW.annual_revenue < 50000000 OR COALESCE(NEW.balance_total, 0) < 43000000)
        );
    END IF;

    -- Calculate company age category
    IF NEW.founded_year IS NOT NULL THEN
        IF EXTRACT(YEAR FROM CURRENT_DATE) - NEW.founded_year <= 5 THEN
            NEW.company_age_category := 'startup';
        ELSE
            NEW.company_age_category := 'etabliert';
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_update_profile_kmu
BEFORE INSERT OR UPDATE OF employees_count, annual_revenue, balance_total, founded_year
ON unternehmensprofile
FOR EACH ROW EXECUTE FUNCTION update_profile_kmu_flag();

-- Auto-update updated_at timestamps
CREATE OR REPLACE FUNCTION update_foerderung_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_foerderungen_updated_at
BEFORE UPDATE ON foerderungen
FOR EACH ROW EXECUTE FUNCTION update_foerderung_updated_at();

CREATE TRIGGER trg_profile_updated_at
BEFORE UPDATE ON unternehmensprofile
FOR EACH ROW EXECUTE FUNCTION update_foerderung_updated_at();

CREATE TRIGGER trg_antraege_updated_at
BEFORE UPDATE ON foerderungs_antraege
FOR EACH ROW EXECUTE FUNCTION update_foerderung_updated_at();

-- ============================================
-- ROW LEVEL SECURITY
-- ============================================

ALTER TABLE unternehmensprofile ENABLE ROW LEVEL SECURITY;
ALTER TABLE foerderungs_suchen ENABLE ROW LEVEL SECURITY;
ALTER TABLE profil_monitore ENABLE ROW LEVEL SECURITY;
ALTER TABLE foerderungs_antraege ENABLE ROW LEVEL SECURITY;

-- Profiles: tenant isolation
CREATE POLICY profile_tenant_isolation ON unternehmensprofile
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

-- Searches: tenant isolation
CREATE POLICY suchen_tenant_isolation ON foerderungs_suchen
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

-- Monitors: tenant isolation
CREATE POLICY monitor_tenant_isolation ON profil_monitore
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

-- Applications: tenant isolation
CREATE POLICY antraege_tenant_isolation ON foerderungs_antraege
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

-- Förderungen: public read, admin write
-- (no RLS on foerderungen - all users can read, only admins can modify via application layer)
