package payment

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
	ErrBatchNotFound     = errors.New("batch not found")
	ErrStatementNotFound = errors.New("statement not found")
)

// Repository handles payment database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new payment repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// CreateBatch creates a new payment batch with items
func (r *Repository) CreateBatch(ctx context.Context, batch *Batch, items []*Item) (*Batch, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	batch.ID = uuid.New()
	batch.CreatedAt = time.Now()
	batch.UpdatedAt = batch.CreatedAt
	batch.Status = StatusDraft

	query := `
		INSERT INTO payment_batches (
			id, tenant_id, name, type, debtor_name, debtor_iban, debtor_bic,
			creditor_id, execution_date, item_count, total_amount, status,
			created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id`

	err = tx.QueryRow(ctx, query,
		batch.ID, batch.TenantID, batch.Name, batch.Type, batch.DebtorName, batch.DebtorIBAN, batch.DebtorBIC,
		batch.CreditorID, batch.ExecutionDate, len(items), batch.TotalAmount, batch.Status,
		batch.CreatedBy, batch.CreatedAt, batch.UpdatedAt,
	).Scan(&batch.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to create batch: %w", err)
	}

	// Insert items and calculate total
	var totalAmount int64
	for _, item := range items {
		item.ID = uuid.New()
		item.BatchID = batch.ID
		item.Status = StatusDraft
		item.CreatedAt = time.Now()
		totalAmount += item.Amount

		itemQuery := `
			INSERT INTO payment_items (
				id, batch_id, end_to_end_id, amount, currency, creditor_name, creditor_iban,
				creditor_bic, remittance_info, purpose, mandate_id, mandate_date,
				sequence_type, status, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`

		_, err = tx.Exec(ctx, itemQuery,
			item.ID, item.BatchID, item.EndToEndID, item.Amount, item.Currency, item.CreditorName, item.CreditorIBAN,
			item.CreditorBIC, item.RemittanceInfo, item.Purpose, item.MandateID, item.MandateDate,
			item.SequenceType, item.Status, item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create payment item: %w", err)
		}
	}

	// Update batch with calculated totals
	batch.ItemCount = len(items)
	batch.TotalAmount = totalAmount
	_, err = tx.Exec(ctx, `UPDATE payment_batches SET item_count = $1, total_amount = $2 WHERE id = $3`,
		batch.ItemCount, batch.TotalAmount, batch.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update batch totals: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return batch, nil
}

// GetBatchByID retrieves a batch by ID
func (r *Repository) GetBatchByID(ctx context.Context, id, tenantID uuid.UUID) (*Batch, error) {
	query := `
		SELECT id, tenant_id, name, type, debtor_name, debtor_iban, debtor_bic,
			creditor_id, execution_date, item_count, total_amount, status,
			validation_errors, xml_content IS NOT NULL as has_xml, generated_at, sent_at,
			created_by, created_at, updated_at
		FROM payment_batches
		WHERE id = $1 AND tenant_id = $2`

	var batch Batch
	var debtorBIC, creditorID sql.NullString
	var executionDate, generatedAt, sentAt sql.NullTime
	var createdBy uuid.NullUUID
	var hasXML bool

	err := r.db.QueryRow(ctx, query, id, tenantID).Scan(
		&batch.ID, &batch.TenantID, &batch.Name, &batch.Type, &batch.DebtorName, &batch.DebtorIBAN, &debtorBIC,
		&creditorID, &executionDate, &batch.ItemCount, &batch.TotalAmount, &batch.Status,
		&batch.ValidationErrors, &hasXML, &generatedAt, &sentAt,
		&createdBy, &batch.CreatedAt, &batch.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBatchNotFound
		}
		return nil, fmt.Errorf("failed to get batch: %w", err)
	}

	if debtorBIC.Valid {
		batch.DebtorBIC = &debtorBIC.String
	}
	if creditorID.Valid {
		batch.CreditorID = &creditorID.String
	}
	if executionDate.Valid {
		batch.ExecutionDate = &executionDate.Time
	}
	if generatedAt.Valid {
		batch.GeneratedAt = &generatedAt.Time
	}
	if sentAt.Valid {
		batch.SentAt = &sentAt.Time
	}
	if createdBy.Valid {
		batch.CreatedBy = &createdBy.UUID
	}

	return &batch, nil
}

