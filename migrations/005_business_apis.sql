-- Migration: 005_business_apis
-- Description: Business APIs - UVA, ZM, UID, E-Rechnung, SEPA, Firmenbuch
-- Spec: 009-business-apis

-- =============================================================================
-- UVA_SUBMISSIONS TABLE - Umsatzsteuervoranmeldung submissions
-- =============================================================================

CREATE TABLE uva_submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    period_year INTEGER NOT NULL CHECK (period_year >= 2000 AND period_year <= 2100),
    period_month INTEGER CHECK (period_month >= 1 AND period_month <= 12),
    period_quarter INTEGER CHECK (period_quarter >= 1 AND period_quarter <= 4),
    period_type VARCHAR(20) NOT NULL CHECK (period_type IN ('monthly', 'quarterly')),

    -- UVA Data fields
    data JSONB NOT NULL DEFAULT '{}', -- All UVA form fields (KZ values)

    -- Validation
    validation_status VARCHAR(50) DEFAULT 'pending' CHECK (validation_status IN ('pending', 'valid', 'invalid')),
    validation_errors JSONB DEFAULT '[]',
    validated_at TIMESTAMPTZ,

    -- Submission
    status VARCHAR(50) DEFAULT 'draft' CHECK (status IN ('draft', 'validated', 'submitted', 'confirmed', 'rejected', 'failed')),
    submitted_at TIMESTAMPTZ,
    submitted_by UUID REFERENCES users(id),

    -- FinanzOnline response
    fo_reference VARCHAR(255), -- FinanzOnline confirmation number
    fo_response JSONB DEFAULT '{}',
    fo_error TEXT,

    -- Batch tracking
    batch_id UUID, -- For batch submissions

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Ensure unique period per account
    UNIQUE(account_id, period_year, period_month) WHERE period_type = 'monthly',
    UNIQUE(account_id, period_year, period_quarter) WHERE period_type = 'quarterly'
);

CREATE INDEX idx_uva_submissions_tenant ON uva_submissions(tenant_id);
CREATE INDEX idx_uva_submissions_account ON uva_submissions(account_id);
CREATE INDEX idx_uva_submissions_period ON uva_submissions(period_year, period_month);
CREATE INDEX idx_uva_submissions_status ON uva_submissions(status);
CREATE INDEX idx_uva_submissions_batch ON uva_submissions(batch_id) WHERE batch_id IS NOT NULL;
CREATE INDEX idx_uva_submissions_created ON uva_submissions(created_at DESC);

-- =============================================================================
-- ZM_SUBMISSIONS TABLE - Zusammenfassende Meldung submissions
-- =============================================================================

CREATE TABLE zm_submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    period_year INTEGER NOT NULL CHECK (period_year >= 2000 AND period_year <= 2100),
    period_month INTEGER NOT NULL CHECK (period_month >= 1 AND period_month <= 12),

    -- ZM Data - Array of EU transactions
    transactions JSONB NOT NULL DEFAULT '[]', -- [{uid, amount, type}]
    is_nullmeldung BOOLEAN DEFAULT FALSE,

    -- Validation
    validation_status VARCHAR(50) DEFAULT 'pending' CHECK (validation_status IN ('pending', 'valid', 'invalid')),
    validation_errors JSONB DEFAULT '[]',
    validated_at TIMESTAMPTZ,

    -- Submission
    status VARCHAR(50) DEFAULT 'draft' CHECK (status IN ('draft', 'validated', 'submitted', 'confirmed', 'rejected', 'failed')),
    submitted_at TIMESTAMPTZ,
    submitted_by UUID REFERENCES users(id),

    -- FinanzOnline response
    fo_reference VARCHAR(255),
    fo_response JSONB DEFAULT '{}',
    fo_error TEXT,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(account_id, period_year, period_month)
);

CREATE INDEX idx_zm_submissions_tenant ON zm_submissions(tenant_id);
CREATE INDEX idx_zm_submissions_account ON zm_submissions(account_id);
CREATE INDEX idx_zm_submissions_period ON zm_submissions(period_year, period_month);
CREATE INDEX idx_zm_submissions_status ON zm_submissions(status);

