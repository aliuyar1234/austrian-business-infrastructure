package invoice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"austrian-business-infrastructure/internal/erechnung"
	"github.com/google/uuid"
)

var (
	ErrInvalidInvoiceType = errors.New("invalid invoice type")
	ErrInvoiceNotDraft    = errors.New("invoice is not in draft status")
	ErrNoItems            = errors.New("invoice must have at least one item")
	ErrValidationFailed   = errors.New("validation failed")
)

// Service handles invoice business logic
type Service struct {
	repo *Repository
}

// NewService creates a new invoice service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Create creates a new invoice
func (s *Service) Create(ctx context.Context, tenantID, userID uuid.UUID, input *CreateInvoiceInput) (*Invoice, error) {
	// Validate items
	if len(input.Items) == 0 {
		return nil, ErrNoItems
	}

	// Parse dates
	issueDate, err := time.Parse("2006-01-02", input.IssueDate)
	if err != nil {
		return nil, fmt.Errorf("invalid issue_date format: %w", err)
	}

	var dueDate *time.Time
	if input.DueDate != nil && *input.DueDate != "" {
		d, err := time.Parse("2006-01-02", *input.DueDate)
		if err != nil {
			return nil, fmt.Errorf("invalid due_date format: %w", err)
		}
		dueDate = &d
	}

	// Calculate totals
	var taxExclusive, taxAmount int64
	items := make([]*InvoiceItem, 0, len(input.Items))

	for i, itemInput := range input.Items {
		lineTotal := int64(float64(itemInput.UnitPrice) * itemInput.Quantity)
		lineTax := int64(float64(lineTotal) * itemInput.TaxPercent / 100)

		taxExclusive += lineTotal
		taxAmount += lineTax

		item := &InvoiceItem{
			LineNumber:  i + 1,
			Description: itemInput.Description,
			Quantity:    itemInput.Quantity,
			UnitCode:    itemInput.UnitCode,
			UnitPrice:   itemInput.UnitPrice,
			LineTotal:   lineTotal,
			TaxCategory: itemInput.TaxCategory,
			TaxPercent:  itemInput.TaxPercent,
			ItemID:      itemInput.ItemID,
			GTIN:        itemInput.GTIN,
		}
		items = append(items, item)
	}

	// Serialize addresses
	var sellerAddr, buyerAddr json.RawMessage
	if input.SellerAddress != nil {
		sellerAddr, _ = json.Marshal(input.SellerAddress)
	}
	if input.BuyerAddress != nil {
		buyerAddr, _ = json.Marshal(input.BuyerAddress)
	}

	inv := &Invoice{
		TenantID:           tenantID,
		InvoiceNumber:      input.InvoiceNumber,
		InvoiceType:        input.InvoiceType,
		IssueDate:          issueDate,
		DueDate:            dueDate,
		Currency:           input.Currency,
		SellerID:           input.SellerID,
		SellerName:         input.SellerName,
		SellerVAT:          input.SellerVAT,
		SellerAddress:      sellerAddr,
		BuyerID:            input.BuyerID,
		BuyerName:          input.BuyerName,
		BuyerVAT:           input.BuyerVAT,
		BuyerAddress:       buyerAddr,
		BuyerReference:     input.BuyerReference,
		OrderReference:     input.OrderReference,
		TaxExclusiveAmount: taxExclusive,
		TaxAmount:          taxAmount,
		TaxInclusiveAmount: taxExclusive + taxAmount,
		PayableAmount:      taxExclusive + taxAmount,
		PaymentTerms:       input.PaymentTerms,
		PaymentIBAN:        input.PaymentIBAN,
		PaymentBIC:         input.PaymentBIC,
		Notes:              input.Notes,
		CreatedBy:          &userID,
	}

	if inv.Currency == "" {
		inv.Currency = "EUR"
	}
	if inv.InvoiceType == "" {
		inv.InvoiceType = string(erechnung.InvoiceTypeCommercial)
	}

	return s.repo.Create(ctx, inv, items)
}

// Get retrieves an invoice by ID
func (s *Service) Get(ctx context.Context, id, tenantID uuid.UUID) (*Invoice, error) {
	return s.repo.GetByID(ctx, id, tenantID)
}

// GetWithItems retrieves an invoice with its items
func (s *Service) GetWithItems(ctx context.Context, id, tenantID uuid.UUID) (*Invoice, []*InvoiceItem, error) {
	inv, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, nil, err
	}

	items, err := s.repo.GetItems(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	return inv, items, nil
}

// List lists invoices with filtering
func (s *Service) List(ctx context.Context, filter ListFilter) ([]*Invoice, int, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	return s.repo.List(ctx, filter)
}

// Delete deletes an invoice (only drafts)
func (s *Service) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	inv, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return err
	}

	if inv.Status != StatusDraft {
		return ErrInvoiceNotDraft
	}

	return s.repo.Delete(ctx, id, tenantID)
}

