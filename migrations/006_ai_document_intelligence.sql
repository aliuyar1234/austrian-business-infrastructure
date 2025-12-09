-- Migration: 006_ai_document_intelligence
-- AI Document Intelligence - OCR, Classification, Extraction, Suggestions

-- Document analyses (main analysis record per document)
CREATE TABLE IF NOT EXISTS document_analyses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Analysis status
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'needs_review')),

    -- Classification results
    document_type VARCHAR(50), -- bescheid, ersuchen, info, rechnung, sonstige
    document_subtype VARCHAR(100), -- e.g., "ergaenzungsersuchen", "steuerbescheid", "mahnbescheid"
    classification_confidence DECIMAL(3,2), -- 0.00 to 1.00
    priority VARCHAR(20), -- critical, high, medium, low

    -- OCR results
    is_scanned BOOLEAN DEFAULT FALSE,
    ocr_provider VARCHAR(20), -- hunyuan, tesseract, none
    ocr_confidence DECIMAL(3,2),
    extracted_text TEXT,

    -- Summary
    summary TEXT,
    key_points JSONB, -- array of key points

    -- Raw AI responses for debugging
    raw_classification_response JSONB,
    raw_summary_response JSONB,

    -- Processing metadata
    processing_time_ms INTEGER,
    token_count INTEGER,
    cost_cents INTEGER, -- cost in euro cents
    error_message TEXT,

    -- Manual corrections tracking
    manually_corrected BOOLEAN DEFAULT FALSE,
    corrected_by UUID REFERENCES users(id),
    corrected_at TIMESTAMP WITH TIME ZONE,
    correction_notes TEXT,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT unique_document_analysis UNIQUE (document_id)
);

CREATE INDEX idx_document_analyses_document ON document_analyses(document_id);
CREATE INDEX idx_document_analyses_tenant ON document_analyses(tenant_id);
CREATE INDEX idx_document_analyses_status ON document_analyses(status);
CREATE INDEX idx_document_analyses_type ON document_analyses(document_type);
CREATE INDEX idx_document_analyses_priority ON document_analyses(priority);

-- Extracted deadlines
CREATE TABLE IF NOT EXISTS extracted_deadlines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    analysis_id UUID NOT NULL REFERENCES document_analyses(id) ON DELETE CASCADE,
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Deadline info
    deadline_type VARCHAR(50) NOT NULL, -- response, payment, appeal, submission, other
    deadline_date DATE NOT NULL,
    calculated_from DATE, -- original date if calculated
    calculation_rule VARCHAR(100), -- e.g., "4 Wochen ab Zustellung"

    -- Source in document
    source_text TEXT, -- original text mentioning deadline
    page_number INTEGER,

    -- Confidence and status
    confidence DECIMAL(3,2) NOT NULL,
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'completed', 'dismissed', 'overdue')),

    -- Linked action item
    action_item_id UUID,

    -- Manual correction
    manually_corrected BOOLEAN DEFAULT FALSE,
    original_date DATE, -- before correction
    corrected_by UUID REFERENCES users(id),

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_extracted_deadlines_analysis ON extracted_deadlines(analysis_id);
CREATE INDEX idx_extracted_deadlines_document ON extracted_deadlines(document_id);
CREATE INDEX idx_extracted_deadlines_tenant ON extracted_deadlines(tenant_id);
CREATE INDEX idx_extracted_deadlines_date ON extracted_deadlines(deadline_date);
CREATE INDEX idx_extracted_deadlines_status ON extracted_deadlines(status);

-- Extracted amounts
CREATE TABLE IF NOT EXISTS extracted_amounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    analysis_id UUID NOT NULL REFERENCES document_analyses(id) ON DELETE CASCADE,
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Amount info
    amount_type VARCHAR(50) NOT NULL, -- tax_due, refund, penalty, fee, total, other
    amount DECIMAL(15,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'EUR',
    is_negative BOOLEAN DEFAULT FALSE, -- true for Gutschrift

    -- Source in document
    source_text TEXT,
    label TEXT, -- e.g., "Umsatzsteuer", "Verspätungszuschlag"
    page_number INTEGER,

    -- Confidence
    confidence DECIMAL(3,2) NOT NULL,

    -- Manual correction
    manually_corrected BOOLEAN DEFAULT FALSE,
    original_amount DECIMAL(15,2),
    corrected_by UUID REFERENCES users(id),

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_extracted_amounts_analysis ON extracted_amounts(analysis_id);
CREATE INDEX idx_extracted_amounts_document ON extracted_amounts(document_id);
CREATE INDEX idx_extracted_amounts_tenant ON extracted_amounts(tenant_id);
CREATE INDEX idx_extracted_amounts_type ON extracted_amounts(amount_type);

-- Action items (from document analysis)
CREATE TABLE IF NOT EXISTS action_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    document_id UUID REFERENCES documents(id) ON DELETE SET NULL,
    analysis_id UUID REFERENCES document_analyses(id) ON DELETE SET NULL,
    deadline_id UUID REFERENCES extracted_deadlines(id) ON DELETE SET NULL,

    -- Action item details
    title VARCHAR(255) NOT NULL,
    description TEXT,
    action_type VARCHAR(50) NOT NULL, -- respond, pay, appeal, review, submit, other

    -- Priority and status
    priority VARCHAR(20) DEFAULT 'medium' CHECK (priority IN ('critical', 'high', 'medium', 'low')),
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'completed', 'dismissed')),

    -- Timing
    due_date DATE,
    reminder_date DATE,
    completed_at TIMESTAMP WITH TIME ZONE,

    -- Assignment
    assigned_to UUID REFERENCES users(id),
    created_by UUID REFERENCES users(id),

    -- Source tracking
    source VARCHAR(20) DEFAULT 'ai' CHECK (source IN ('ai', 'manual', 'system')),
    ai_confidence DECIMAL(3,2),

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_action_items_tenant ON action_items(tenant_id);
CREATE INDEX idx_action_items_document ON action_items(document_id);
CREATE INDEX idx_action_items_status ON action_items(status);
CREATE INDEX idx_action_items_due_date ON action_items(due_date);
CREATE INDEX idx_action_items_assigned ON action_items(assigned_to);
CREATE INDEX idx_action_items_priority ON action_items(priority);