-- =============================================================================
-- UID_VALIDATIONS TABLE - UID validation history
-- =============================================================================

CREATE TABLE uid_validations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Validated UID
    uid_number VARCHAR(20) NOT NULL, -- e.g., ATU12345678
    own_uid VARCHAR(20), -- Own UID used for Level 2 validation

    -- Validation level
    level INTEGER NOT NULL CHECK (level IN (1, 2)), -- Level 1: valid/invalid, Level 2: with company data

    -- Result
    is_valid BOOLEAN NOT NULL,
    company_name VARCHAR(500),
    company_address TEXT,

    -- Response details
    response_data JSONB DEFAULT '{}',

    -- Cache info
    cached BOOLEAN DEFAULT FALSE,
    cache_expires_at TIMESTAMPTZ,

    -- Timestamps
    validated_at TIMESTAMPTZ DEFAULT NOW(),
    validated_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_uid_validations_tenant ON uid_validations(tenant_id);
CREATE INDEX idx_uid_validations_uid ON uid_validations(uid_number);
CREATE INDEX idx_uid_validations_validated_at ON uid_validations(validated_at DESC);
-- Cache lookup index
CREATE INDEX idx_uid_validations_cache ON uid_validations(uid_number, level, validated_at DESC)
    WHERE cached = TRUE AND cache_expires_at > NOW();

-- =============================================================================
-- INVOICES TABLE - Electronic invoices (XRechnung/ZUGFeRD)
-- =============================================================================

CREATE TABLE invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Invoice identification
    invoice_number VARCHAR(100) NOT NULL,
    invoice_date DATE NOT NULL,
    due_date DATE,

    -- Supplier (Seller)
    supplier_name VARCHAR(500) NOT NULL,
    supplier_street VARCHAR(500),
    supplier_city VARCHAR(200),
    supplier_postal_code VARCHAR(20),
    supplier_country VARCHAR(2) DEFAULT 'AT',
    supplier_uid VARCHAR(20),
    supplier_iban VARCHAR(50),
    supplier_bic VARCHAR(20),
    supplier_contact_name VARCHAR(200),
    supplier_contact_email VARCHAR(500),
    supplier_contact_phone VARCHAR(100),

    -- Customer (Buyer)
    customer_name VARCHAR(500) NOT NULL,
    customer_street VARCHAR(500),
    customer_city VARCHAR(200),
    customer_postal_code VARCHAR(20),
    customer_country VARCHAR(2) DEFAULT 'AT',
    customer_uid VARCHAR(20),
    customer_reference VARCHAR(200), -- Leitweg-ID or customer reference

    -- Amounts (in cents for precision)
    net_amount_cents BIGINT NOT NULL DEFAULT 0,
    tax_amount_cents BIGINT NOT NULL DEFAULT 0,
    gross_amount_cents BIGINT NOT NULL DEFAULT 0,
    currency VARCHAR(3) DEFAULT 'EUR',

    -- Payment
    payment_terms TEXT,
    payment_reference VARCHAR(200),

    -- Notes
    notes TEXT,

    -- Format and generation
    format VARCHAR(20) DEFAULT 'xrechnung' CHECK (format IN ('xrechnung', 'zugferd', 'both')),
    xml_content TEXT, -- Generated XRechnung XML
    pdf_content BYTEA, -- Generated ZUGFeRD PDF

    -- Validation
    validation_status VARCHAR(50) DEFAULT 'pending' CHECK (validation_status IN ('pending', 'valid', 'invalid')),
    validation_errors JSONB DEFAULT '[]',
    validated_at TIMESTAMPTZ,

    -- Status
    status VARCHAR(50) DEFAULT 'draft' CHECK (status IN ('draft', 'validated', 'finalized', 'sent', 'cancelled')),

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID REFERENCES users(id),

    UNIQUE(tenant_id, invoice_number)
);

