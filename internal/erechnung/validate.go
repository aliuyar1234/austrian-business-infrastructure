package erechnung

// ValidationError represents a validation error
type ValidationError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

// InvoiceValidationResult represents the result of invoice validation
type InvoiceValidationResult struct {
	Valid    bool              `json:"valid"`
	Errors   []ValidationError `json:"errors,omitempty"`
	Warnings []ValidationError `json:"warnings,omitempty"`
}

// ValidateEN16931 validates an invoice against EN16931 business rules
func ValidateEN16931(inv *Invoice) *InvoiceValidationResult {
	result := &InvoiceValidationResult{
		Valid:  true,
		Errors: make([]ValidationError, 0),
	}

	// BR-01: Invoice shall have a Specification identifier
	// (We add this automatically during generation)

	// BR-02: Invoice shall have an Invoice number (BT-1)
	if inv.ID == "" {
		result.addError("BR-02", "Invoice number (BT-1) is mandatory", "id")
	}

	// BR-03: Invoice shall have an Invoice issue date (BT-2)
	if inv.IssueDate.IsZero() {
		result.addError("BR-03", "Invoice issue date (BT-2) is mandatory", "issue_date")
	}

	// BR-04: Invoice shall have an Invoice type code (BT-3)
	if inv.InvoiceType == "" {
		result.addError("BR-04", "Invoice type code (BT-3) is mandatory", "invoice_type")
	}

	// BR-05: Invoice shall have an Invoice currency code (BT-5)
	if inv.Currency == "" {
		result.addError("BR-05", "Invoice currency code (BT-5) is mandatory", "currency")
	}

	// BR-06: Seller (BG-4) is mandatory
	if inv.Seller == nil {
		result.addError("BR-06", "Seller (BG-4) is mandatory", "seller")
	} else {
		// BR-CO-26: Seller name (BT-27) is mandatory
		if inv.Seller.Name == "" {
			result.addError("BT-27", "Seller name (BT-27) is mandatory", "seller.name")
		}
		// BR-09: Seller country code (BT-40) is mandatory
		if inv.Seller.Country == "" {
			result.addError("BR-09", "Seller country code (BT-40) is mandatory", "seller.country")
		}
	}

	// BR-07: Buyer (BG-7) is mandatory
	if inv.Buyer == nil {
		result.addError("BR-07", "Buyer (BG-7) is mandatory", "buyer")
	} else {
		// BR-CO-25: Buyer name (BT-44) is mandatory
		if inv.Buyer.Name == "" {
			result.addError("BT-44", "Buyer name (BT-44) is mandatory", "buyer.name")
		}
		// BR-11: Buyer country code (BT-55) is mandatory
		if inv.Buyer.Country == "" {
			result.addError("BR-11", "Buyer country code (BT-55) is mandatory", "buyer.country")
		}
	}

	// BR-16: Invoice shall have at least one Invoice line (BG-25)
	if len(inv.Lines) == 0 {
		result.addError("BR-16", "Invoice shall have at least one Invoice line (BG-25)", "lines")
	}

	// Validate each line
	for i, line := range inv.Lines {
		validateLine(result, line, i)
	}

	// BR-CO-10: Sum of Invoice line net amounts shall equal Tax exclusive amount
	// (Calculated automatically, but verify if manually set)

	// BR-CO-13: Invoice total with VAT = Tax exclusive amount + Tax amount
	// (Calculated automatically)

	// BR-CO-14: Amount due for payment = Invoice total with VAT - Paid amount + Rounding amount
	// (Calculated automatically)

	return result
}

// validateLine validates a single invoice line
func validateLine(result *InvoiceValidationResult, line *InvoiceLine, index int) {
	prefix := "lines[" + line.ID + "]"

	// BR-21: Invoice line identifier (BT-126) is mandatory
	if line.ID == "" {
		result.addError("BR-21", "Invoice line identifier (BT-126) is mandatory", prefix+".id")
	}

	// BR-22: Invoiced quantity (BT-129) is mandatory
	if line.Quantity == 0 {
		result.addError("BR-22", "Invoiced quantity (BT-129) is mandatory", prefix+".quantity")
	}

	// BR-23: Invoiced quantity unit of measure code (BT-130) is mandatory
	if line.UnitCode == "" {
		result.addError("BR-23", "Invoiced quantity unit of measure code (BT-130) is mandatory", prefix+".unit_code")
	}

	// BR-24: Invoice line net amount (BT-131) is mandatory
	// (Calculated automatically)

	// BR-25: Item name (BT-153) is mandatory
	if line.Description == "" {
		result.addError("BR-25", "Item name (BT-153) is mandatory", prefix+".description")
	}

	// BR-26: Item net price (BT-146) is mandatory
	if line.UnitPrice == 0 {
		result.addError("BR-26", "Item net price (BT-146) is mandatory", prefix+".unit_price")
	}

	// BR-CO-18: VAT category code (BT-151) is mandatory for each line
	if line.TaxCategory == "" {
		result.addError("BR-CO-18", "VAT category code (BT-151) is mandatory", prefix+".tax_category")
	}

	// Validate tax category specific rules
	switch line.TaxCategory {
	case TaxCategoryStandard, TaxCategoryReduced:
		// BR-S-05, BR-AA-05: VAT rate shall be greater than zero
		if line.TaxPercent <= 0 {
			result.addError("BR-S-05", "VAT rate shall be greater than zero for category "+line.TaxCategory, prefix+".tax_percent")
		}
	case TaxCategoryZero:
		// BR-Z-05: VAT rate shall be 0
		if line.TaxPercent != 0 {
			result.addError("BR-Z-05", "VAT rate shall be 0 for category Z", prefix+".tax_percent")
		}
	case TaxCategoryExempt:
		// BR-E-05: VAT rate shall be 0
		if line.TaxPercent != 0 {
			result.addError("BR-E-05", "VAT rate shall be 0 for category E", prefix+".tax_percent")
		}
	case TaxCategoryReverseCharge:
		// BR-AE-05: VAT rate shall be 0
		if line.TaxPercent != 0 {
			result.addError("BR-AE-05", "VAT rate shall be 0 for category AE (reverse charge)", prefix+".tax_percent")
		}
	}
}

// addError adds an error to the result
func (r *InvoiceValidationResult) addError(code, message, field string) {
	r.Valid = false
	r.Errors = append(r.Errors, ValidationError{
		Code:    code,
		Message: message,
		Field:   field,
	})
}

// addWarning adds a warning to the result
func (r *InvoiceValidationResult) addWarning(code, message, field string) {
	r.Warnings = append(r.Warnings, ValidationError{
		Code:    code,
		Message: message,
		Field:   field,
	})
}

// ValidateInvoice is an alias for ValidateEN16931
func ValidateInvoice(inv *Invoice) *InvoiceValidationResult {
	return ValidateEN16931(inv)
}
