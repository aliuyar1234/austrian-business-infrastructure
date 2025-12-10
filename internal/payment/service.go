package payment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"austrian-business-infrastructure/internal/sepa"
	"github.com/google/uuid"
)

var (
	ErrBatchNotDraft   = errors.New("batch is not in draft status")
	ErrNoItems         = errors.New("batch must have at least one item")
	ErrInvalidBatchType = errors.New("invalid batch type")
	ErrValidationFailed = errors.New("validation failed")
)

// Service handles payment business logic
type Service struct {
	repo *Repository
}

// NewService creates a new payment service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// CreateBatch creates a new payment batch
func (s *Service) CreateBatch(ctx context.Context, tenantID, userID uuid.UUID, input *CreateBatchInput) (*Batch, error) {
	// Validate items
	if len(input.Items) == 0 {
		return nil, ErrNoItems
	}

	// Validate batch type
	if input.Type != TypeCreditTransfer && input.Type != TypeDirectDebit {
		return nil, ErrInvalidBatchType
	}

	// Parse execution date
	var executionDate *time.Time
	if input.ExecutionDate != nil && *input.ExecutionDate != "" {
		d, err := time.Parse("2006-01-02", *input.ExecutionDate)
		if err != nil {
			return nil, fmt.Errorf("invalid execution_date format: %w", err)
		}
		executionDate = &d
	}

	batch := &Batch{
		TenantID:      tenantID,
		Name:          input.Name,
		Type:          input.Type,
		DebtorName:    input.DebtorName,
		DebtorIBAN:    input.DebtorIBAN,
		DebtorBIC:     input.DebtorBIC,
		CreditorID:    input.CreditorID,
		ExecutionDate: executionDate,
		CreatedBy:     &userID,
	}

	// Convert items
	items := make([]*Item, 0, len(input.Items))
	for _, itemInput := range input.Items {
		item := &Item{
			EndToEndID:     itemInput.EndToEndID,
			Amount:         itemInput.Amount,
			Currency:       itemInput.Currency,
			CreditorName:   itemInput.CreditorName,
			CreditorIBAN:   itemInput.CreditorIBAN,
			CreditorBIC:    itemInput.CreditorBIC,
			RemittanceInfo: itemInput.RemittanceInfo,
			Purpose:        itemInput.Purpose,
			MandateID:      itemInput.MandateID,
			SequenceType:   itemInput.SequenceType,
		}
		if item.Currency == "" {
			item.Currency = "EUR"
		}
		if item.EndToEndID == "" {
			item.EndToEndID = fmt.Sprintf("TXN-%s", uuid.New().String()[:8])
		}

		// Parse mandate date for direct debits
		if itemInput.MandateDate != nil && *itemInput.MandateDate != "" {
			d, err := time.Parse("2006-01-02", *itemInput.MandateDate)
			if err == nil {
				item.MandateDate = &d
			}
		}

		items = append(items, item)
	}

	return s.repo.CreateBatch(ctx, batch, items)
}

// GetBatch retrieves a batch by ID
func (s *Service) GetBatch(ctx context.Context, id, tenantID uuid.UUID) (*Batch, error) {
	return s.repo.GetBatchByID(ctx, id, tenantID)
}