-- Response suggestions for Ergänzungsersuchen
CREATE TABLE IF NOT EXISTS response_suggestions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    analysis_id UUID NOT NULL REFERENCES document_analyses(id) ON DELETE CASCADE,
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Suggestion content
    suggestion_text TEXT NOT NULL,
    key_points JSONB, -- array of points to address
    required_documents JSONB, -- array of documents to attach

    -- Metadata
    confidence DECIMAL(3,2),
    disclaimer TEXT DEFAULT 'Diese Antwort wurde automatisch generiert und ersetzt keine rechtliche Beratung.',

    -- Usage tracking
    was_used BOOLEAN DEFAULT FALSE,
    used_at TIMESTAMP WITH TIME ZONE,
    user_rating INTEGER CHECK (user_rating BETWEEN 1 AND 5),

    -- Raw response
    raw_response JSONB,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_response_suggestions_analysis ON response_suggestions(analysis_id);
CREATE INDEX idx_response_suggestions_document ON response_suggestions(document_id);
CREATE INDEX idx_response_suggestions_tenant ON response_suggestions(tenant_id);

-- Response templates (saved by users)
CREATE TABLE IF NOT EXISTS response_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    created_by UUID NOT NULL REFERENCES users(id),

    -- Template info
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(50), -- ergaenzungsersuchen, einspruch, allgemein

    -- Content
    template_text TEXT NOT NULL,
    placeholders JSONB, -- array of placeholder names

    -- Usage
    use_count INTEGER DEFAULT 0,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_response_templates_tenant ON response_templates(tenant_id);
CREATE INDEX idx_response_templates_category ON response_templates(category);

-- Analysis prompts (stored in DB for easy iteration)
CREATE TABLE IF NOT EXISTS analysis_prompts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Prompt identification
    prompt_type VARCHAR(50) NOT NULL UNIQUE, -- classification, deadline, summary, amount, suggestion
    version INTEGER DEFAULT 1,
    is_active BOOLEAN DEFAULT TRUE,

    -- Prompt content
    system_prompt TEXT NOT NULL,
    user_prompt_template TEXT NOT NULL, -- with {placeholders}

    -- Model settings
    model VARCHAR(50) DEFAULT 'claude-sonnet-4-20250514',
    max_tokens INTEGER DEFAULT 2048,
    temperature DECIMAL(2,1) DEFAULT 0.3,

    -- Response schema
    response_schema JSONB, -- expected JSON structure

    -- Metadata
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_analysis_prompts_type ON analysis_prompts(prompt_type);
CREATE INDEX idx_analysis_prompts_active ON analysis_prompts(is_active);

-- AI usage logs (for cost tracking)
CREATE TABLE IF NOT EXISTS ai_usage_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Request info
    prompt_type VARCHAR(50) NOT NULL,
    model VARCHAR(50) NOT NULL,
    document_id UUID REFERENCES documents(id) ON DELETE SET NULL,

    -- Token usage
    input_tokens INTEGER NOT NULL,
    output_tokens INTEGER NOT NULL,
    total_tokens INTEGER NOT NULL,

    -- Cost (in cents)
    cost_cents INTEGER NOT NULL,

    -- Timing
    latency_ms INTEGER,

    -- Status
    success BOOLEAN NOT NULL,
    error_message TEXT,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_ai_usage_logs_tenant ON ai_usage_logs(tenant_id);
CREATE INDEX idx_ai_usage_logs_created ON ai_usage_logs(created_at);
CREATE INDEX idx_ai_usage_logs_type ON ai_usage_logs(prompt_type);