// GetBatchItems retrieves all items for a batch
func (r *Repository) GetBatchItems(ctx context.Context, batchID uuid.UUID) ([]*Item, error) {
	query := `
		SELECT id, batch_id, end_to_end_id, amount, currency, creditor_name, creditor_iban,
			creditor_bic, remittance_info, purpose, mandate_id, mandate_date,
			sequence_type, status, error_message, created_at
		FROM payment_items
		WHERE batch_id = $1
		ORDER BY created_at`

	rows, err := r.db.Query(ctx, query, batchID)
	if err != nil {
		return nil, fmt.Errorf("failed to get batch items: %w", err)
	}
	defer rows.Close()

	var items []*Item
	for rows.Next() {
		var item Item
		var creditorBIC, remittanceInfo, purpose, mandateID, sequenceType, errorMsg sql.NullString
		var mandateDate sql.NullTime

		err := rows.Scan(
			&item.ID, &item.BatchID, &item.EndToEndID, &item.Amount, &item.Currency, &item.CreditorName, &item.CreditorIBAN,
			&creditorBIC, &remittanceInfo, &purpose, &mandateID, &mandateDate,
			&sequenceType, &item.Status, &errorMsg, &item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payment item: %w", err)
		}

		if creditorBIC.Valid {
			item.CreditorBIC = &creditorBIC.String
		}
		if remittanceInfo.Valid {
			item.RemittanceInfo = &remittanceInfo.String
		}
		if purpose.Valid {
			item.Purpose = &purpose.String
		}
		if mandateID.Valid {
			item.MandateID = &mandateID.String
		}
		if mandateDate.Valid {
			item.MandateDate = &mandateDate.Time
		}
		if sequenceType.Valid {
			item.SequenceType = &sequenceType.String
		}
		if errorMsg.Valid {
			item.ErrorMessage = &errorMsg.String
		}

		items = append(items, &item)
	}

	return items, nil
}

// ListBatches lists batches with filtering
func (r *Repository) ListBatches(ctx context.Context, filter ListFilter) ([]*Batch, int, error) {
	baseQuery := ` FROM payment_batches WHERE tenant_id = $1`
	args := []interface{}{filter.TenantID}
	argIdx := 2

	if filter.Type != nil {
		baseQuery += fmt.Sprintf(" AND type = $%d", argIdx)
		args = append(args, *filter.Type)
		argIdx++
	}

	if filter.Status != nil {
		baseQuery += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *filter.Status)
		argIdx++
	}

	if filter.DateFrom != nil {
		baseQuery += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, *filter.DateFrom)
		argIdx++
	}

	if filter.DateTo != nil {
		baseQuery += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, *filter.DateTo)
		argIdx++
	}

	// Count total
	var total int
	countQuery := "SELECT COUNT(*)" + baseQuery
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count batches: %w", err)
	}

	// Get paginated results
	selectQuery := `
		SELECT id, tenant_id, name, type, debtor_name, debtor_iban,
			item_count, total_amount, status, generated_at, sent_at, created_at, updated_at
		` + baseQuery + `
		ORDER BY created_at DESC
		LIMIT $` + fmt.Sprintf("%d", argIdx) + ` OFFSET $` + fmt.Sprintf("%d", argIdx+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list batches: %w", err)
	}
	defer rows.Close()

	var batches []*Batch
	for rows.Next() {
		var batch Batch
		var generatedAt, sentAt sql.NullTime

		err := rows.Scan(
			&batch.ID, &batch.TenantID, &batch.Name, &batch.Type, &batch.DebtorName, &batch.DebtorIBAN,
			&batch.ItemCount, &batch.TotalAmount, &batch.Status, &generatedAt, &sentAt, &batch.CreatedAt, &batch.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan batch: %w", err)
		}

		if generatedAt.Valid {
			batch.GeneratedAt = &generatedAt.Time
		}
		if sentAt.Valid {
			batch.SentAt = &sentAt.Time
		}

		batches = append(batches, &batch)
	}

	return batches, total, nil
}