CREATE INDEX idx_invoices_tenant ON invoices(tenant_id);
CREATE INDEX idx_invoices_number ON invoices(invoice_number);
CREATE INDEX idx_invoices_date ON invoices(invoice_date DESC);
CREATE INDEX idx_invoices_customer ON invoices(customer_name);
CREATE INDEX idx_invoices_status ON invoices(status);

-- =============================================================================
-- INVOICE_ITEMS TABLE - Invoice line items
-- =============================================================================

CREATE TABLE invoice_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    position INTEGER NOT NULL,

    -- Item details
    description VARCHAR(1000) NOT NULL,
    quantity DECIMAL(12, 4) NOT NULL DEFAULT 1,
    unit VARCHAR(50) DEFAULT 'EA', -- UN/ECE Recommendation 20 units

    -- Prices (in cents)
    unit_price_cents BIGINT NOT NULL,
    net_amount_cents BIGINT NOT NULL,
    tax_rate DECIMAL(5, 2) NOT NULL DEFAULT 20.00, -- e.g., 20.00 for 20%
    tax_amount_cents BIGINT NOT NULL,
    gross_amount_cents BIGINT NOT NULL,

    -- Optional
    item_number VARCHAR(100), -- Article/product number
    buyer_reference VARCHAR(200),

    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_invoice_items_invoice ON invoice_items(invoice_id);
CREATE INDEX idx_invoice_items_position ON invoice_items(invoice_id, position);

-- =============================================================================
-- PAYMENT_BATCHES TABLE - SEPA payment file batches
-- =============================================================================

CREATE TABLE payment_batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Batch identification
    batch_reference VARCHAR(100) NOT NULL,
    batch_type VARCHAR(20) NOT NULL CHECK (batch_type IN ('credit_transfer', 'direct_debit')), -- pain.001 or pain.008

    -- Initiator (debtor for credit transfer, creditor for direct debit)
    initiator_name VARCHAR(200) NOT NULL,
    initiator_iban VARCHAR(50) NOT NULL,
    initiator_bic VARCHAR(20),

    -- For direct debit only
    creditor_id VARCHAR(50), -- Gläubiger-ID

    -- Totals
    total_amount_cents BIGINT NOT NULL DEFAULT 0,
    total_items INTEGER NOT NULL DEFAULT 0,
    currency VARCHAR(3) DEFAULT 'EUR',

    -- Execution
    requested_execution_date DATE NOT NULL,

    -- Generated file
    file_content TEXT, -- Generated pain.001 or pain.008 XML
    file_name VARCHAR(255),

    -- Status
    status VARCHAR(50) DEFAULT 'draft' CHECK (status IN ('draft', 'finalized', 'downloaded', 'submitted')),
    finalized_at TIMESTAMPTZ,
    downloaded_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID REFERENCES users(id)
);

CREATE INDEX idx_payment_batches_tenant ON payment_batches(tenant_id);
CREATE INDEX idx_payment_batches_type ON payment_batches(batch_type);
CREATE INDEX idx_payment_batches_date ON payment_batches(requested_execution_date);
CREATE INDEX idx_payment_batches_status ON payment_batches(status);

-- =============================================================================
-- PAYMENT_ITEMS TABLE - Individual payments in a batch
-- =============================================================================

CREATE TABLE payment_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id UUID NOT NULL REFERENCES payment_batches(id) ON DELETE CASCADE,
    position INTEGER NOT NULL,

    -- Counterparty (creditor for credit transfer, debtor for direct debit)
    counterparty_name VARCHAR(200) NOT NULL,
    counterparty_iban VARCHAR(50) NOT NULL,
    counterparty_bic VARCHAR(20),

    -- Amount
    amount_cents BIGINT NOT NULL,
    currency VARCHAR(3) DEFAULT 'EUR',

    -- Payment details
    reference VARCHAR(140), -- Verwendungszweck (SEPA max 140 chars)
    end_to_end_id VARCHAR(35), -- Unique payment ID

    -- Direct debit specific
    mandate_reference VARCHAR(35),
    mandate_date DATE,
    sequence_type VARCHAR(10) CHECK (sequence_type IN ('FRST', 'RCUR', 'FNAL', 'OOFF')),

    -- Validation
    is_valid BOOLEAN DEFAULT TRUE,
    validation_error TEXT,

    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_payment_items_batch ON payment_items(batch_id);
