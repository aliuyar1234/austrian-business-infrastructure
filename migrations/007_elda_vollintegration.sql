-- Migration: 007_elda_vollintegration
-- Description: ELDA full integration - mBGM, L16, Databox, Meldungen, Protokolle
-- Spec: 013-elda-vollintegration

-- ============================================================================
-- ELDA Accounts
-- ============================================================================

CREATE TABLE IF NOT EXISTS elda_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,

    -- Credentials
    dienstgeber_nummer VARCHAR(20) NOT NULL,
    beitragskontonummer VARCHAR(20),
    certificate_path VARCHAR(500),
    certificate_password_encrypted TEXT,  -- AES-256-GCM encrypted
    certificate_expires_at TIMESTAMPTZ,

    -- Status
    last_sync_at TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,
    status VARCHAR(50) DEFAULT 'active',

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(account_id)
);

CREATE INDEX IF NOT EXISTS idx_elda_accounts_account ON elda_accounts(account_id);
CREATE INDEX IF NOT EXISTS idx_elda_accounts_cert_expiry ON elda_accounts(certificate_expires_at);
CREATE INDEX IF NOT EXISTS idx_elda_accounts_status ON elda_accounts(status);

-- ============================================================================
-- mBGM (Monatliche Beitragsgrundlagenmeldung)
-- ============================================================================

CREATE TABLE IF NOT EXISTS mbgm (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    elda_account_id UUID NOT NULL REFERENCES elda_accounts(id) ON DELETE CASCADE,

    -- Period
    year INTEGER NOT NULL,
    month INTEGER NOT NULL CHECK (month BETWEEN 1 AND 12),

    -- Status
    status VARCHAR(50) DEFAULT 'draft',  -- draft, validated, submitted, accepted, rejected
    protokollnummer VARCHAR(50),

    -- Metadata
    total_dienstnehmer INTEGER DEFAULT 0,
    total_beitragsgrundlage DECIMAL(15,2),

    -- ELDA Response
    submitted_at TIMESTAMPTZ,
    response_received_at TIMESTAMPTZ,
    request_xml TEXT,
    response_xml TEXT,
    error_message TEXT,
    error_code VARCHAR(20),

    -- Correction
    is_correction BOOLEAN DEFAULT FALSE,
    corrects_id UUID REFERENCES mbgm(id),

    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(elda_account_id, year, month) WHERE NOT is_correction
);

CREATE INDEX IF NOT EXISTS idx_mbgm_elda_account ON mbgm(elda_account_id);
CREATE INDEX IF NOT EXISTS idx_mbgm_period ON mbgm(year, month);
CREATE INDEX IF NOT EXISTS idx_mbgm_status ON mbgm(status);
CREATE INDEX IF NOT EXISTS idx_mbgm_protokoll ON mbgm(protokollnummer);

-- ============================================================================
-- mBGM Positionen (einzelne Dienstnehmer)
-- ============================================================================

