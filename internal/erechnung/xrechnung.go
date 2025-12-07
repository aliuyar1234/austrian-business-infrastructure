package erechnung

import (
	"bytes"
	"encoding/xml"
	"fmt"
)

const (
	UBLInvoiceNS = "urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"
	CACNS        = "urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2"
	CBCNS        = "urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2"
)

// UBLInvoice represents a UBL 2.1 Invoice for XRechnung
type UBLInvoice struct {
	XMLName xml.Name `xml:"Invoice"`
	XMLNS   string   `xml:"xmlns,attr"`
	CAC     string   `xml:"xmlns:cac,attr"`
	CBC     string   `xml:"xmlns:cbc,attr"`

	// BT-24: Specification identifier
	CustomizationID string `xml:"cbc:CustomizationID"`

	// BT-23: Business process type
	ProfileID string `xml:"cbc:ProfileID,omitempty"`

	// BT-1: Invoice number
	ID string `xml:"cbc:ID"`

	// BT-2: Issue date
	IssueDate string `xml:"cbc:IssueDate"`

	// BT-9: Due date
	DueDate string `xml:"cbc:DueDate,omitempty"`

	// BT-3: Invoice type code
	InvoiceTypeCode string `xml:"cbc:InvoiceTypeCode"`

	// BT-22: Note
	Note string `xml:"cbc:Note,omitempty"`

	// BT-5: Currency code
	DocumentCurrencyCode string `xml:"cbc:DocumentCurrencyCode"`

	// BT-10: Buyer reference
	BuyerReference string `xml:"cbc:BuyerReference,omitempty"`

	// BT-13: Order reference
	OrderReference *UBLOrderReference `xml:"cac:OrderReference,omitempty"`

	// BG-4: Seller
	AccountingSupplierParty *UBLParty `xml:"cac:AccountingSupplierParty"`

	// BG-7: Buyer
	AccountingCustomerParty *UBLParty `xml:"cac:AccountingCustomerParty"`

	// BG-16: Payment means
	PaymentMeans *UBLPaymentMeans `xml:"cac:PaymentMeans,omitempty"`

	// BT-20: Payment terms
	PaymentTerms *UBLPaymentTerms `xml:"cac:PaymentTerms,omitempty"`

	// BG-23: Tax total
	TaxTotal *UBLTaxTotal `xml:"cac:TaxTotal"`

	// BG-22: Document totals
	LegalMonetaryTotal *UBLMonetaryTotal `xml:"cac:LegalMonetaryTotal"`

	// BG-25: Invoice lines
	InvoiceLines []*UBLInvoiceLine `xml:"cac:InvoiceLine"`
}

// UBLOrderReference represents an order reference
type UBLOrderReference struct {
	ID string `xml:"cbc:ID"`
}

// UBLParty represents a party (supplier/customer)
type UBLParty struct {
	Party *UBLPartyDetails `xml:"cac:Party"`
}

// UBLPartyDetails contains party details
type UBLPartyDetails struct {
	PartyIdentification *UBLPartyIdentification `xml:"cac:PartyIdentification,omitempty"`
	PartyName           *UBLPartyName           `xml:"cac:PartyName"`
	PostalAddress       *UBLPostalAddress       `xml:"cac:PostalAddress"`
	PartyTaxScheme      *UBLPartyTaxScheme      `xml:"cac:PartyTaxScheme,omitempty"`
	Contact             *UBLContact             `xml:"cac:Contact,omitempty"`
}

// UBLPartyIdentification represents party identification
type UBLPartyIdentification struct {
	ID string `xml:"cbc:ID"`
}

// UBLPartyName represents a party name
type UBLPartyName struct {
	Name string `xml:"cbc:Name"`
}

// UBLPostalAddress represents a postal address
type UBLPostalAddress struct {
	StreetName           string      `xml:"cbc:StreetName,omitempty"`
	AdditionalStreetName string      `xml:"cbc:AdditionalStreetName,omitempty"`
	CityName             string      `xml:"cbc:CityName,omitempty"`
	PostalZone           string      `xml:"cbc:PostalZone,omitempty"`
	Country              *UBLCountry `xml:"cac:Country"`
}

