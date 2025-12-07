package sepa

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Camt053Document represents a camt.053 bank statement document
type Camt053Document struct {
	XMLName       xml.Name        `xml:"Document"`
	BkToCstmrStmt *Camt053BkToStmt `xml:"BkToCstmrStmt"`
}

// Camt053BkToStmt represents the bank to customer statement
type Camt053BkToStmt struct {
	Stmt []*Camt053Stmt `xml:"Stmt"`
}

// Camt053Stmt represents a single statement
type Camt053Stmt struct {
	Id      string          `xml:"Id"`
	CreDtTm string          `xml:"CreDtTm"`
	Acct    *Camt053Acct    `xml:"Acct"`
	Bal     []*Camt053Bal   `xml:"Bal"`
	Ntry    []*Camt053Ntry  `xml:"Ntry"`
}

// Camt053Acct represents the account
type Camt053Acct struct {
	Id  *Camt053AcctId `xml:"Id"`
	Ccy string         `xml:"Ccy"`
}

// Camt053AcctId represents account identification
type Camt053AcctId struct {
	IBAN string `xml:"IBAN"`
}

// Camt053Bal represents a balance
type Camt053Bal struct {
	Tp        *Camt053BalTp   `xml:"Tp"`
	Amt       *Camt053BalAmt  `xml:"Amt"`
	CdtDbtInd string          `xml:"CdtDbtInd"`
	Dt        *Camt053BalDt   `xml:"Dt"`
}

// Camt053BalTp represents balance type
type Camt053BalTp struct {
	CdOrPrtry *Camt053CdOrPrtry `xml:"CdOrPrtry"`
}

// Camt053CdOrPrtry represents code or proprietary
type Camt053CdOrPrtry struct {
	Cd string `xml:"Cd"`
}

// Camt053BalAmt represents balance amount
type Camt053BalAmt struct {
	Ccy   string `xml:"Ccy,attr"`
	Value string `xml:",chardata"`
}

// Camt053BalDt represents balance date
type Camt053BalDt struct {
	Dt string `xml:"Dt"`
}

// Camt053Ntry represents a statement entry
type Camt053Ntry struct {
	Amt       *Camt053NtryAmt   `xml:"Amt"`
	CdtDbtInd string            `xml:"CdtDbtInd"`
	BookgDt   *Camt053BookgDt   `xml:"BookgDt"`
	ValDt     *Camt053BookgDt   `xml:"ValDt,omitempty"`
	NtryDtls  *Camt053NtryDtls  `xml:"NtryDtls,omitempty"`
}

// Camt053NtryAmt represents entry amount
type Camt053NtryAmt struct {
	Ccy   string `xml:"Ccy,attr"`
	Value string `xml:",chardata"`
}

// Camt053BookgDt represents booking date
type Camt053BookgDt struct {
	Dt string `xml:"Dt"`
}

// Camt053NtryDtls represents entry details
type Camt053NtryDtls struct {
	TxDtls *Camt053TxDtls `xml:"TxDtls,omitempty"`
}

// Camt053TxDtls represents transaction details
type Camt053TxDtls struct {
	Refs   *Camt053Refs   `xml:"Refs,omitempty"`
	RmtInf *Camt053RmtInf `xml:"RmtInf,omitempty"`
	RltdPties *Camt053RltdPties `xml:"RltdPties,omitempty"`
}

// Camt053Refs represents references
type Camt053Refs struct {
	EndToEndId string `xml:"EndToEndId,omitempty"`
	TxId       string `xml:"TxId,omitempty"`
}

// Camt053RmtInf represents remittance information
type Camt053RmtInf struct {
	Ustrd string `xml:"Ustrd,omitempty"`
}

// Camt053RltdPties represents related parties
type Camt053RltdPties struct {
	Dbtr    *Camt053RltdPty `xml:"Dbtr,omitempty"`
	DbtrAcct *Camt053AcctId `xml:"DbtrAcct,omitempty"`
	Cdtr    *Camt053RltdPty `xml:"Cdtr,omitempty"`
	CdtrAcct *Camt053AcctId `xml:"CdtrAcct,omitempty"`
}

// Camt053RltdPty represents a related party
type Camt053RltdPty struct {
	Nm string `xml:"Nm,omitempty"`
}