// UpdateBatch updates a batch
func (r *Repository) UpdateBatch(ctx context.Context, batch *Batch) error {
	batch.UpdatedAt = time.Now()

	query := `
		UPDATE payment_batches SET
			status = $1,
			validation_errors = $2,
			updated_at = $3
		WHERE id = $4 AND tenant_id = $5`

	result, err := r.db.Exec(ctx, query,
		batch.Status, batch.ValidationErrors, batch.UpdatedAt, batch.ID, batch.TenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to update batch: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrBatchNotFound
	}

	return nil
}

// SaveBatchXML saves generated XML content
func (r *Repository) SaveBatchXML(ctx context.Context, id, tenantID uuid.UUID, xmlContent []byte) error {
	now := time.Now()
	query := `UPDATE payment_batches SET xml_content = $1, generated_at = $2, status = $3, updated_at = $4 WHERE id = $5 AND tenant_id = $6`
	_, err := r.db.Exec(ctx, query, xmlContent, now, StatusGenerated, now, id, tenantID)
	return err
}

// GetBatchXML retrieves XML content
func (r *Repository) GetBatchXML(ctx context.Context, id, tenantID uuid.UUID) ([]byte, error) {
	query := `SELECT xml_content FROM payment_batches WHERE id = $1 AND tenant_id = $2`
	var content []byte
	err := r.db.QueryRow(ctx, query, id, tenantID).Scan(&content)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBatchNotFound
		}
		return nil, err
	}
	return content, nil
}

// DeleteBatch deletes a batch (only drafts)
func (r *Repository) DeleteBatch(ctx context.Context, id, tenantID uuid.UUID) error {
	query := `DELETE FROM payment_batches WHERE id = $1 AND tenant_id = $2 AND status = 'draft'`
	result, err := r.db.Exec(ctx, query, id, tenantID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrBatchNotFound
	}
	return nil
}

