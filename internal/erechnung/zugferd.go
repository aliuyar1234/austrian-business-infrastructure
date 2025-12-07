package erechnung

import (
	"bytes"
	"encoding/xml"
	"fmt"
)

const (
	CIINS = "urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100"
	RAMNS = "urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100"
	QDTNS = "urn:un:unece:uncefact:data:standard:QualifiedDataType:100"
	UDTNS = "urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100"
)

// CIIInvoice represents a UN/CEFACT Cross-Industry Invoice for ZUGFeRD
type CIIInvoice struct {
	XMLName xml.Name `xml:"rsm:CrossIndustryInvoice"`
	RSM     string   `xml:"xmlns:rsm,attr"`
	RAM     string   `xml:"xmlns:ram,attr"`
	QDT     string   `xml:"xmlns:qdt,attr"`
	UDT     string   `xml:"xmlns:udt,attr"`

	ExchangedDocumentContext *CIIDocumentContext `xml:"rsm:ExchangedDocumentContext"`
	ExchangedDocument        *CIIExchangedDocument `xml:"rsm:ExchangedDocument"`
	SupplyChainTradeTransaction *CIITradeTransaction `xml:"rsm:SupplyChainTradeTransaction"`
}

// CIIDocumentContext represents the document context
type CIIDocumentContext struct {
	GuidelineSpecifiedDocumentContextParameter *CIIContextParameter `xml:"ram:GuidelineSpecifiedDocumentContextParameter"`
}

// CIIContextParameter represents a context parameter
type CIIContextParameter struct {
	ID string `xml:"ram:ID"`
}

// CIIExchangedDocument represents the exchanged document header
type CIIExchangedDocument struct {
	ID       string           `xml:"ram:ID"`
	TypeCode string           `xml:"ram:TypeCode"`
	IssueDateTime *CIIDateTime `xml:"ram:IssueDateTime"`
	IncludedNote *CIINote     `xml:"ram:IncludedNote,omitempty"`
}

// CIIDateTime represents a date/time
type CIIDateTime struct {
	DateTimeString *CIIDateTimeString `xml:"udt:DateTimeString"`
}

// CIIDateTimeString represents a formatted date
type CIIDateTimeString struct {
	Format string `xml:"format,attr"`
	Value  string `xml:",chardata"`
}

// CIINote represents a note
type CIINote struct {
	Content string `xml:"ram:Content"`
}

// CIITradeTransaction represents the trade transaction
type CIITradeTransaction struct {
	ApplicableHeaderTradeAgreement  *CIIHeaderTradeAgreement  `xml:"ram:ApplicableHeaderTradeAgreement"`
	ApplicableHeaderTradeDelivery   *CIIHeaderTradeDelivery   `xml:"ram:ApplicableHeaderTradeDelivery,omitempty"`
	ApplicableHeaderTradeSettlement *CIIHeaderTradeSettlement `xml:"ram:ApplicableHeaderTradeSettlement"`
	IncludedSupplyChainTradeLineItem []*CIITradeLineItem      `xml:"ram:IncludedSupplyChainTradeLineItem"`
}

// CIIHeaderTradeAgreement represents trade agreement
type CIIHeaderTradeAgreement struct {
	BuyerReference    string          `xml:"ram:BuyerReference,omitempty"`
	SellerTradeParty  *CIITradeParty  `xml:"ram:SellerTradeParty"`
	BuyerTradeParty   *CIITradeParty  `xml:"ram:BuyerTradeParty"`
	BuyerOrderReferencedDocument *CIIReferencedDocument `xml:"ram:BuyerOrderReferencedDocument,omitempty"`
}

// CIIReferencedDocument represents a referenced document
type CIIReferencedDocument struct {
	IssuerAssignedID string `xml:"ram:IssuerAssignedID"`
}