// UBLCountry represents a country
type UBLCountry struct {
	IdentificationCode string `xml:"cbc:IdentificationCode"`
}

// UBLPartyTaxScheme represents tax scheme information
type UBLPartyTaxScheme struct {
	CompanyID string        `xml:"cbc:CompanyID"`
	TaxScheme *UBLTaxScheme `xml:"cac:TaxScheme"`
}

// UBLTaxScheme represents a tax scheme
type UBLTaxScheme struct {
	ID string `xml:"cbc:ID"`
}

// UBLContact represents contact information
type UBLContact struct {
	Name           string `xml:"cbc:Name,omitempty"`
	Telephone      string `xml:"cbc:Telephone,omitempty"`
	ElectronicMail string `xml:"cbc:ElectronicMail,omitempty"`
}

// UBLPaymentMeans represents payment means
type UBLPaymentMeans struct {
	PaymentMeansCode       string                  `xml:"cbc:PaymentMeansCode"`
	PayeeFinancialAccount *UBLFinancialAccount    `xml:"cac:PayeeFinancialAccount,omitempty"`
}

// UBLFinancialAccount represents a financial account
type UBLFinancialAccount struct {
	ID                         string                      `xml:"cbc:ID"`
	Name                       string                      `xml:"cbc:Name,omitempty"`
	FinancialInstitutionBranch *UBLFinancialInstitution    `xml:"cac:FinancialInstitutionBranch,omitempty"`
}

// UBLFinancialInstitution represents a financial institution
type UBLFinancialInstitution struct {
	ID string `xml:"cbc:ID"`
}

// UBLPaymentTerms represents payment terms
type UBLPaymentTerms struct {
	Note string `xml:"cbc:Note"`
}

// UBLTaxTotal represents tax totals
type UBLTaxTotal struct {
	TaxAmount   *UBLAmount       `xml:"cbc:TaxAmount"`
	TaxSubtotal []*UBLTaxSubtotal `xml:"cac:TaxSubtotal"`
}

// UBLTaxSubtotal represents a tax subtotal
type UBLTaxSubtotal struct {
	TaxableAmount *UBLAmount      `xml:"cbc:TaxableAmount"`
	TaxAmount     *UBLAmount      `xml:"cbc:TaxAmount"`
	TaxCategory   *UBLTaxCategory `xml:"cac:TaxCategory"`
}

// UBLTaxCategory represents a tax category
type UBLTaxCategory struct {
	ID                   string        `xml:"cbc:ID"`
	Percent              float64       `xml:"cbc:Percent"`
	TaxExemptionReason   string        `xml:"cbc:TaxExemptionReason,omitempty"`
	TaxScheme            *UBLTaxScheme `xml:"cac:TaxScheme"`
}

// UBLMonetaryTotal represents monetary totals
type UBLMonetaryTotal struct {
	LineExtensionAmount  *UBLAmount `xml:"cbc:LineExtensionAmount"`
	TaxExclusiveAmount   *UBLAmount `xml:"cbc:TaxExclusiveAmount"`
	TaxInclusiveAmount   *UBLAmount `xml:"cbc:TaxInclusiveAmount"`
	PayableAmount        *UBLAmount `xml:"cbc:PayableAmount"`
}

// UBLAmount represents a monetary amount
type UBLAmount struct {
	CurrencyID string  `xml:"currencyID,attr"`
	Value      float64 `xml:",chardata"`
}

// UBLInvoiceLine represents an invoice line
type UBLInvoiceLine struct {
	ID                  string              `xml:"cbc:ID"`
	InvoicedQuantity    *UBLQuantity        `xml:"cbc:InvoicedQuantity"`
	LineExtensionAmount *UBLAmount          `xml:"cbc:LineExtensionAmount"`
	Item                *UBLItem            `xml:"cac:Item"`
	Price               *UBLPrice           `xml:"cac:Price"`
}