// Validate validates an invoice
func (s *Service) Validate(ctx context.Context, id, tenantID uuid.UUID) (*Invoice, error) {
	inv, items, err := s.GetWithItems(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	// Build erechnung Invoice for validation
	ereInv := s.toErechnungInvoice(inv, items)

	// Validate
	validationResult := erechnung.ValidateInvoice(ereInv)

	if !validationResult.Valid {
		errJSON, _ := json.Marshal(validationResult.Errors)
		inv.ValidationStatus = "failed"
		inv.ValidationErrors = errJSON
	} else {
		inv.ValidationStatus = "passed"
		inv.ValidationErrors = nil
		inv.Status = StatusValidated
	}

	if err := s.repo.Update(ctx, inv); err != nil {
		return nil, err
	}

	return s.repo.GetByID(ctx, id, tenantID)
}

// GenerateXML generates XRechnung or ZUGFeRD XML
func (s *Service) GenerateXML(ctx context.Context, id, tenantID uuid.UUID, format string) ([]byte, error) {
	inv, items, err := s.GetWithItems(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	// Build erechnung Invoice
	ereInv := s.toErechnungInvoice(inv, items)

	// Generate XML
	var xmlContent []byte
	if format == FormatZUGFeRD {
		xmlContent, err = erechnung.GenerateZUGFeRD(ereInv)
	} else {
		xmlContent, err = erechnung.GenerateXRechnung(ereInv)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate %s XML: %w", format, err)
	}

	// Save XML
	if err := s.repo.SaveXML(ctx, id, tenantID, format, xmlContent); err != nil {
		return nil, err
	}

	// Update status
	inv.Status = StatusGenerated
	if err := s.repo.Update(ctx, inv); err != nil {
		return nil, err
	}

	return xmlContent, nil
}

// GetXML retrieves stored XML
func (s *Service) GetXML(ctx context.Context, id, tenantID uuid.UUID, format string) ([]byte, error) {
	return s.repo.GetXML(ctx, id, tenantID, format)
}

// Helper methods

func (s *Service) toErechnungInvoice(inv *Invoice, items []*InvoiceItem) *erechnung.Invoice {
	ereInv := &erechnung.Invoice{
		ID:          inv.InvoiceNumber,
		InvoiceType: erechnung.InvoiceType(inv.InvoiceType),
		IssueDate:   inv.IssueDate,
		Currency:    inv.Currency,
		Seller: &erechnung.InvoiceParty{
			Name: inv.SellerName,
		},
		Buyer: &erechnung.InvoiceParty{
			Name: inv.BuyerName,
		},
		Lines:              make([]*erechnung.InvoiceLine, 0, len(items)),
		TaxExclusiveAmount: inv.TaxExclusiveAmount,
		TaxAmount:          inv.TaxAmount,
		TaxInclusiveAmount: inv.TaxInclusiveAmount,
		PayableAmount:      inv.PayableAmount,
	}

	if inv.DueDate != nil {
		ereInv.DueDate = *inv.DueDate
	}

	if inv.SellerVAT != nil {
		ereInv.Seller.VATNumber = *inv.SellerVAT
	}
	if inv.BuyerVAT != nil {
		ereInv.Buyer.VATNumber = *inv.BuyerVAT
	}
	if inv.BuyerReference != nil {
		ereInv.BuyerReference = *inv.BuyerReference
	}
	if inv.OrderReference != nil {
		ereInv.OrderReference = *inv.OrderReference
	}
	if inv.PaymentTerms != nil {
		ereInv.PaymentTerms = *inv.PaymentTerms
	}
	if inv.Notes != nil {
		ereInv.Notes = *inv.Notes
	}

	// Parse addresses
	if len(inv.SellerAddress) > 0 {
		var addr Address
		if json.Unmarshal(inv.SellerAddress, &addr) == nil {
			ereInv.Seller.Street = addr.Street
			ereInv.Seller.City = addr.City
			ereInv.Seller.PostalCode = addr.PostalCode
			ereInv.Seller.Country = addr.Country
		}
	}
	if len(inv.BuyerAddress) > 0 {
		var addr Address
		if json.Unmarshal(inv.BuyerAddress, &addr) == nil {
			ereInv.Buyer.Street = addr.Street
			ereInv.Buyer.City = addr.City
			ereInv.Buyer.PostalCode = addr.PostalCode
			ereInv.Buyer.Country = addr.Country
		}
	}

	// Bank account
	if inv.PaymentIBAN != nil {
		ereInv.BankAccount = &erechnung.BankAccount{
			IBAN: *inv.PaymentIBAN,
		}
		if inv.PaymentBIC != nil {
			ereInv.BankAccount.BIC = *inv.PaymentBIC
		}
	}

	// Convert items
	for _, item := range items {
		ereLine := &erechnung.InvoiceLine{
			ID:          fmt.Sprintf("%d", item.LineNumber),
			Description: item.Description,
			Quantity:    item.Quantity,
			UnitCode:    item.UnitCode,
			UnitPrice:   item.UnitPrice,
			LineTotal:   item.LineTotal,
			TaxCategory: item.TaxCategory,
			TaxPercent:  item.TaxPercent,
		}
		if item.ItemID != nil {
			ereLine.ItemID = *item.ItemID
		}
		if item.GTIN != nil {
			ereLine.GTIN = *item.GTIN
		}
		ereInv.Lines = append(ereInv.Lines, ereLine)
	}

	return ereInv
}