// CIITradeParty represents a trade party
type CIITradeParty struct {
	ID                 string          `xml:"ram:ID,omitempty"`
	Name               string          `xml:"ram:Name"`
	SpecifiedTaxRegistration *CIITaxRegistration `xml:"ram:SpecifiedTaxRegistration,omitempty"`
	PostalTradeAddress *CIIPostalAddress `xml:"ram:PostalTradeAddress,omitempty"`
	URIUniversalCommunication *CIIUniversalCommunication `xml:"ram:URIUniversalCommunication,omitempty"`
}

// CIITaxRegistration represents tax registration
type CIITaxRegistration struct {
	ID *CIIIDType `xml:"ram:ID"`
}

// CIIIDType represents an ID with scheme
type CIIIDType struct {
	SchemeID string `xml:"schemeID,attr,omitempty"`
	Value    string `xml:",chardata"`
}

// CIIPostalAddress represents a postal address
type CIIPostalAddress struct {
	PostcodeCode   string `xml:"ram:PostcodeCode,omitempty"`
	LineOne        string `xml:"ram:LineOne,omitempty"`
	LineTwo        string `xml:"ram:LineTwo,omitempty"`
	CityName       string `xml:"ram:CityName,omitempty"`
	CountryID      string `xml:"ram:CountryID"`
}

// CIIUniversalCommunication represents electronic communication
type CIIUniversalCommunication struct {
	URIID *CIIIDType `xml:"ram:URIID"`
}

// CIIHeaderTradeDelivery represents trade delivery
type CIIHeaderTradeDelivery struct {
	// Minimal delivery information
}

// CIIHeaderTradeSettlement represents trade settlement
type CIIHeaderTradeSettlement struct {
	InvoiceCurrencyCode              string                       `xml:"ram:InvoiceCurrencyCode"`
	SpecifiedTradeSettlementPaymentMeans *CIIPaymentMeans          `xml:"ram:SpecifiedTradeSettlementPaymentMeans,omitempty"`
	ApplicableTradeTax               []*CIITradeTax               `xml:"ram:ApplicableTradeTax"`
	SpecifiedTradePaymentTerms       *CIIPaymentTerms             `xml:"ram:SpecifiedTradePaymentTerms,omitempty"`
	SpecifiedTradeSettlementHeaderMonetarySummation *CIIMonetarySummation `xml:"ram:SpecifiedTradeSettlementHeaderMonetarySummation"`
}

// CIIPaymentMeans represents payment means
type CIIPaymentMeans struct {
	TypeCode                    string              `xml:"ram:TypeCode"`
	PayeePartyCreditorFinancialAccount *CIIFinancialAccount `xml:"ram:PayeePartyCreditorFinancialAccount,omitempty"`
	PayeeSpecifiedCreditorFinancialInstitution *CIIFinancialInstitution `xml:"ram:PayeeSpecifiedCreditorFinancialInstitution,omitempty"`
}

// CIIFinancialAccount represents a financial account
type CIIFinancialAccount struct {
	IBANID      string `xml:"ram:IBANID,omitempty"`
	AccountName string `xml:"ram:AccountName,omitempty"`
}

// CIIFinancialInstitution represents a financial institution
type CIIFinancialInstitution struct {
	BICID string `xml:"ram:BICID"`
}

// CIITradeTax represents tax information
type CIITradeTax struct {
	CalculatedAmount     *CIIAmount `xml:"ram:CalculatedAmount"`
	TypeCode             string     `xml:"ram:TypeCode"`
	ExemptionReason      string     `xml:"ram:ExemptionReason,omitempty"`
	BasisAmount          *CIIAmount `xml:"ram:BasisAmount"`
	CategoryCode         string     `xml:"ram:CategoryCode"`
	RateApplicablePercent float64   `xml:"ram:RateApplicablePercent"`
}

// CIIAmount represents an amount
type CIIAmount struct {
	CurrencyID string  `xml:"currencyID,attr,omitempty"`
	Value      float64 `xml:",chardata"`
}