// GetBatchWithItems retrieves a batch with its items
func (s *Service) GetBatchWithItems(ctx context.Context, id, tenantID uuid.UUID) (*Batch, []*Item, error) {
	batch, err := s.repo.GetBatchByID(ctx, id, tenantID)
	if err != nil {
		return nil, nil, err
	}

	items, err := s.repo.GetBatchItems(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	return batch, items, nil
}

// ListBatches lists batches with filtering
func (s *Service) ListBatches(ctx context.Context, filter ListFilter) ([]*Batch, int, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	return s.repo.ListBatches(ctx, filter)
}

// DeleteBatch deletes a batch (only drafts)
func (s *Service) DeleteBatch(ctx context.Context, id, tenantID uuid.UUID) error {
	batch, err := s.repo.GetBatchByID(ctx, id, tenantID)
	if err != nil {
		return err
	}

	if batch.Status != StatusDraft {
		return ErrBatchNotDraft
	}

	return s.repo.DeleteBatch(ctx, id, tenantID)
}

// ValidateBatch validates a payment batch
func (s *Service) ValidateBatch(ctx context.Context, id, tenantID uuid.UUID) (*Batch, error) {
	batch, items, err := s.GetBatchWithItems(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	validationErrors := s.validateBatchItems(batch, items)

	if len(validationErrors) > 0 {
		errJSON, _ := json.Marshal(validationErrors)
		batch.ValidationErrors = errJSON
		batch.Status = StatusFailed
	} else {
		batch.ValidationErrors = nil
		batch.Status = StatusValidated
	}

	if err := s.repo.UpdateBatch(ctx, batch); err != nil {
		return nil, err
	}

	return s.repo.GetBatchByID(ctx, id, tenantID)
}

// validateBatchItems validates items in a batch
func (s *Service) validateBatchItems(batch *Batch, items []*Item) []map[string]string {
	var errors []map[string]string

	for i, item := range items {
		// Validate IBAN format (basic check)
		if len(item.CreditorIBAN) < 15 || len(item.CreditorIBAN) > 34 {
			errors = append(errors, map[string]string{
				"index": fmt.Sprintf("%d", i),
				"field": "creditor_iban",
				"error": "invalid IBAN length",
			})
		}

		// Validate amount
		if item.Amount <= 0 {
			errors = append(errors, map[string]string{
				"index": fmt.Sprintf("%d", i),
				"field": "amount",
				"error": "amount must be positive",
			})
		}

		// For direct debits, mandate is required
		if batch.Type == TypeDirectDebit {
			if item.MandateID == nil || *item.MandateID == "" {
				errors = append(errors, map[string]string{
					"index": fmt.Sprintf("%d", i),
					"field": "mandate_id",
					"error": "mandate_id is required for direct debits",
				})
			}
		}
	}

	return errors
}

// GenerateXML generates pain.001 or pain.008 XML for the batch
func (s *Service) GenerateXML(ctx context.Context, id, tenantID uuid.UUID) ([]byte, error) {
	batch, items, err := s.GetBatchWithItems(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	var xmlContent []byte

	if batch.Type == TypeCreditTransfer {
		xmlContent, err = s.generatePain001(batch, items)
	} else {
		xmlContent, err = s.generatePain008(batch, items)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate XML: %w", err)
	}

	// Save XML
	if err := s.repo.SaveBatchXML(ctx, id, tenantID, xmlContent); err != nil {
		return nil, err
	}

	return xmlContent, nil
}

// generatePain001 generates a pain.001 credit transfer XML
func (s *Service) generatePain001(batch *Batch, items []*Item) ([]byte, error) {
	ct := sepa.NewCreditTransfer(
		fmt.Sprintf("BATCH-%s", batch.ID.String()[:8]),
		batch.DebtorName,
	)

	ct.Debtor = sepa.SEPAParty{Name: batch.DebtorName}
	ct.DebtorAccount = sepa.SEPAAccount{
		IBAN: batch.DebtorIBAN,
	}
	if batch.DebtorBIC != nil {
		ct.DebtorAccount.BIC = *batch.DebtorBIC
	}

	for _, item := range items {
		txn := &sepa.SEPACreditTransaction{
			InstructionID:  item.ID.String()[:8],
			EndToEndID:     item.EndToEndID,
			Amount:         item.Amount,
			Currency:       item.Currency,
			Creditor:       sepa.SEPAParty{Name: item.CreditorName},
			CreditorAccount: sepa.SEPAAccount{IBAN: item.CreditorIBAN},
		}
		if item.CreditorBIC != nil {
			txn.CreditorAccount.BIC = *item.CreditorBIC
		}
		if item.RemittanceInfo != nil {
			txn.RemittanceInfo = *item.RemittanceInfo
		}
		ct.Transactions = append(ct.Transactions, txn)
	}

	return sepa.GeneratePain001(ct)
}

// generatePain008 generates a pain.008 direct debit XML
func (s *Service) generatePain008(batch *Batch, items []*Item) ([]byte, error) {
	dd := sepa.NewDirectDebit(
		fmt.Sprintf("BATCH-%s", batch.ID.String()[:8]),
		sepa.SEPAParty{Name: batch.DebtorName},
	)

	dd.CreditorAccount = sepa.SEPAAccount{
		IBAN: batch.DebtorIBAN,
	}
	if batch.DebtorBIC != nil {
		dd.CreditorAccount.BIC = *batch.DebtorBIC
	}
	if batch.CreditorID != nil {
		dd.CreditorID = *batch.CreditorID
	}

	for _, item := range items {
		mandateDate := time.Now()
		if item.MandateDate != nil {
			mandateDate = *item.MandateDate
		}

		seqType := sepa.SequenceTypeRecurrent
		if item.SequenceType != nil {
			seqType = sepa.SequenceType(*item.SequenceType)
		}

		mandateID := ""
		if item.MandateID != nil {
			mandateID = *item.MandateID
		}

		txn := &sepa.SEPADirectDebitTransaction{
			InstructionID:  item.ID.String()[:8],
			EndToEndID:     item.EndToEndID,
			Amount:         item.Amount,
			Currency:       item.Currency,
			Debtor:         sepa.SEPAParty{Name: item.CreditorName},
			DebtorAccount:  sepa.SEPAAccount{IBAN: item.CreditorIBAN},
			MandateID:      mandateID,
			MandateDate:    mandateDate,
			SequenceType:   seqType,
		}
		if item.CreditorBIC != nil {
			txn.DebtorAccount.BIC = *item.CreditorBIC
		}
		if item.RemittanceInfo != nil {
			txn.RemittanceInfo = *item.RemittanceInfo
		}
		dd.Transactions = append(dd.Transactions, txn)
	}

	return sepa.GeneratePain008(dd)
}

// GetBatchXML retrieves stored XML
func (s *Service) GetBatchXML(ctx context.Context, id, tenantID uuid.UUID) ([]byte, error) {
	return s.repo.GetBatchXML(ctx, id, tenantID)
}

// ImportBankStatement imports a camt.053 bank statement
func (s *Service) ImportBankStatement(ctx context.Context, tenantID uuid.UUID, xmlData []byte) (*BankStatement, error) {
	// Parse camt.053
	sepaStmt, err := sepa.ParseCamt053(xmlData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse camt.053: %w", err)
	}

	// Convert to our types
	stmt := &BankStatement{
		TenantID:       tenantID,
		IBAN:           sepaStmt.Account.IBAN,
		StatementID:    sepaStmt.ID,
		StatementDate:  sepaStmt.CreationTime,
		OpeningBalance: sepaStmt.OpeningBalance,
		ClosingBalance: sepaStmt.ClosingBalance,
	}

	// Convert transactions
	txns := make([]*Transaction, 0, len(sepaStmt.Entries))
	for _, entry := range sepaStmt.Entries {
		txn := &Transaction{
			Amount:      entry.Amount,
			Currency:    entry.Currency,
			CreditDebit: string(entry.CreditDebit),
			BookingDate: entry.BookingDate,
		}
		if !entry.ValueDate.IsZero() {
			txn.ValueDate = &entry.ValueDate
		}
		if entry.Reference != "" {
			txn.Reference = &entry.Reference
		}
		if entry.EndToEndID != "" {
			txn.EndToEndID = &entry.EndToEndID
		}
		if entry.RemittanceInfo != "" {
			txn.RemittanceInfo = &entry.RemittanceInfo
		}
		if entry.CounterpartyName != "" {
			txn.CounterpartyName = &entry.CounterpartyName
		}
		if entry.CounterpartyIBAN != "" {
			txn.CounterpartyIBAN = &entry.CounterpartyIBAN
		}
		txns = append(txns, txn)
	}

	return s.repo.CreateBankStatement(ctx, stmt, txns)
}

// GetStatement retrieves a bank statement by ID
func (s *Service) GetStatement(ctx context.Context, id, tenantID uuid.UUID) (*BankStatement, error) {
	return s.repo.GetStatementByID(ctx, id, tenantID)
}

// GetStatementWithTransactions retrieves a statement with its transactions
func (s *Service) GetStatementWithTransactions(ctx context.Context, id, tenantID uuid.UUID) (*BankStatement, []*Transaction, error) {
	stmt, err := s.repo.GetStatementByID(ctx, id, tenantID)
	if err != nil {
		return nil, nil, err
	}

	txns, err := s.repo.GetStatementTransactions(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	return stmt, txns, nil
}

// ListStatements lists bank statements
func (s *Service) ListStatements(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*BankStatement, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.ListStatements(ctx, tenantID, limit, offset)
}

// DeleteStatement deletes a bank statement
func (s *Service) DeleteStatement(ctx context.Context, id, tenantID uuid.UUID) error {
	return s.repo.DeleteStatement(ctx, id, tenantID)
}

// MatchTransaction matches a transaction to a payment or invoice
func (s *Service) MatchTransaction(ctx context.Context, txnID uuid.UUID, paymentID, invoiceID *uuid.UUID) error {
	return s.repo.MatchTransaction(ctx, txnID, paymentID, invoiceID)
}

// ImportCSVBatch imports payments from CSV
func (s *Service) ImportCSVBatch(ctx context.Context, tenantID, userID uuid.UUID, name, debtorName, debtorIBAN string, csvData []byte) (*Batch, error) {
	// Parse CSV using sepa library
	txns, err := sepa.ParseCreditTransferCSV(csvData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(txns) == 0 {
		return nil, ErrNoItems
	}

	// Convert to batch input
	items := make([]ItemInput, 0, len(txns))
	for _, txn := range txns {
		item := ItemInput{
			EndToEndID:   txn.EndToEndID,
			Amount:       txn.Amount,
			Currency:     txn.Currency,
			CreditorName: txn.Creditor.Name,
			CreditorIBAN: txn.CreditorAccount.IBAN,
		}
		if txn.CreditorAccount.BIC != "" {
			item.CreditorBIC = &txn.CreditorAccount.BIC
		}
		if txn.RemittanceInfo != "" {
			item.RemittanceInfo = &txn.RemittanceInfo
		}
		items = append(items, item)
	}

	input := &CreateBatchInput{
		Name:       name,
		Type:       TypeCreditTransfer,
		DebtorName: debtorName,
		DebtorIBAN: debtorIBAN,
		Items:      items,
	}

	return s.CreateBatch(ctx, tenantID, userID, input)
}
