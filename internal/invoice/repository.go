package invoice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvoiceNotFound = errors.New("invoice not found")
	ErrDuplicateNumber = errors.New("invoice number already exists")
)

// Repository handles invoice database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new invoice repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create creates a new invoice with items
func (r *Repository) Create(ctx context.Context, inv *Invoice, items []*InvoiceItem) (*Invoice, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	inv.ID = uuid.New()
	inv.CreatedAt = time.Now()
	inv.UpdatedAt = inv.CreatedAt
	inv.Status = StatusDraft
	inv.ValidationStatus = "pending"

	query := `
		INSERT INTO invoices (
			id, tenant_id, invoice_number, invoice_type, issue_date, due_date,
			currency, seller_id, seller_name, seller_vat, seller_address,
			buyer_id, buyer_name, buyer_vat, buyer_address, buyer_reference,
			order_reference, tax_exclusive_amount, tax_amount, tax_inclusive_amount,
			payable_amount, payment_terms, payment_iban, payment_bic, notes,
			status, validation_status, created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30)
		RETURNING id`

	err = tx.QueryRow(ctx, query,
		inv.ID, inv.TenantID, inv.InvoiceNumber, inv.InvoiceType, inv.IssueDate, inv.DueDate,
		inv.Currency, inv.SellerID, inv.SellerName, inv.SellerVAT, inv.SellerAddress,
		inv.BuyerID, inv.BuyerName, inv.BuyerVAT, inv.BuyerAddress, inv.BuyerReference,
		inv.OrderReference, inv.TaxExclusiveAmount, inv.TaxAmount, inv.TaxInclusiveAmount,
		inv.PayableAmount, inv.PaymentTerms, inv.PaymentIBAN, inv.PaymentBIC, inv.Notes,
		inv.Status, inv.ValidationStatus, inv.CreatedBy, inv.CreatedAt, inv.UpdatedAt,
	).Scan(&inv.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	// Insert items
	for i, item := range items {
		item.ID = uuid.New()
		item.InvoiceID = inv.ID
		item.LineNumber = i + 1
		item.CreatedAt = time.Now()

		itemQuery := `
			INSERT INTO invoice_items (
				id, invoice_id, line_number, description, quantity, unit_code,
				unit_price, line_total, tax_category, tax_percent, item_id, gtin, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

		_, err = tx.Exec(ctx, itemQuery,
			item.ID, item.InvoiceID, item.LineNumber, item.Description, item.Quantity, item.UnitCode,
			item.UnitPrice, item.LineTotal, item.TaxCategory, item.TaxPercent, item.ItemID, item.GTIN, item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create invoice item: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return inv, nil
}

// GetByID retrieves an invoice by ID
func (r *Repository) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*Invoice, error) {
	query := `
		SELECT id, tenant_id, invoice_number, invoice_type, issue_date, due_date,
			currency, seller_id, seller_name, seller_vat, seller_address,
			buyer_id, buyer_name, buyer_vat, buyer_address, buyer_reference,
			order_reference, tax_exclusive_amount, tax_amount, tax_inclusive_amount,
			payable_amount, payment_terms, payment_iban, payment_bic, notes,
			status, validation_status, validation_errors,
			xrechnung_xml IS NOT NULL as has_xrechnung,
			zugferd_xml IS NOT NULL as has_zugferd,
			pdf_content IS NOT NULL as has_pdf,
			created_by, created_at, updated_at
		FROM invoices
		WHERE id = $1 AND tenant_id = $2`

	var inv Invoice
	var dueDate sql.NullTime
	var sellerID, buyerID, createdBy uuid.NullUUID
	var sellerVAT, buyerVAT, buyerRef, orderRef, paymentTerms, paymentIBAN, paymentBIC, notes sql.NullString
	var hasXRechnung, hasZUGFeRD, hasPDF bool

	err := r.db.QueryRow(ctx, query, id, tenantID).Scan(
		&inv.ID, &inv.TenantID, &inv.InvoiceNumber, &inv.InvoiceType, &inv.IssueDate, &dueDate,
		&inv.Currency, &sellerID, &inv.SellerName, &sellerVAT, &inv.SellerAddress,
		&buyerID, &inv.BuyerName, &buyerVAT, &inv.BuyerAddress, &buyerRef,
		&orderRef, &inv.TaxExclusiveAmount, &inv.TaxAmount, &inv.TaxInclusiveAmount,
		&inv.PayableAmount, &paymentTerms, &paymentIBAN, &paymentBIC, &notes,
		&inv.Status, &inv.ValidationStatus, &inv.ValidationErrors,
		&hasXRechnung, &hasZUGFeRD, &hasPDF,
		&createdBy, &inv.CreatedAt, &inv.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}

	if dueDate.Valid {
		inv.DueDate = &dueDate.Time
	}
	if sellerID.Valid {
		inv.SellerID = &sellerID.UUID
	}
	if buyerID.Valid {
		inv.BuyerID = &buyerID.UUID
	}
	if sellerVAT.Valid {
		inv.SellerVAT = &sellerVAT.String
	}
	if buyerVAT.Valid {
		inv.BuyerVAT = &buyerVAT.String
	}
	if buyerRef.Valid {
		inv.BuyerReference = &buyerRef.String
	}
	if orderRef.Valid {
		inv.OrderReference = &orderRef.String
	}
	if paymentTerms.Valid {
		inv.PaymentTerms = &paymentTerms.String
	}
	if paymentIBAN.Valid {
		inv.PaymentIBAN = &paymentIBAN.String
	}
	if paymentBIC.Valid {
		inv.PaymentBIC = &paymentBIC.String
	}
	if notes.Valid {
		inv.Notes = &notes.String
	}
	if createdBy.Valid {
		inv.CreatedBy = &createdBy.UUID
	}

	return &inv, nil
}

// GetItems retrieves all items for an invoice
func (r *Repository) GetItems(ctx context.Context, invoiceID uuid.UUID) ([]*InvoiceItem, error) {
	query := `
		SELECT id, invoice_id, line_number, description, quantity, unit_code,
			unit_price, line_total, tax_category, tax_percent, item_id, gtin, created_at
		FROM invoice_items
		WHERE invoice_id = $1
		ORDER BY line_number`

	rows, err := r.db.Query(ctx, query, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice items: %w", err)
	}
	defer rows.Close()

	var items []*InvoiceItem
	for rows.Next() {
		var item InvoiceItem
		var itemID, gtin sql.NullString

		err := rows.Scan(
			&item.ID, &item.InvoiceID, &item.LineNumber, &item.Description, &item.Quantity, &item.UnitCode,
			&item.UnitPrice, &item.LineTotal, &item.TaxCategory, &item.TaxPercent, &itemID, &gtin, &item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan invoice item: %w", err)
		}

		if itemID.Valid {
			item.ItemID = &itemID.String
		}
		if gtin.Valid {
			item.GTIN = &gtin.String
		}

		items = append(items, &item)
	}

	return items, nil
}

// List retrieves invoices with filtering
func (r *Repository) List(ctx context.Context, filter ListFilter) ([]*Invoice, int, error) {
	baseQuery := ` FROM invoices WHERE tenant_id = $1`
	args := []interface{}{filter.TenantID}
	argIdx := 2

	if filter.Status != nil {
		baseQuery += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *filter.Status)
		argIdx++
	}

	if filter.BuyerID != nil {
		baseQuery += fmt.Sprintf(" AND buyer_id = $%d", argIdx)
		args = append(args, *filter.BuyerID)
		argIdx++
	}

	if filter.DateFrom != nil {
		baseQuery += fmt.Sprintf(" AND issue_date >= $%d", argIdx)
		args = append(args, *filter.DateFrom)
		argIdx++
	}

	if filter.DateTo != nil {
		baseQuery += fmt.Sprintf(" AND issue_date <= $%d", argIdx)
		args = append(args, *filter.DateTo)
		argIdx++
	}

	if filter.Search != nil {
		baseQuery += fmt.Sprintf(" AND (invoice_number ILIKE $%d OR buyer_name ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+*filter.Search+"%")
		argIdx++
	}

	// Count total
	var total int
	countQuery := "SELECT COUNT(*)" + baseQuery
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count invoices: %w", err)
	}

	// Get paginated results
	selectQuery := `
		SELECT id, tenant_id, invoice_number, invoice_type, issue_date, due_date,
			currency, seller_name, seller_vat, buyer_name, buyer_vat,
			tax_exclusive_amount, tax_amount, tax_inclusive_amount, payable_amount,
			status, validation_status, created_at, updated_at
		` + baseQuery + `
		ORDER BY issue_date DESC, created_at DESC
		LIMIT $` + fmt.Sprintf("%d", argIdx) + ` OFFSET $` + fmt.Sprintf("%d", argIdx+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list invoices: %w", err)
	}
	defer rows.Close()

	var invoices []*Invoice
	for rows.Next() {
		var inv Invoice
		var dueDate sql.NullTime
		var sellerVAT, buyerVAT sql.NullString

		err := rows.Scan(
			&inv.ID, &inv.TenantID, &inv.InvoiceNumber, &inv.InvoiceType, &inv.IssueDate, &dueDate,
			&inv.Currency, &inv.SellerName, &sellerVAT, &inv.BuyerName, &buyerVAT,
			&inv.TaxExclusiveAmount, &inv.TaxAmount, &inv.TaxInclusiveAmount, &inv.PayableAmount,
			&inv.Status, &inv.ValidationStatus, &inv.CreatedAt, &inv.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan invoice: %w", err)
		}

		if dueDate.Valid {
			inv.DueDate = &dueDate.Time
		}
		if sellerVAT.Valid {
			inv.SellerVAT = &sellerVAT.String
		}
		if buyerVAT.Valid {
			inv.BuyerVAT = &buyerVAT.String
		}

		invoices = append(invoices, &inv)
	}

	return invoices, total, nil
}

// Update updates an invoice
func (r *Repository) Update(ctx context.Context, inv *Invoice) error {
	inv.UpdatedAt = time.Now()

	query := `
		UPDATE invoices SET
			status = $1,
			validation_status = $2,
			validation_errors = $3,
			updated_at = $4
		WHERE id = $5 AND tenant_id = $6`

	result, err := r.db.Exec(ctx, query,
		inv.Status, inv.ValidationStatus, inv.ValidationErrors,
		inv.UpdatedAt, inv.ID, inv.TenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to update invoice: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrInvoiceNotFound
	}

	return nil
}

// SaveXML saves generated XML content
func (r *Repository) SaveXML(ctx context.Context, id, tenantID uuid.UUID, format string, xmlContent []byte) error {
	var query string
	if format == FormatXRechnung {
		query = `UPDATE invoices SET xrechnung_xml = $1, updated_at = $2 WHERE id = $3 AND tenant_id = $4`
	} else {
		query = `UPDATE invoices SET zugferd_xml = $1, updated_at = $2 WHERE id = $3 AND tenant_id = $4`
	}

	_, err := r.db.Exec(ctx, query, xmlContent, time.Now(), id, tenantID)
	return err
}

// GetXML retrieves XML content
func (r *Repository) GetXML(ctx context.Context, id, tenantID uuid.UUID, format string) ([]byte, error) {
	var query string
	if format == FormatXRechnung {
		query = `SELECT xrechnung_xml FROM invoices WHERE id = $1 AND tenant_id = $2`
	} else {
		query = `SELECT zugferd_xml FROM invoices WHERE id = $1 AND tenant_id = $2`
	}

	var content []byte
	err := r.db.QueryRow(ctx, query, id, tenantID).Scan(&content)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvoiceNotFound
		}
		return nil, err
	}

	return content, nil
}

// Delete deletes an invoice (only drafts)
func (r *Repository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	query := `DELETE FROM invoices WHERE id = $1 AND tenant_id = $2 AND status = 'draft'`
	result, err := r.db.Exec(ctx, query, id, tenantID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrInvoiceNotFound
	}
	return nil
}