CREATE INDEX idx_payment_items_position ON payment_items(batch_id, position);

-- =============================================================================
-- BANK_STATEMENTS TABLE - Imported camt.053 statements
-- =============================================================================

CREATE TABLE bank_statements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Statement identification
    statement_id VARCHAR(100), -- From camt.053
    sequence_number INTEGER,

    -- Account
    iban VARCHAR(50) NOT NULL,
    bic VARCHAR(20),
    account_name VARCHAR(200),

    -- Period
    from_date DATE,
    to_date DATE,

    -- Balances (in cents)
    opening_balance_cents BIGINT,
    closing_balance_cents BIGINT,
    currency VARCHAR(3) DEFAULT 'EUR',

    -- Original file
    file_name VARCHAR(255),
    file_content TEXT, -- Original camt.053 XML

    -- Processing
    status VARCHAR(50) DEFAULT 'imported' CHECK (status IN ('imported', 'processing', 'processed', 'error')),
    processed_at TIMESTAMPTZ,
    processing_error TEXT,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    imported_by UUID REFERENCES users(id)
);

CREATE INDEX idx_bank_statements_tenant ON bank_statements(tenant_id);
CREATE INDEX idx_bank_statements_iban ON bank_statements(iban);
CREATE INDEX idx_bank_statements_date ON bank_statements(to_date DESC);

-- =============================================================================
-- TRANSACTIONS TABLE - Parsed transactions from bank statements
-- =============================================================================

CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    statement_id UUID NOT NULL REFERENCES bank_statements(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Transaction identification
    transaction_id VARCHAR(100), -- From camt.053
    entry_reference VARCHAR(100),

    -- Amount
    amount_cents BIGINT NOT NULL,
    currency VARCHAR(3) DEFAULT 'EUR',
    credit_debit VARCHAR(10) NOT NULL CHECK (credit_debit IN ('credit', 'debit')),

    -- Dates
    booking_date DATE NOT NULL,
    value_date DATE,

    -- Counterparty
    counterparty_name VARCHAR(200),
    counterparty_iban VARCHAR(50),
    counterparty_bic VARCHAR(20),

    -- Details
    reference VARCHAR(500),
    additional_info TEXT,
    bank_reference VARCHAR(100),

    -- Matching
    matched_invoice_id UUID REFERENCES invoices(id),
    matched_payment_id UUID REFERENCES payment_items(id),
    match_status VARCHAR(50) DEFAULT 'unmatched' CHECK (match_status IN ('unmatched', 'suggested', 'matched', 'ignored')),
    matched_at TIMESTAMPTZ,
    matched_by UUID REFERENCES users(id),

    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_transactions_statement ON transactions(statement_id);
CREATE INDEX idx_transactions_tenant ON transactions(tenant_id);
CREATE INDEX idx_transactions_booking ON transactions(booking_date DESC);
CREATE INDEX idx_transactions_match ON transactions(match_status);
CREATE INDEX idx_transactions_counterparty ON transactions(counterparty_iban);

-- =============================================================================
-- FIRMENBUCH_CACHE TABLE - Cached Firmenbuch extracts
-- =============================================================================

CREATE TABLE firmenbuch_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE, -- NULL for shared cache

    -- Company identification
    company_number VARCHAR(50) NOT NULL, -- FN number e.g., "12345d"

    -- Cached data
    company_name VARCHAR(500),
    legal_form VARCHAR(100),
    address TEXT,
    extract_data JSONB NOT NULL DEFAULT '{}', -- Full FB extract

    -- Cache metadata
    fetched_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ DEFAULT NOW() + INTERVAL '24 hours',

    -- Query info
    queried_by UUID REFERENCES users(id),
    query_count INTEGER DEFAULT 1
);