// CreateBankStatement creates a bank statement with transactions
func (r *Repository) CreateBankStatement(ctx context.Context, stmt *BankStatement, txns []*Transaction) (*BankStatement, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	stmt.ID = uuid.New()
	stmt.ImportedAt = time.Now()
	stmt.CreatedAt = stmt.ImportedAt

	query := `
		INSERT INTO bank_statements (
			id, tenant_id, iban, statement_id, statement_date, opening_balance,
			closing_balance, entry_count, imported_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`

	err = tx.QueryRow(ctx, query,
		stmt.ID, stmt.TenantID, stmt.IBAN, stmt.StatementID, stmt.StatementDate, stmt.OpeningBalance,
		stmt.ClosingBalance, len(txns), stmt.ImportedAt, stmt.CreatedAt,
	).Scan(&stmt.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to create statement: %w", err)
	}

	// Insert transactions
	for _, txn := range txns {
		txn.ID = uuid.New()
		txn.StatementID = stmt.ID
		txn.CreatedAt = time.Now()

		txnQuery := `
			INSERT INTO transactions (
				id, statement_id, amount, currency, credit_debit, booking_date, value_date,
				reference, end_to_end_id, remittance_info, counterparty_name, counterparty_iban,
				created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

		_, err = tx.Exec(ctx, txnQuery,
			txn.ID, txn.StatementID, txn.Amount, txn.Currency, txn.CreditDebit, txn.BookingDate, txn.ValueDate,
			txn.Reference, txn.EndToEndID, txn.RemittanceInfo, txn.CounterpartyName, txn.CounterpartyIBAN,
			txn.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create transaction: %w", err)
		}
	}

	stmt.EntryCount = len(txns)

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return stmt, nil
}

// GetStatementByID retrieves a bank statement by ID
func (r *Repository) GetStatementByID(ctx context.Context, id, tenantID uuid.UUID) (*BankStatement, error) {
	query := `
		SELECT id, tenant_id, iban, statement_id, statement_date, opening_balance,
			closing_balance, entry_count, imported_at, created_at
		FROM bank_statements
		WHERE id = $1 AND tenant_id = $2`

	var stmt BankStatement
	err := r.db.QueryRow(ctx, query, id, tenantID).Scan(
		&stmt.ID, &stmt.TenantID, &stmt.IBAN, &stmt.StatementID, &stmt.StatementDate, &stmt.OpeningBalance,
		&stmt.ClosingBalance, &stmt.EntryCount, &stmt.ImportedAt, &stmt.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStatementNotFound
		}
		return nil, fmt.Errorf("failed to get statement: %w", err)
	}

	return &stmt, nil
}

// GetStatementTransactions retrieves all transactions for a statement
func (r *Repository) GetStatementTransactions(ctx context.Context, statementID uuid.UUID) ([]*Transaction, error) {
	query := `
		SELECT id, statement_id, amount, currency, credit_debit, booking_date, value_date,
			reference, end_to_end_id, remittance_info, counterparty_name, counterparty_iban,
			matched_payment_id, matched_invoice_id, created_at
		FROM transactions
		WHERE statement_id = $1
		ORDER BY booking_date, created_at`

	rows, err := r.db.Query(ctx, query, statementID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}
	defer rows.Close()

	var txns []*Transaction
	for rows.Next() {
		var txn Transaction
		var valueDate sql.NullTime
		var reference, endToEndID, remittanceInfo, counterpartyName, counterpartyIBAN sql.NullString
		var matchedPaymentID, matchedInvoiceID uuid.NullUUID

		err := rows.Scan(
			&txn.ID, &txn.StatementID, &txn.Amount, &txn.Currency, &txn.CreditDebit, &txn.BookingDate, &valueDate,
			&reference, &endToEndID, &remittanceInfo, &counterpartyName, &counterpartyIBAN,
			&matchedPaymentID, &matchedInvoiceID, &txn.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}

		if valueDate.Valid {
			txn.ValueDate = &valueDate.Time
		}
		if reference.Valid {
			txn.Reference = &reference.String
		}
		if endToEndID.Valid {
			txn.EndToEndID = &endToEndID.String
		}
		if remittanceInfo.Valid {
			txn.RemittanceInfo = &remittanceInfo.String
		}
		if counterpartyName.Valid {
			txn.CounterpartyName = &counterpartyName.String
		}
		if counterpartyIBAN.Valid {
			txn.CounterpartyIBAN = &counterpartyIBAN.String
		}
		if matchedPaymentID.Valid {
			txn.MatchedPaymentID = &matchedPaymentID.UUID
		}
		if matchedInvoiceID.Valid {
			txn.MatchedInvoiceID = &matchedInvoiceID.UUID
		}

		txns = append(txns, &txn)
	}

	return txns, nil
}

// ListStatements lists bank statements
func (r *Repository) ListStatements(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*BankStatement, int, error) {
	// Count total
	var total int
	countQuery := `SELECT COUNT(*) FROM bank_statements WHERE tenant_id = $1`
	err := r.db.QueryRow(ctx, countQuery, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count statements: %w", err)
	}

	// Get paginated results
	selectQuery := `
		SELECT id, tenant_id, iban, statement_id, statement_date, opening_balance,
			closing_balance, entry_count, imported_at, created_at
		FROM bank_statements
		WHERE tenant_id = $1
		ORDER BY statement_date DESC, imported_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, selectQuery, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list statements: %w", err)
	}
	defer rows.Close()

	var statements []*BankStatement
	for rows.Next() {
		var stmt BankStatement
		err := rows.Scan(
			&stmt.ID, &stmt.TenantID, &stmt.IBAN, &stmt.StatementID, &stmt.StatementDate, &stmt.OpeningBalance,
			&stmt.ClosingBalance, &stmt.EntryCount, &stmt.ImportedAt, &stmt.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan statement: %w", err)
		}
		statements = append(statements, &stmt)
	}

	return statements, total, nil
}

// DeleteStatement deletes a bank statement
func (r *Repository) DeleteStatement(ctx context.Context, id, tenantID uuid.UUID) error {
	query := `DELETE FROM bank_statements WHERE id = $1 AND tenant_id = $2`
	result, err := r.db.Exec(ctx, query, id, tenantID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrStatementNotFound
	}
	return nil
}

// MatchTransaction updates a transaction with matched payment/invoice
func (r *Repository) MatchTransaction(ctx context.Context, txnID uuid.UUID, paymentID, invoiceID *uuid.UUID) error {
	query := `UPDATE transactions SET matched_payment_id = $1, matched_invoice_id = $2 WHERE id = $3`
	_, err := r.db.Exec(ctx, query, paymentID, invoiceID, txnID)
	return err
}