CREATE TABLE IF NOT EXISTS mbgm_positionen (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mbgm_id UUID NOT NULL REFERENCES mbgm(id) ON DELETE CASCADE,

    -- Dienstnehmer
    sv_nummer VARCHAR(10) NOT NULL,
    familienname VARCHAR(100) NOT NULL,
    vorname VARCHAR(100) NOT NULL,
    geburtsdatum DATE,

    -- Beschäftigung
    beitragsgruppe VARCHAR(10) NOT NULL,
    beitragsgrundlage DECIMAL(10,2) NOT NULL,
    sonderzahlung DECIMAL(10,2) DEFAULT 0,

    -- Zeitraum (wenn abweichend vom Monat)
    von_datum DATE,
    bis_datum DATE,

    -- Stunden (optional)
    wochenstunden DECIMAL(5,2),

    -- Validation
    is_valid BOOLEAN DEFAULT TRUE,
    validation_errors JSONB DEFAULT '[]',

    position_index INTEGER NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mbgm_positionen_mbgm ON mbgm_positionen(mbgm_id);
CREATE INDEX IF NOT EXISTS idx_mbgm_positionen_sv ON mbgm_positionen(sv_nummer);

-- ============================================================================
-- Lohnzettel Batches
-- ============================================================================

CREATE TABLE IF NOT EXISTS lohnzettel_batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    elda_account_id UUID NOT NULL REFERENCES elda_accounts(id) ON DELETE CASCADE,

    -- Period
    year INTEGER NOT NULL,

    -- Statistics
    total_lohnzettel INTEGER DEFAULT 0,
    submitted_count INTEGER DEFAULT 0,
    accepted_count INTEGER DEFAULT 0,
    rejected_count INTEGER DEFAULT 0,

    -- Status
    status VARCHAR(50) DEFAULT 'draft',  -- draft, submitting, completed, partial_failure

    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_l16_batches_elda_account ON lohnzettel_batches(elda_account_id);
CREATE INDEX IF NOT EXISTS idx_l16_batches_year ON lohnzettel_batches(year);
CREATE INDEX IF NOT EXISTS idx_l16_batches_status ON lohnzettel_batches(status);

-- ============================================================================
-- Lohnzettel (L16)
-- ============================================================================

CREATE TABLE IF NOT EXISTS lohnzettel (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    elda_account_id UUID NOT NULL REFERENCES elda_accounts(id) ON DELETE CASCADE,

    -- Period
    year INTEGER NOT NULL,

    -- Dienstnehmer
    sv_nummer VARCHAR(10) NOT NULL,
    familienname VARCHAR(100) NOT NULL,
    vorname VARCHAR(100) NOT NULL,
    geburtsdatum DATE,

    -- L16 Fields (structured as per BMF spec)
    l16_data JSONB NOT NULL,  -- All L16 fields (kz210, kz215, kz220, etc.)

    -- Status
    status VARCHAR(50) DEFAULT 'draft',
    protokollnummer VARCHAR(50),

    -- Batch reference
    batch_id UUID REFERENCES lohnzettel_batches(id),

    -- ELDA Response
    submitted_at TIMESTAMPTZ,
    request_xml TEXT,
    response_xml TEXT,
    error_message TEXT,
    error_code VARCHAR(20),

    -- Correction
    is_berichtigung BOOLEAN DEFAULT FALSE,
    berichtigt_id UUID REFERENCES lohnzettel(id),

    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_lohnzettel_elda_account ON lohnzettel(elda_account_id);
CREATE INDEX IF NOT EXISTS idx_lohnzettel_year ON lohnzettel(year);
CREATE INDEX IF NOT EXISTS idx_lohnzettel_sv ON lohnzettel(sv_nummer);
CREATE INDEX IF NOT EXISTS idx_lohnzettel_batch ON lohnzettel(batch_id);
CREATE INDEX IF NOT EXISTS idx_lohnzettel_status ON lohnzettel(status);

-- ============================================================================
-- ELDA Meldungen (Anmeldung, Abmeldung, Änderung)
-- ============================================================================

CREATE TABLE IF NOT EXISTS elda_meldungen (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    elda_account_id UUID NOT NULL REFERENCES elda_accounts(id) ON DELETE CASCADE,

    -- Type
    meldung_type VARCHAR(50) NOT NULL,  -- anmeldung, abmeldung, aenderung

    -- Dienstnehmer
    sv_nummer VARCHAR(10),
    familienname VARCHAR(100),
    vorname VARCHAR(100),

    -- Payload
    payload_json JSONB NOT NULL,  -- All fields
    xml_sent TEXT,                 -- Original XML

    -- Status
    status VARCHAR(50) DEFAULT 'draft',
    protokollnummer VARCHAR(50),

    -- Response
    submitted_at TIMESTAMPTZ,
    response_xml TEXT,
    error_message TEXT,
    error_code VARCHAR(20),

    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_elda_meldungen_account ON elda_meldungen(elda_account_id);
CREATE INDEX IF NOT EXISTS idx_elda_meldungen_type ON elda_meldungen(meldung_type);
CREATE INDEX IF NOT EXISTS idx_elda_meldungen_sv ON elda_meldungen(sv_nummer);
CREATE INDEX IF NOT EXISTS idx_elda_meldungen_status ON elda_meldungen(status);
CREATE INDEX IF NOT EXISTS idx_elda_meldungen_protokoll ON elda_meldungen(protokollnummer);

-- ============================================================================
-- ELDA Documents (Databox)
-- ============================================================================

CREATE TABLE IF NOT EXISTS elda_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    elda_account_id UUID NOT NULL REFERENCES elda_accounts(id) ON DELETE CASCADE,

    -- Document Info
    external_id VARCHAR(255),
    document_type VARCHAR(100),
    title VARCHAR(500),
    sender VARCHAR(255),
    received_at TIMESTAMPTZ,

    -- Storage
    storage_path VARCHAR(500),
    file_size INTEGER,
    mime_type VARCHAR(100) DEFAULT 'application/pdf',
    content_hash VARCHAR(64),

    -- Status
    status VARCHAR(50) DEFAULT 'new',  -- new, read, archived

    -- Metadata
    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(elda_account_id, external_id)
);

CREATE INDEX IF NOT EXISTS idx_elda_documents_account ON elda_documents(elda_account_id);
CREATE INDEX IF NOT EXISTS idx_elda_documents_status ON elda_documents(status);
CREATE INDEX IF NOT EXISTS idx_elda_documents_received ON elda_documents(received_at DESC);
CREATE INDEX IF NOT EXISTS idx_elda_documents_type ON elda_documents(document_type);

-- ============================================================================
-- ELDA Protokolle (Audit Trail)
-- ============================================================================

CREATE TABLE IF NOT EXISTS elda_protokolle (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    elda_account_id UUID NOT NULL REFERENCES elda_accounts(id) ON DELETE CASCADE,

    -- Reference
    protokollnummer VARCHAR(50) NOT NULL,
    reference_type VARCHAR(50) NOT NULL,  -- mbgm, lohnzettel, meldung
    reference_id UUID NOT NULL,

    -- Details
    meldung_type VARCHAR(50),
    operation VARCHAR(50),        -- create, update, delete, query

    -- Result
    success BOOLEAN,
    error_code VARCHAR(20),
    error_message TEXT,

    -- Request/Response
    request_xml TEXT,
    response_xml TEXT,

    -- Timing
    sent_at TIMESTAMPTZ,
    received_at TIMESTAMPTZ,
    duration_ms INTEGER,

    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_elda_protokolle_account ON elda_protokolle(elda_account_id);
CREATE INDEX IF NOT EXISTS idx_elda_protokolle_nummer ON elda_protokolle(protokollnummer);
CREATE INDEX IF NOT EXISTS idx_elda_protokolle_ref ON elda_protokolle(reference_type, reference_id);
CREATE INDEX IF NOT EXISTS idx_elda_protokolle_date ON elda_protokolle(created_at DESC);

-- ============================================================================
-- ELDA Sync Jobs
-- ============================================================================

CREATE TABLE IF NOT EXISTS elda_sync_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    elda_account_id UUID NOT NULL REFERENCES elda_accounts(id) ON DELETE CASCADE,

    -- Progress
    status VARCHAR(50) DEFAULT 'pending',
    documents_found INTEGER DEFAULT 0,
    documents_new INTEGER DEFAULT 0,

    -- Timing
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    -- Errors
    error_message TEXT,

    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_elda_sync_jobs_account ON elda_sync_jobs(elda_account_id);
CREATE INDEX IF NOT EXISTS idx_elda_sync_jobs_status ON elda_sync_jobs(status);

-- ============================================================================
-- Reference Tables
-- ============================================================================

-- Beitragsgruppen (SV Contribution Groups)
CREATE TABLE IF NOT EXISTS beitragsgruppen (
    code VARCHAR(10) PRIMARY KEY,
    bezeichnung VARCHAR(255) NOT NULL,
    beschreibung TEXT,
    valid_from DATE NOT NULL,
    valid_until DATE,
    is_active BOOLEAN DEFAULT TRUE
);

-- Kollektivverträge (Collective Agreements)
CREATE TABLE IF NOT EXISTS kollektivvertraege (
    code VARCHAR(20) PRIMARY KEY,
    bezeichnung VARCHAR(500) NOT NULL,
    branche VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================================
-- Seed Data: Common Beitragsgruppen
-- ============================================================================

INSERT INTO beitragsgruppen (code, bezeichnung, beschreibung, valid_from) VALUES
('A1', 'Angestellte allgemein', 'Vollversicherte Angestellte', '2020-01-01'),
('A2', 'Angestellte geringfügig', 'Geringfügig beschäftigte Angestellte', '2020-01-01'),
('A3', 'Angestellte fallweise', 'Fallweise beschäftigte Angestellte', '2020-01-01'),
('D1', 'Arbeiter allgemein', 'Vollversicherte Arbeiter', '2020-01-01'),
('D2', 'Arbeiter geringfügig', 'Geringfügig beschäftigte Arbeiter', '2020-01-01'),
('D3', 'Arbeiter fallweise', 'Fallweise beschäftigte Arbeiter', '2020-01-01'),
('N1', 'Freie Dienstnehmer', 'Freie Dienstnehmer vollversichert', '2020-01-01'),
('N2', 'Freie DN geringfügig', 'Freie Dienstnehmer geringfügig', '2020-01-01'),
('L1', 'Lehrlinge', 'Lehrlinge im Lehrverhältnis', '2020-01-01'),
('P1', 'Praktikanten', 'Pflichtpraktikanten', '2020-01-01'),
('B1', 'Bauarbeiter', 'Bauarbeiter (BUAK-pflichtig)', '2020-01-01'),
('S1', 'Saisonarbeiter', 'Saisonbeschäftigte', '2020-01-01')
ON CONFLICT (code) DO NOTHING;

-- ============================================================================
-- Views
-- ============================================================================

-- Pending mBGM
CREATE OR REPLACE VIEW v_pending_mbgm AS
SELECT
    m.id,
    m.year,
    m.month,
    m.status,
    m.total_dienstnehmer,
    m.total_beitragsgrundlage,
    a.name AS account_name,
    ea.dienstgeber_nummer
FROM mbgm m
JOIN elda_accounts ea ON m.elda_account_id = ea.id
JOIN accounts a ON ea.account_id = a.id
WHERE m.status = 'draft'
ORDER BY m.year DESC, m.month DESC;

-- Certificate Expiry Warning
CREATE OR REPLACE VIEW v_expiring_certificates AS
SELECT
    ea.id AS elda_account_id,
    a.id AS account_id,
    a.name AS account_name,
    ea.dienstgeber_nummer,
    ea.certificate_expires_at,
    EXTRACT(DAY FROM ea.certificate_expires_at - NOW()) AS days_until_expiry
FROM elda_accounts ea
JOIN accounts a ON ea.account_id = a.id
WHERE ea.certificate_expires_at IS NOT NULL
  AND ea.certificate_expires_at < NOW() + INTERVAL '30 days'
  AND ea.status = 'active'
ORDER BY ea.certificate_expires_at ASC;

-- L16 Submission Status by Year
CREATE OR REPLACE VIEW v_l16_status AS
SELECT
    l.elda_account_id,
    l.year,
    COUNT(*) AS total_lohnzettel,
    COUNT(*) FILTER (WHERE l.status = 'accepted') AS accepted,
    COUNT(*) FILTER (WHERE l.status = 'rejected') AS rejected,
    COUNT(*) FILTER (WHERE l.status = 'draft') AS pending
FROM lohnzettel l
GROUP BY l.elda_account_id, l.year;

-- Recent ELDA Activity
CREATE OR REPLACE VIEW v_elda_activity AS
SELECT
    p.elda_account_id,
    p.protokollnummer,
    p.reference_type,
    p.meldung_type,
    p.operation,
    p.success,
    p.created_at
FROM elda_protokolle p
ORDER BY p.created_at DESC
LIMIT 100;

-- ============================================================================
-- Comments
-- ============================================================================

COMMENT ON TABLE elda_accounts IS 'ELDA account credentials per organization';
COMMENT ON TABLE mbgm IS 'Monthly contribution reports (mBGM)';
COMMENT ON TABLE mbgm_positionen IS 'Individual employee entries in mBGM';
COMMENT ON TABLE lohnzettel IS 'Annual tax forms (L16) per employee';
COMMENT ON TABLE lohnzettel_batches IS 'Batch submissions of L16 forms';
COMMENT ON TABLE elda_meldungen IS 'Registration/deregistration submissions';
COMMENT ON TABLE elda_documents IS 'Documents from ELDA databox';
COMMENT ON TABLE elda_protokolle IS 'Audit trail of all ELDA communications';
COMMENT ON TABLE beitragsgruppen IS 'Reference: SV contribution group codes';
COMMENT ON TABLE kollektivvertraege IS 'Reference: Collective agreement codes';