// CIIPaymentTerms represents payment terms
type CIIPaymentTerms struct {
	Description     string       `xml:"ram:Description,omitempty"`
	DueDateDateTime *CIIDateTime `xml:"ram:DueDateDateTime,omitempty"`
}

// CIIMonetarySummation represents monetary totals
type CIIMonetarySummation struct {
	LineTotalAmount     *CIIAmount `xml:"ram:LineTotalAmount"`
	TaxBasisTotalAmount *CIIAmount `xml:"ram:TaxBasisTotalAmount"`
	TaxTotalAmount      *CIIAmount `xml:"ram:TaxTotalAmount"`
	GrandTotalAmount    *CIIAmount `xml:"ram:GrandTotalAmount"`
	DuePayableAmount    *CIIAmount `xml:"ram:DuePayableAmount"`
}

// CIITradeLineItem represents an invoice line
type CIITradeLineItem struct {
	AssociatedDocumentLineDocument *CIILineDocument     `xml:"ram:AssociatedDocumentLineDocument"`
	SpecifiedTradeProduct          *CIITradeProduct     `xml:"ram:SpecifiedTradeProduct"`
	SpecifiedLineTradeAgreement    *CIILineTradeAgreement `xml:"ram:SpecifiedLineTradeAgreement"`
	SpecifiedLineTradeDelivery     *CIILineTradeDelivery  `xml:"ram:SpecifiedLineTradeDelivery"`
	SpecifiedLineTradeSettlement   *CIILineTradeSettlement `xml:"ram:SpecifiedLineTradeSettlement"`
}

// CIILineDocument represents line document
type CIILineDocument struct {
	LineID string `xml:"ram:LineID"`
}

// CIITradeProduct represents a product
type CIITradeProduct struct {
	SellerAssignedID string `xml:"ram:SellerAssignedID,omitempty"`
	GlobalID         *CIIIDType `xml:"ram:GlobalID,omitempty"`
	Name             string `xml:"ram:Name"`
	Description      string `xml:"ram:Description,omitempty"`
}

// CIILineTradeAgreement represents line trade agreement
type CIILineTradeAgreement struct {
	NetPriceProductTradePrice *CIITradePrice `xml:"ram:NetPriceProductTradePrice"`
}

// CIITradePrice represents a trade price
type CIITradePrice struct {
	ChargeAmount *CIIAmount `xml:"ram:ChargeAmount"`
}

// CIILineTradeDelivery represents line delivery
type CIILineTradeDelivery struct {
	BilledQuantity *CIIQuantity `xml:"ram:BilledQuantity"`
}

// CIIQuantity represents a quantity
type CIIQuantity struct {
	UnitCode string  `xml:"unitCode,attr"`
	Value    float64 `xml:",chardata"`
}

// CIILineTradeSettlement represents line settlement
type CIILineTradeSettlement struct {
	ApplicableTradeTax            *CIILineTradeTax `xml:"ram:ApplicableTradeTax"`
	SpecifiedTradeSettlementLineMonetarySummation *CIILineMonetarySummation `xml:"ram:SpecifiedTradeSettlementLineMonetarySummation"`
}

// CIILineTradeTax represents line tax
type CIILineTradeTax struct {
	TypeCode             string  `xml:"ram:TypeCode"`
	CategoryCode         string  `xml:"ram:CategoryCode"`
	RateApplicablePercent float64 `xml:"ram:RateApplicablePercent"`
}

// CIILineMonetarySummation represents line monetary summation
type CIILineMonetarySummation struct {
	LineTotalAmount *CIIAmount `xml:"ram:LineTotalAmount"`
}