// UBLQuantity represents a quantity
type UBLQuantity struct {
	UnitCode string  `xml:"unitCode,attr"`
	Value    float64 `xml:",chardata"`
}

// UBLItem represents an item
type UBLItem struct {
	Description              string                    `xml:"cbc:Description,omitempty"`
	Name                     string                    `xml:"cbc:Name"`
	SellersItemIdentification *UBLItemIdentification   `xml:"cac:SellersItemIdentification,omitempty"`
	StandardItemIdentification *UBLItemIdentification  `xml:"cac:StandardItemIdentification,omitempty"`
	ClassifiedTaxCategory    *UBLClassifiedTaxCategory `xml:"cac:ClassifiedTaxCategory"`
}

// UBLItemIdentification represents item identification
type UBLItemIdentification struct {
	ID string `xml:"cbc:ID"`
}

// UBLClassifiedTaxCategory represents classified tax category
type UBLClassifiedTaxCategory struct {
	ID        string        `xml:"cbc:ID"`
	Percent   float64       `xml:"cbc:Percent"`
	TaxScheme *UBLTaxScheme `xml:"cac:TaxScheme"`
}

// UBLPrice represents a price
type UBLPrice struct {
	PriceAmount *UBLAmount `xml:"cbc:PriceAmount"`
}