CREATE INDEX idx_firmenbuch_cache_fn ON firmenbuch_cache(company_number);
CREATE INDEX idx_firmenbuch_cache_expires ON firmenbuch_cache(expires_at);
CREATE INDEX idx_firmenbuch_cache_tenant ON firmenbuch_cache(tenant_id) WHERE tenant_id IS NOT NULL;

-- =============================================================================
-- FIRMENBUCH_HISTORY TABLE - History of FB queries per company
-- =============================================================================

CREATE TABLE firmenbuch_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    company_number VARCHAR(50) NOT NULL,
    company_name VARCHAR(500),
    extract_data JSONB,
    queried_at TIMESTAMPTZ DEFAULT NOW(),
    queried_by UUID REFERENCES users(id)
);

CREATE INDEX idx_firmenbuch_history_tenant ON firmenbuch_history(tenant_id);
CREATE INDEX idx_firmenbuch_history_fn ON firmenbuch_history(company_number);
CREATE INDEX idx_firmenbuch_history_queried ON firmenbuch_history(queried_at DESC);

-- =============================================================================
-- MASTER_DATA_SUPPLIERS TABLE - Reusable supplier data
-- =============================================================================

CREATE TABLE master_data_suppliers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Identification
    code VARCHAR(50), -- Internal code
    name VARCHAR(500) NOT NULL,

    -- Address
    street VARCHAR(500),
    city VARCHAR(200),
    postal_code VARCHAR(20),
    country VARCHAR(2) DEFAULT 'AT',

    -- Tax
    uid VARCHAR(20),

    -- Bank
    iban VARCHAR(50),
    bic VARCHAR(20),

    -- Contact
    contact_name VARCHAR(200),
    contact_email VARCHAR(500),
    contact_phone VARCHAR(100),

    -- Status
    is_active BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(tenant_id, code) WHERE code IS NOT NULL
);

CREATE INDEX idx_master_suppliers_tenant ON master_data_suppliers(tenant_id);
CREATE INDEX idx_master_suppliers_name ON master_data_suppliers(name);

-- =============================================================================
-- MASTER_DATA_CUSTOMERS TABLE - Reusable customer data
-- =============================================================================

CREATE TABLE master_data_customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Identification
    code VARCHAR(50), -- Internal code
    name VARCHAR(500) NOT NULL,

    -- Address
    street VARCHAR(500),
    city VARCHAR(200),
    postal_code VARCHAR(20),
    country VARCHAR(2) DEFAULT 'AT',

    -- Tax
    uid VARCHAR(20),

    -- Reference for e-invoicing (Leitweg-ID)
    leitweg_id VARCHAR(200),

    -- Contact
    contact_name VARCHAR(200),
    contact_email VARCHAR(500),
    contact_phone VARCHAR(100),

    -- Status
    is_active BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(tenant_id, code) WHERE code IS NOT NULL
);

CREATE INDEX idx_master_customers_tenant ON master_data_customers(tenant_id);
CREATE INDEX idx_master_customers_name ON master_data_customers(name);

-- =============================================================================
-- PAYMENT_TEMPLATES TABLE - Reusable payment templates
-- =============================================================================

CREATE TABLE payment_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Template identification
    name VARCHAR(200) NOT NULL,
    description TEXT,

    -- Recipient
    recipient_name VARCHAR(200) NOT NULL,
    recipient_iban VARCHAR(50) NOT NULL,
    recipient_bic VARCHAR(20),

    -- Default amount (optional)
    default_amount_cents BIGINT,
    currency VARCHAR(3) DEFAULT 'EUR',

    -- Default reference
    default_reference VARCHAR(140),

    -- For direct debit
    mandate_reference VARCHAR(35),
    mandate_date DATE,
    creditor_id VARCHAR(50),

    -- Usage tracking
    use_count INTEGER DEFAULT 0,
    last_used_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_payment_templates_tenant ON payment_templates(tenant_id);
CREATE INDEX idx_payment_templates_name ON payment_templates(name);

-- =============================================================================
-- UVA_BATCHES TABLE - Batch UVA submissions
-- =============================================================================