// GenerateZUGFeRD generates ZUGFeRD (UN/CEFACT CII) XML from an invoice
func GenerateZUGFeRD(inv *Invoice) ([]byte, error) {
	// Calculate totals if not done
	if inv.TaxExclusiveAmount == 0 {
		if err := inv.CalculateTotals(); err != nil {
			return nil, fmt.Errorf("failed to calculate totals: %w", err)
		}
	}

	cii := &CIIInvoice{
		RSM: CIINS,
		RAM: RAMNS,
		QDT: QDTNS,
		UDT: UDTNS,
		ExchangedDocumentContext: &CIIDocumentContext{
			GuidelineSpecifiedDocumentContextParameter: &CIIContextParameter{
				ID: "urn:factur-x.eu:1p0:extended",
			},
		},
		ExchangedDocument: &CIIExchangedDocument{
			ID:       inv.ID,
			TypeCode: string(inv.InvoiceType),
			IssueDateTime: &CIIDateTime{
				DateTimeString: &CIIDateTimeString{
					Format: "102",
					Value:  inv.IssueDate.Format("20060102"),
				},
			},
		},
	}

	if inv.Notes != "" {
		cii.ExchangedDocument.IncludedNote = &CIINote{Content: inv.Notes}
	}

	// Trade transaction
	cii.SupplyChainTradeTransaction = &CIITradeTransaction{
		ApplicableHeaderTradeAgreement: &CIIHeaderTradeAgreement{
			BuyerReference: inv.BuyerReference,
		},
		ApplicableHeaderTradeDelivery: &CIIHeaderTradeDelivery{},
		ApplicableHeaderTradeSettlement: &CIIHeaderTradeSettlement{
			InvoiceCurrencyCode: inv.Currency,
		},
	}

	if inv.OrderReference != "" {
		cii.SupplyChainTradeTransaction.ApplicableHeaderTradeAgreement.BuyerOrderReferencedDocument = &CIIReferencedDocument{
			IssuerAssignedID: inv.OrderReference,
		}
	}

	// Seller
	if inv.Seller != nil {
		cii.SupplyChainTradeTransaction.ApplicableHeaderTradeAgreement.SellerTradeParty = convertPartyToCII(inv.Seller)
	}

	// Buyer
	if inv.Buyer != nil {
		cii.SupplyChainTradeTransaction.ApplicableHeaderTradeAgreement.BuyerTradeParty = convertPartyToCII(inv.Buyer)
	}

	// Payment means
	if inv.PaymentMeans != "" {
		cii.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.SpecifiedTradeSettlementPaymentMeans = &CIIPaymentMeans{
			TypeCode: string(inv.PaymentMeans),
		}
		if inv.BankAccount != nil {
			cii.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.SpecifiedTradeSettlementPaymentMeans.PayeePartyCreditorFinancialAccount = &CIIFinancialAccount{
				IBANID:      inv.BankAccount.IBAN,
				AccountName: inv.BankAccount.Name,
			}
			if inv.BankAccount.BIC != "" {
				cii.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.SpecifiedTradeSettlementPaymentMeans.PayeeSpecifiedCreditorFinancialInstitution = &CIIFinancialInstitution{
					BICID: inv.BankAccount.BIC,
				}
			}
		}
	}

	// Payment terms
	if inv.PaymentTerms != "" || !inv.DueDate.IsZero() {
		cii.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.SpecifiedTradePaymentTerms = &CIIPaymentTerms{
			Description: inv.PaymentTerms,
		}
		if !inv.DueDate.IsZero() {
			cii.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.SpecifiedTradePaymentTerms.DueDateDateTime = &CIIDateTime{
				DateTimeString: &CIIDateTimeString{
					Format: "102",
					Value:  inv.DueDate.Format("20060102"),
				},
			}
		}
	}

	// Tax breakdown
	for _, ts := range inv.TaxSubtotals {
		cii.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.ApplicableTradeTax = append(
			cii.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.ApplicableTradeTax,
			&CIITradeTax{
				CalculatedAmount:     &CIIAmount{Value: AmountEUR(ts.TaxAmount)},
				TypeCode:             "VAT",
				ExemptionReason:      ts.ExemptionReason,
				BasisAmount:          &CIIAmount{Value: AmountEUR(ts.TaxableAmount)},
				CategoryCode:         ts.TaxCategory,
				RateApplicablePercent: ts.TaxPercent,
			},
		)
	}

	// Monetary summation
	cii.SupplyChainTradeTransaction.ApplicableHeaderTradeSettlement.SpecifiedTradeSettlementHeaderMonetarySummation = &CIIMonetarySummation{
		LineTotalAmount:     &CIIAmount{Value: AmountEUR(inv.TaxExclusiveAmount)},
		TaxBasisTotalAmount: &CIIAmount{Value: AmountEUR(inv.TaxExclusiveAmount)},
		TaxTotalAmount:      &CIIAmount{CurrencyID: inv.Currency, Value: AmountEUR(inv.TaxAmount)},
		GrandTotalAmount:    &CIIAmount{Value: AmountEUR(inv.TaxInclusiveAmount)},
		DuePayableAmount:    &CIIAmount{Value: AmountEUR(inv.PayableAmount)},
	}

	// Invoice lines
	for _, line := range inv.Lines {
		ciiLine := &CIITradeLineItem{
			AssociatedDocumentLineDocument: &CIILineDocument{LineID: line.ID},
			SpecifiedTradeProduct: &CIITradeProduct{
				Name:        line.Description,
				Description: line.DetailedDescription,
			},
			SpecifiedLineTradeAgreement: &CIILineTradeAgreement{
				NetPriceProductTradePrice: &CIITradePrice{
					ChargeAmount: &CIIAmount{Value: AmountEUR(line.UnitPrice)},
				},
			},
			SpecifiedLineTradeDelivery: &CIILineTradeDelivery{
				BilledQuantity: &CIIQuantity{
					UnitCode: line.UnitCode,
					Value:    line.Quantity,
				},
			},
			SpecifiedLineTradeSettlement: &CIILineTradeSettlement{
				ApplicableTradeTax: &CIILineTradeTax{
					TypeCode:             "VAT",
					CategoryCode:         line.TaxCategory,
					RateApplicablePercent: line.TaxPercent,
				},
				SpecifiedTradeSettlementLineMonetarySummation: &CIILineMonetarySummation{
					LineTotalAmount: &CIIAmount{Value: AmountEUR(line.LineTotal)},
				},
			},
		}

		if line.ItemID != "" {
			ciiLine.SpecifiedTradeProduct.SellerAssignedID = line.ItemID
		}
		if line.GTIN != "" {
			ciiLine.SpecifiedTradeProduct.GlobalID = &CIIIDType{
				SchemeID: "0160",
				Value:    line.GTIN,
			}
		}

		cii.SupplyChainTradeTransaction.IncludedSupplyChainTradeLineItem = append(
			cii.SupplyChainTradeTransaction.IncludedSupplyChainTradeLineItem,
			ciiLine,
		)
	}

	// Marshal to XML
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(cii); err != nil {
		return nil, fmt.Errorf("failed to encode XML: %w", err)
	}

	return buf.Bytes(), nil
}

// convertPartyToCII converts an InvoiceParty to CII format
func convertPartyToCII(party *InvoiceParty) *CIITradeParty {
	ciiParty := &CIITradeParty{
		ID:   party.ID,
		Name: party.Name,
		PostalTradeAddress: &CIIPostalAddress{
			LineOne:      party.Street,
			LineTwo:      party.AdditionalStreet,
			CityName:     party.City,
			PostcodeCode: party.PostalCode,
			CountryID:    party.Country,
		},
	}

	if party.VATNumber != "" {
		ciiParty.SpecifiedTaxRegistration = &CIITaxRegistration{
			ID: &CIIIDType{SchemeID: "VA", Value: party.VATNumber},
		}
	}

	if party.Email != "" {
		ciiParty.URIUniversalCommunication = &CIIUniversalCommunication{
			URIID: &CIIIDType{SchemeID: "EM", Value: party.Email},
		}
	}

	return ciiParty
}