// GenerateXRechnung generates XRechnung (UBL 2.1) XML from an invoice
func GenerateXRechnung(inv *Invoice) ([]byte, error) {
	// Calculate totals if not done
	if inv.TaxExclusiveAmount == 0 {
		if err := inv.CalculateTotals(); err != nil {
			return nil, fmt.Errorf("failed to calculate totals: %w", err)
		}
	}

	ubl := &UBLInvoice{
		XMLNS:                UBLInvoiceNS,
		CAC:                  CACNS,
		CBC:                  CBCNS,
		CustomizationID:      "urn:cen.eu:en16931:2017#compliant#urn:xoev-de:kosit:standard:xrechnung_2.3",
		ProfileID:            "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0",
		ID:                   inv.ID,
		IssueDate:            inv.IssueDate.Format("2006-01-02"),
		InvoiceTypeCode:      string(inv.InvoiceType),
		DocumentCurrencyCode: inv.Currency,
		Note:                 inv.Notes,
		BuyerReference:       inv.BuyerReference,
	}

	if !inv.DueDate.IsZero() {
		ubl.DueDate = inv.DueDate.Format("2006-01-02")
	}

	if inv.OrderReference != "" {
		ubl.OrderReference = &UBLOrderReference{ID: inv.OrderReference}
	}

	// Seller
	if inv.Seller != nil {
		ubl.AccountingSupplierParty = convertPartyToUBL(inv.Seller)
	}

	// Buyer
	if inv.Buyer != nil {
		ubl.AccountingCustomerParty = convertPartyToUBL(inv.Buyer)
	}

	// Payment means
	if inv.PaymentMeans != "" {
		ubl.PaymentMeans = &UBLPaymentMeans{
			PaymentMeansCode: string(inv.PaymentMeans),
		}
		if inv.BankAccount != nil {
			ubl.PaymentMeans.PayeeFinancialAccount = &UBLFinancialAccount{
				ID:   inv.BankAccount.IBAN,
				Name: inv.BankAccount.Name,
			}
			if inv.BankAccount.BIC != "" {
				ubl.PaymentMeans.PayeeFinancialAccount.FinancialInstitutionBranch = &UBLFinancialInstitution{
					ID: inv.BankAccount.BIC,
				}
			}
		}
	}

	// Payment terms
	if inv.PaymentTerms != "" {
		ubl.PaymentTerms = &UBLPaymentTerms{Note: inv.PaymentTerms}
	}

	// Tax total
	ubl.TaxTotal = &UBLTaxTotal{
		TaxAmount: &UBLAmount{CurrencyID: inv.Currency, Value: AmountEUR(inv.TaxAmount)},
	}
	for _, ts := range inv.TaxSubtotals {
		ubl.TaxTotal.TaxSubtotal = append(ubl.TaxTotal.TaxSubtotal, &UBLTaxSubtotal{
			TaxableAmount: &UBLAmount{CurrencyID: inv.Currency, Value: AmountEUR(ts.TaxableAmount)},
			TaxAmount:     &UBLAmount{CurrencyID: inv.Currency, Value: AmountEUR(ts.TaxAmount)},
			TaxCategory: &UBLTaxCategory{
				ID:      ts.TaxCategory,
				Percent: ts.TaxPercent,
				TaxExemptionReason: ts.ExemptionReason,
				TaxScheme: &UBLTaxScheme{ID: "VAT"},
			},
		})
	}

	// Monetary totals
	ubl.LegalMonetaryTotal = &UBLMonetaryTotal{
		LineExtensionAmount: &UBLAmount{CurrencyID: inv.Currency, Value: AmountEUR(inv.TaxExclusiveAmount)},
		TaxExclusiveAmount:  &UBLAmount{CurrencyID: inv.Currency, Value: AmountEUR(inv.TaxExclusiveAmount)},
		TaxInclusiveAmount:  &UBLAmount{CurrencyID: inv.Currency, Value: AmountEUR(inv.TaxInclusiveAmount)},
		PayableAmount:       &UBLAmount{CurrencyID: inv.Currency, Value: AmountEUR(inv.PayableAmount)},
	}

	// Invoice lines
	for _, line := range inv.Lines {
		ublLine := &UBLInvoiceLine{
			ID: line.ID,
			InvoicedQuantity: &UBLQuantity{
				UnitCode: line.UnitCode,
				Value:    line.Quantity,
			},
			LineExtensionAmount: &UBLAmount{CurrencyID: inv.Currency, Value: AmountEUR(line.LineTotal)},
			Item: &UBLItem{
				Name:        line.Description,
				Description: line.DetailedDescription,
				ClassifiedTaxCategory: &UBLClassifiedTaxCategory{
					ID:        line.TaxCategory,
					Percent:   line.TaxPercent,
					TaxScheme: &UBLTaxScheme{ID: "VAT"},
				},
			},
			Price: &UBLPrice{
				PriceAmount: &UBLAmount{CurrencyID: inv.Currency, Value: AmountEUR(line.UnitPrice)},
			},
		}

		if line.ItemID != "" {
			ublLine.Item.SellersItemIdentification = &UBLItemIdentification{ID: line.ItemID}
		}
		if line.GTIN != "" {
			ublLine.Item.StandardItemIdentification = &UBLItemIdentification{ID: line.GTIN}
		}

		ubl.InvoiceLines = append(ubl.InvoiceLines, ublLine)
	}

	// Marshal to XML
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(ubl); err != nil {
		return nil, fmt.Errorf("failed to encode XML: %w", err)
	}

	return buf.Bytes(), nil
}

// convertPartyToUBL converts an InvoiceParty to UBL format
func convertPartyToUBL(party *InvoiceParty) *UBLParty {
	ublParty := &UBLParty{
		Party: &UBLPartyDetails{
			PartyName: &UBLPartyName{Name: party.Name},
			PostalAddress: &UBLPostalAddress{
				StreetName:           party.Street,
				AdditionalStreetName: party.AdditionalStreet,
				CityName:             party.City,
				PostalZone:           party.PostalCode,
				Country:              &UBLCountry{IdentificationCode: party.Country},
			},
		},
	}

	if party.ID != "" {
		ublParty.Party.PartyIdentification = &UBLPartyIdentification{ID: party.ID}
	}

	if party.VATNumber != "" {
		ublParty.Party.PartyTaxScheme = &UBLPartyTaxScheme{
			CompanyID: party.VATNumber,
			TaxScheme: &UBLTaxScheme{ID: "VAT"},
		}
	}

	if party.ContactName != "" || party.ContactPhone != "" || party.Email != "" {
		ublParty.Party.Contact = &UBLContact{
			Name:           party.ContactName,
			Telephone:      party.ContactPhone,
			ElectronicMail: party.Email,
		}
	}

	return ublParty
}