CREATE TABLE uva_batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Batch info
    period_year INTEGER NOT NULL,
    period_month INTEGER,
    period_quarter INTEGER,
    period_type VARCHAR(20) NOT NULL CHECK (period_type IN ('monthly', 'quarterly')),

    -- Counts
    total_accounts INTEGER DEFAULT 0,
    submitted_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    failed_count INTEGER DEFAULT 0,

    -- Status
    status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    -- Job reference (for background processing)
    job_id UUID REFERENCES jobs(id),

    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID REFERENCES users(id)
);

CREATE INDEX idx_uva_batches_tenant ON uva_batches(tenant_id);
CREATE INDEX idx_uva_batches_status ON uva_batches(status);
CREATE INDEX idx_uva_batches_period ON uva_batches(period_year, period_month);

-- =============================================================================
-- UPDATE TRIGGERS
-- =============================================================================

CREATE TRIGGER update_uva_submissions_updated_at
    BEFORE UPDATE ON uva_submissions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_zm_submissions_updated_at
    BEFORE UPDATE ON zm_submissions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_invoices_updated_at
    BEFORE UPDATE ON invoices
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_payment_batches_updated_at
    BEFORE UPDATE ON payment_batches
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_master_suppliers_updated_at
    BEFORE UPDATE ON master_data_suppliers
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_master_customers_updated_at
    BEFORE UPDATE ON master_data_customers
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_payment_templates_updated_at
    BEFORE UPDATE ON payment_templates
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- =============================================================================
-- COMMENTS
-- =============================================================================

COMMENT ON TABLE uva_submissions IS 'Umsatzsteuervoranmeldung submissions to FinanzOnline';
COMMENT ON COLUMN uva_submissions.data IS 'All UVA form fields stored as JSONB (KZ values)';
COMMENT ON COLUMN uva_submissions.fo_reference IS 'FinanzOnline confirmation number after successful submission';

COMMENT ON TABLE zm_submissions IS 'Zusammenfassende Meldung (EU transaction report) submissions';
COMMENT ON COLUMN zm_submissions.transactions IS 'Array of EU transactions: [{uid, amount, type}]';
COMMENT ON COLUMN zm_submissions.is_nullmeldung IS 'True if this is a null declaration (no transactions)';

COMMENT ON TABLE uid_validations IS 'History of UID number validations for compliance';
COMMENT ON COLUMN uid_validations.level IS 'Validation level: 1=valid/invalid only, 2=with company data';

COMMENT ON TABLE invoices IS 'Electronic invoices in XRechnung or ZUGFeRD format';
COMMENT ON COLUMN invoices.customer_reference IS 'Leitweg-ID for B2G invoices or customer reference';

COMMENT ON TABLE payment_batches IS 'SEPA payment file batches (pain.001/pain.008)';
COMMENT ON COLUMN payment_batches.creditor_id IS 'Gläubiger-Identifikationsnummer for direct debits';

COMMENT ON TABLE payment_items IS 'Individual payments within a SEPA batch';
COMMENT ON COLUMN payment_items.sequence_type IS 'FRST=First, RCUR=Recurring, FNAL=Final, OOFF=One-off';

COMMENT ON TABLE bank_statements IS 'Imported bank statements in camt.053 format';
COMMENT ON TABLE transactions IS 'Parsed transactions from bank statements';
COMMENT ON COLUMN transactions.match_status IS 'Transaction matching status with invoices/payments';

COMMENT ON TABLE firmenbuch_cache IS 'Cached Firmenbuch extracts (24h TTL)';
COMMENT ON TABLE firmenbuch_history IS 'Historical Firmenbuch queries for audit trail';

COMMENT ON TABLE master_data_suppliers IS 'Reusable supplier master data for invoicing';
COMMENT ON TABLE master_data_customers IS 'Reusable customer master data for invoicing';
COMMENT ON TABLE payment_templates IS 'Reusable payment templates for frequent recipients';

COMMENT ON TABLE uva_batches IS 'Batch UVA submission tracking for multiple accounts';