// ParseCamt053 parses a camt.053 XML document into a statement
func ParseCamt053(data []byte) (*SEPAStatement, error) {
	var doc Camt053Document
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse camt.053: %w", err)
	}

	if doc.BkToCstmrStmt == nil || len(doc.BkToCstmrStmt.Stmt) == 0 {
		return nil, fmt.Errorf("no statement found in document")
	}

	// Parse first statement
	stmt := doc.BkToCstmrStmt.Stmt[0]

	result := &SEPAStatement{
		ID: stmt.Id,
	}

	// Parse creation time
	if stmt.CreDtTm != "" {
		if t, err := time.Parse("2006-01-02T15:04:05", stmt.CreDtTm); err == nil {
			result.CreationTime = t
		}
	}

	// Parse account
	if stmt.Acct != nil {
		result.Account = SEPAAccount{
			Currency: stmt.Acct.Ccy,
		}
		if stmt.Acct.Id != nil {
			result.Account.IBAN = stmt.Acct.Id.IBAN
		}
	}

	// Parse balances
	for _, bal := range stmt.Bal {
		if bal.Tp != nil && bal.Tp.CdOrPrtry != nil {
			amount := parseAmount(bal.Amt)
			if bal.CdtDbtInd == "DBIT" {
				amount = -amount
			}

			switch bal.Tp.CdOrPrtry.Cd {
			case "OPBD": // Opening balance
				result.OpeningBalance = amount
			case "CLBD": // Closing balance
				result.ClosingBalance = amount
			}
		}
	}

	// Parse entries
	for _, ntry := range stmt.Ntry {
		entry := &SEPAStatementEntry{
			Amount:   parseAmount(ntry.Amt),
			Currency: "EUR",
		}

		if ntry.Amt != nil {
			entry.Currency = ntry.Amt.Ccy
		}

		// Credit/Debit indicator
		if ntry.CdtDbtInd == "CRDT" {
			entry.CreditDebit = CreditDebitCredit
		} else {
			entry.CreditDebit = CreditDebitDebit
		}

		// Booking date
		if ntry.BookgDt != nil && ntry.BookgDt.Dt != "" {
			if t, err := time.Parse("2006-01-02", ntry.BookgDt.Dt); err == nil {
				entry.BookingDate = t
			}
		}

		// Value date
		if ntry.ValDt != nil && ntry.ValDt.Dt != "" {
			if t, err := time.Parse("2006-01-02", ntry.ValDt.Dt); err == nil {
				entry.ValueDate = t
			}
		}

		// Transaction details
		if ntry.NtryDtls != nil && ntry.NtryDtls.TxDtls != nil {
			txDtls := ntry.NtryDtls.TxDtls

			if txDtls.Refs != nil {
				entry.EndToEndID = txDtls.Refs.EndToEndId
				entry.Reference = txDtls.Refs.TxId
			}

			if txDtls.RmtInf != nil {
				entry.RemittanceInfo = txDtls.RmtInf.Ustrd
			}

			if txDtls.RltdPties != nil {
				// For credits, get debtor info
				if entry.IsCredit() && txDtls.RltdPties.Dbtr != nil {
					entry.CounterpartyName = txDtls.RltdPties.Dbtr.Nm
				}
				if entry.IsCredit() && txDtls.RltdPties.DbtrAcct != nil {
					entry.CounterpartyIBAN = txDtls.RltdPties.DbtrAcct.IBAN
				}
				// For debits, get creditor info
				if entry.IsDebit() && txDtls.RltdPties.Cdtr != nil {
					entry.CounterpartyName = txDtls.RltdPties.Cdtr.Nm
				}
				if entry.IsDebit() && txDtls.RltdPties.CdtrAcct != nil {
					entry.CounterpartyIBAN = txDtls.RltdPties.CdtrAcct.IBAN
				}
			}
		}

		result.Entries = append(result.Entries, entry)
	}

	return result, nil
}

// parseAmount parses an amount from camt.053 format
func parseAmount(amt interface{}) int64 {
	switch a := amt.(type) {
	case *Camt053NtryAmt:
		if a == nil {
			return 0
		}
		return parseAmountString(a.Value)
	case *Camt053BalAmt:
		if a == nil {
			return 0
		}
		return parseAmountString(a.Value)
	default:
		return 0
	}
}

// parseAmountString parses an amount string to cents
func parseAmountString(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	// Handle decimal amounts
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}

	return int64(f * 100)
}
