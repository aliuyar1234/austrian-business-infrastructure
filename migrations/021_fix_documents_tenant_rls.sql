-- Migration: 021_fix_documents_tenant_rls
-- Description: Fix documents table missing tenant_id for RLS policies
-- Issue: Migration 020 enabled RLS on documents referencing tenant_id, but column didn't exist

-- =============================================================================
-- Step 1: Add tenant_id column to documents
-- =============================================================================

ALTER TABLE documents ADD COLUMN IF NOT EXISTS tenant_id UUID;

-- =============================================================================
-- Step 2: Backfill tenant_id from accounts
-- =============================================================================

UPDATE documents d
SET tenant_id = a.tenant_id
FROM accounts a
WHERE d.account_id = a.id
  AND d.tenant_id IS NULL;

-- =============================================================================
-- Step 3: Add NOT NULL constraint and foreign key
-- =============================================================================

-- Make tenant_id NOT NULL (only if all rows have been backfilled)
DO $$
BEGIN
    -- Check if any documents have NULL tenant_id (would indicate orphaned records)
    IF EXISTS (SELECT 1 FROM documents WHERE tenant_id IS NULL) THEN
        RAISE NOTICE 'Warning: Some documents have NULL tenant_id (orphaned records). Deleting them.';
        DELETE FROM documents WHERE tenant_id IS NULL;
    END IF;
END $$;

ALTER TABLE documents ALTER COLUMN tenant_id SET NOT NULL;

-- Add foreign key constraint
ALTER TABLE documents
    ADD CONSTRAINT fk_documents_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- =============================================================================
-- Step 4: Add index for tenant queries
-- =============================================================================

CREATE INDEX IF NOT EXISTS idx_documents_tenant ON documents(tenant_id);

-- Composite index for common query pattern: tenant + account + active documents
CREATE INDEX IF NOT EXISTS idx_documents_tenant_account_active
    ON documents(tenant_id, account_id, received_at DESC)
    WHERE archived_at IS NULL;

-- =============================================================================
-- Step 5: Fix RLS policy (recreate with correct column reference)
-- =============================================================================

-- Drop the broken policy from migration 020
DROP POLICY IF EXISTS tenant_isolation_documents ON documents;

-- Ensure RLS is enabled
ALTER TABLE documents ENABLE ROW LEVEL SECURITY;

-- Create proper tenant isolation policy
CREATE POLICY tenant_isolation_documents ON documents
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- =============================================================================
-- Step 6: Add trigger to auto-populate tenant_id on insert
-- =============================================================================

CREATE OR REPLACE FUNCTION set_document_tenant_id()
RETURNS TRIGGER AS $$
BEGIN
    -- If tenant_id not provided, derive from account
    IF NEW.tenant_id IS NULL AND NEW.account_id IS NOT NULL THEN
        SELECT tenant_id INTO NEW.tenant_id
        FROM accounts
        WHERE id = NEW.account_id;
    END IF;

    -- Validate tenant_id is set
    IF NEW.tenant_id IS NULL THEN
        RAISE EXCEPTION 'tenant_id cannot be NULL for documents';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS set_document_tenant_id_trigger ON documents;
CREATE TRIGGER set_document_tenant_id_trigger
    BEFORE INSERT ON documents
    FOR EACH ROW
    EXECUTE FUNCTION set_document_tenant_id();

-- =============================================================================
-- Comments
-- =============================================================================

COMMENT ON COLUMN documents.tenant_id IS 'Tenant ID for RLS isolation - derived from account on insert';
COMMENT ON INDEX idx_documents_tenant_account_active IS 'Optimized index for listing active documents by tenant/account';