-- Seed default prompts
INSERT INTO analysis_prompts (prompt_type, system_prompt, user_prompt_template, response_schema, description) VALUES
(
    'classification',
    'Du bist ein Experte für österreichische Steuerdokumente und Behördenbriefe. Klassifiziere das folgende Dokument präzise. Antworte ausschließlich im JSON-Format.',
    'Analysiere dieses Dokument und klassifiziere es:

{document_text}

Antworte im folgenden JSON-Format:
{
  "document_type": "bescheid|ersuchen|info|rechnung|mahnung|sonstige",
  "document_subtype": "ergaenzungsersuchen|steuerbescheid|mahnbescheid|...",
  "priority": "critical|high|medium|low",
  "confidence": 0.0-1.0,
  "reasoning": "Kurze Begründung"
}',
    '{"type": "object", "properties": {"document_type": {"type": "string"}, "document_subtype": {"type": "string"}, "priority": {"type": "string"}, "confidence": {"type": "number"}, "reasoning": {"type": "string"}}}',
    'Document classification prompt'
),
(
    'deadline',
    'Du bist ein Experte für österreichische Steuerdokumente. Extrahiere alle Fristen aus dem Dokument. Berechne das konkrete Datum wenn nötig. Antworte ausschließlich im JSON-Format.',
    'Extrahiere alle Fristen aus diesem Dokument:

{document_text}

Heute ist der {current_date}.
Zustelldatum (falls bekannt): {delivery_date}

Antworte im folgenden JSON-Format:
{
  "deadlines": [
    {
      "deadline_type": "response|payment|appeal|submission|other",
      "deadline_date": "YYYY-MM-DD",
      "calculation_rule": "z.B. 4 Wochen ab Zustellung",
      "source_text": "Originaltext aus Dokument",
      "confidence": 0.0-1.0
    }
  ]
}',
    '{"type": "object", "properties": {"deadlines": {"type": "array", "items": {"type": "object"}}}}',
    'Deadline extraction prompt'
),
(
    'summary',
    'Du bist ein Experte für österreichische Steuerdokumente. Fasse das Dokument in einfacher Sprache zusammen. Vermeide Amtsdeutsch. Antworte ausschließlich im JSON-Format.',
    'Fasse dieses Dokument zusammen:

{document_text}

Antworte im folgenden JSON-Format:
{
  "summary": "Zusammenfassung in 2-4 Sätzen, einfache Sprache",
  "key_points": [
    "Wichtiger Punkt 1",
    "Wichtiger Punkt 2"
  ],
  "action_required": true|false,
  "tone": "neutral|dringend|positiv|negativ"
}',
    '{"type": "object", "properties": {"summary": {"type": "string"}, "key_points": {"type": "array"}, "action_required": {"type": "boolean"}, "tone": {"type": "string"}}}',
    'Document summarization prompt'
),
(
    'amount',
    'Du bist ein Experte für österreichische Steuerdokumente. Extrahiere alle Geldbeträge aus dem Dokument. Antworte ausschließlich im JSON-Format.',
    'Extrahiere alle Geldbeträge aus diesem Dokument:

{document_text}

Antworte im folgenden JSON-Format:
{
  "amounts": [
    {
      "amount_type": "tax_due|refund|penalty|fee|total|other",
      "amount": 1234.56,
      "currency": "EUR",
      "is_negative": false,
      "label": "z.B. Umsatzsteuer",
      "source_text": "Originaltext",
      "confidence": 0.0-1.0
    }
  ]
}',
    '{"type": "object", "properties": {"amounts": {"type": "array", "items": {"type": "object"}}}}',
    'Amount extraction prompt'
),
(
    'suggestion',
    'Du bist ein Steuerberater-Assistent. Erstelle einen Antwortvorschlag für das folgende Ergänzungsersuchen. Der Vorschlag sollte professionell und vollständig sein. Antworte ausschließlich im JSON-Format.',
    'Erstelle einen Antwortvorschlag für dieses Ergänzungsersuchen:

{document_text}

Kontext zum Mandanten: {client_context}

Antworte im folgenden JSON-Format:
{
  "suggestion_text": "Vollständiger Antworttext",
  "key_points": ["Punkt 1 der adressiert wird", "Punkt 2"],
  "required_documents": ["Dokument 1", "Dokument 2"],
  "confidence": 0.0-1.0,
  "warnings": ["Eventuelle Hinweise"]
}',
    '{"type": "object", "properties": {"suggestion_text": {"type": "string"}, "key_points": {"type": "array"}, "required_documents": {"type": "array"}, "confidence": {"type": "number"}, "warnings": {"type": "array"}}}',
    'Response suggestion prompt for Ergänzungsersuchen'
)
ON CONFLICT (prompt_type) DO NOTHING;

-- Add trigger for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_document_analyses_updated_at BEFORE UPDATE ON document_analyses FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_extracted_deadlines_updated_at BEFORE UPDATE ON extracted_deadlines FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_action_items_updated_at BEFORE UPDATE ON action_items FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_response_templates_updated_at BEFORE UPDATE ON response_templates FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_analysis_prompts_updated_at BEFORE UPDATE ON analysis_prompts FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
