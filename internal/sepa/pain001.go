package sepa

import (
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

const (
	Pain001NS = "urn:iso:std:iso:20022:tech:xsd:pain.001.001.03"
)

// Pain001Document represents a pain.001 XML document
type Pain001Document struct {
	XMLName xml.Name           `xml:"Document"`
	XMLNS   string             `xml:"xmlns,attr"`
	CstmrCdtTrfInitn *Pain001CstmrCdtTrfInitn `xml:"CstmrCdtTrfInitn"`
}

// Pain001CstmrCdtTrfInitn is the customer credit transfer initiation
type Pain001CstmrCdtTrfInitn struct {
	GrpHdr *Pain001GrpHdr     `xml:"GrpHdr"`
	PmtInf []*Pain001PmtInf   `xml:"PmtInf"`
}

// Pain001GrpHdr is the group header
type Pain001GrpHdr struct {
	MsgId    string         `xml:"MsgId"`
	CreDtTm  string         `xml:"CreDtTm"`
	NbOfTxs  int            `xml:"NbOfTxs"`
	CtrlSum  float64        `xml:"CtrlSum"`
	InitgPty *Pain001Party  `xml:"InitgPty"`
}

// Pain001Party represents a party
type Pain001Party struct {
	Nm      string           `xml:"Nm,omitempty"`
	Id      *Pain001PartyId  `xml:"Id,omitempty"`
	PstlAdr *Pain001Address  `xml:"PstlAdr,omitempty"`
}

// Pain001PartyId represents party identification
type Pain001PartyId struct {
	OrgId *Pain001OrgId `xml:"OrgId,omitempty"`
}

// Pain001OrgId represents organization ID
type Pain001OrgId struct {
	Othr *Pain001OthrId `xml:"Othr,omitempty"`
}

// Pain001OthrId represents other ID
type Pain001OthrId struct {
	Id string `xml:"Id"`
}

// Pain001Address represents a postal address
type Pain001Address struct {
	StrtNm  string `xml:"StrtNm,omitempty"`
	BldgNb  string `xml:"BldgNb,omitempty"`
	PstCd   string `xml:"PstCd,omitempty"`
	TwnNm   string `xml:"TwnNm,omitempty"`
	Ctry    string `xml:"Ctry,omitempty"`
}

// Pain001PmtInf represents payment information
type Pain001PmtInf struct {
	PmtInfId    string                `xml:"PmtInfId"`
	PmtMtd      string                `xml:"PmtMtd"` // TRF for transfers
	BtchBookg   bool                  `xml:"BtchBookg,omitempty"`
	NbOfTxs     int                   `xml:"NbOfTxs"`
	CtrlSum     float64               `xml:"CtrlSum"`
	PmtTpInf    *Pain001PmtTpInf      `xml:"PmtTpInf,omitempty"`
	ReqdExctnDt string                `xml:"ReqdExctnDt"`
	Dbtr        *Pain001Party         `xml:"Dbtr"`
	DbtrAcct    *Pain001Account       `xml:"DbtrAcct"`
	DbtrAgt     *Pain001FinInstn      `xml:"DbtrAgt"`
	CdtTrfTxInf []*Pain001CdtTrfTxInf `xml:"CdtTrfTxInf"`
}

// Pain001PmtTpInf represents payment type information
type Pain001PmtTpInf struct {
	SvcLvl *Pain001SvcLvl `xml:"SvcLvl,omitempty"`
}

// Pain001SvcLvl represents service level
type Pain001SvcLvl struct {
	Cd string `xml:"Cd"`
}

// Pain001Account represents an account
type Pain001Account struct {
	Id  *Pain001AccountId `xml:"Id"`
	Ccy string            `xml:"Ccy,omitempty"`
}

// Pain001AccountId represents account identification
type Pain001AccountId struct {
	IBAN string `xml:"IBAN"`
}

// Pain001FinInstn represents a financial institution
type Pain001FinInstn struct {
	FinInstnId *Pain001FinInstnId `xml:"FinInstnId"`
}

// Pain001FinInstnId represents financial institution identification
type Pain001FinInstnId struct {
	BIC string `xml:"BIC,omitempty"`
}

// Pain001CdtTrfTxInf represents a credit transfer transaction
type Pain001CdtTrfTxInf struct {
	PmtId       *Pain001PmtId    `xml:"PmtId"`
	Amt         *Pain001Amt      `xml:"Amt"`
	CdtrAgt     *Pain001FinInstn `xml:"CdtrAgt,omitempty"`
	Cdtr        *Pain001Party    `xml:"Cdtr"`
	CdtrAcct    *Pain001Account  `xml:"CdtrAcct"`
	RmtInf      *Pain001RmtInf   `xml:"RmtInf,omitempty"`
}

// Pain001PmtId represents payment identification
type Pain001PmtId struct {
	InstrId    string `xml:"InstrId,omitempty"`
	EndToEndId string `xml:"EndToEndId"`
}

// Pain001Amt represents an amount
type Pain001Amt struct {
	InstdAmt *Pain001InstdAmt `xml:"InstdAmt"`
}

// Pain001InstdAmt represents an instructed amount
type Pain001InstdAmt struct {
	Ccy   string  `xml:"Ccy,attr"`
	Value float64 `xml:",chardata"`
}

// Pain001RmtInf represents remittance information
type Pain001RmtInf struct {
	Ustrd string `xml:"Ustrd,omitempty"`
}

// GeneratePain001 generates a pain.001 XML document from a credit transfer
func GeneratePain001(ct *SEPACreditTransfer) ([]byte, error) {
	// Calculate totals if not set
	if ct.NumberOfTxs == 0 {
		ct.CalculateTotals()
	}

	doc := &Pain001Document{
		XMLNS: Pain001NS,
		CstmrCdtTrfInitn: &Pain001CstmrCdtTrfInitn{
			GrpHdr: &Pain001GrpHdr{
				MsgId:   ct.MessageID,
				CreDtTm: ct.CreationTime.Format("2006-01-02T15:04:05"),
				NbOfTxs: ct.NumberOfTxs,
				CtrlSum: ct.ControlSumEUR(),
				InitgPty: &Pain001Party{
					Nm: ct.InitiatingParty.Name,
				},
			},
		},
	}

	// Payment information block
	pmtInf := &Pain001PmtInf{
		PmtInfId:    ct.MessageID + "-001",
		PmtMtd:      "TRF",
		BtchBookg:   true,
		NbOfTxs:     ct.NumberOfTxs,
		CtrlSum:     ct.ControlSumEUR(),
		ReqdExctnDt: time.Now().Format("2006-01-02"),
		PmtTpInf: &Pain001PmtTpInf{
			SvcLvl: &Pain001SvcLvl{Cd: "SEPA"},
		},
		Dbtr: convertPartyToPain001(&ct.Debtor),
		DbtrAcct: &Pain001Account{
			Id: &Pain001AccountId{IBAN: ct.DebtorAccount.IBAN},
		},
		DbtrAgt: &Pain001FinInstn{
			FinInstnId: &Pain001FinInstnId{BIC: ct.DebtorAccount.BIC},
		},
	}

	// Add transactions
	for _, txn := range ct.Transactions {
		cdtTrfTxInf := &Pain001CdtTrfTxInf{
			PmtId: &Pain001PmtId{
				InstrId:    txn.InstructionID,
				EndToEndId: txn.EndToEndID,
			},
			Amt: &Pain001Amt{
				InstdAmt: &Pain001InstdAmt{
					Ccy:   txn.Currency,
					Value: txn.AmountEUR(),
				},
			},
			Cdtr: &Pain001Party{
				Nm: txn.Creditor.Name,
			},
			CdtrAcct: &Pain001Account{
				Id: &Pain001AccountId{IBAN: txn.CreditorAccount.IBAN},
			},
		}

		if txn.CreditorAccount.BIC != "" {
			cdtTrfTxInf.CdtrAgt = &Pain001FinInstn{
				FinInstnId: &Pain001FinInstnId{BIC: txn.CreditorAccount.BIC},
			}
		}

		if txn.RemittanceInfo != "" {
			cdtTrfTxInf.RmtInf = &Pain001RmtInf{
				Ustrd: txn.RemittanceInfo,
			}
		}

		pmtInf.CdtTrfTxInf = append(pmtInf.CdtTrfTxInf, cdtTrfTxInf)
	}

	doc.CstmrCdtTrfInitn.PmtInf = []*Pain001PmtInf{pmtInf}

	// Marshal to XML
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return nil, fmt.Errorf("failed to encode XML: %w", err)
	}

	return buf.Bytes(), nil
}

// convertPartyToPain001 converts a SEPAParty to Pain001Party
func convertPartyToPain001(p *SEPAParty) *Pain001Party {
	party := &Pain001Party{
		Nm: p.Name,
	}

	if p.ID != "" {
		party.Id = &Pain001PartyId{
			OrgId: &Pain001OrgId{
				Othr: &Pain001OthrId{Id: p.ID},
			},
		}
	}

	if p.Address != nil {
		party.PstlAdr = &Pain001Address{
			StrtNm: p.Address.StreetName,
			BldgNb: p.Address.BuildingNo,
			PstCd:  p.Address.PostCode,
			TwnNm:  p.Address.TownName,
			Ctry:   p.Address.Country,
		}
	}

	return party
}

// ParseCreditTransferCSV parses a CSV file into credit transfer transactions
// Expected columns: creditor_name,creditor_iban,amount,currency,reference
func ParseCreditTransferCSV(data []byte) ([]*SEPACreditTransaction, error) {
	reader := csv.NewReader(bytes.NewReader(data))

	// Read header
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	// Build column index
	colIndex := make(map[string]int)
	for i, col := range header {
		colIndex[strings.ToLower(strings.TrimSpace(col))] = i
	}

	// Required columns
	required := []string{"creditor_name", "creditor_iban", "amount"}
	for _, col := range required {
		if _, ok := colIndex[col]; !ok {
			return nil, fmt.Errorf("missing required column: %s", col)
		}
	}

	transactions := make([]*SEPACreditTransaction, 0)
	lineNum := 1

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading line %d: %w", lineNum, err)
		}
		lineNum++

		// Parse amount
		amountStr := record[colIndex["amount"]]
		amountFloat, err := strconv.ParseFloat(strings.TrimSpace(amountStr), 64)
		if err != nil {
			return nil, fmt.Errorf("invalid amount on line %d: %w", lineNum, err)
		}
		amount := int64(amountFloat * 100) // Convert to cents

		// Get currency
		currency := "EUR"
		if idx, ok := colIndex["currency"]; ok && idx < len(record) {
			if c := strings.TrimSpace(record[idx]); c != "" {
				currency = c
			}
		}

		// Get reference
		reference := ""
		if idx, ok := colIndex["reference"]; ok && idx < len(record) {
			reference = strings.TrimSpace(record[idx])
		}

		txn := &SEPACreditTransaction{
			InstructionID: fmt.Sprintf("TXN-%d", lineNum-1),
			EndToEndID:    reference,
			Amount:        amount,
			Currency:      currency,
			Creditor: SEPAParty{
				Name: strings.TrimSpace(record[colIndex["creditor_name"]]),
			},
			CreditorAccount: SEPAAccount{
				IBAN: strings.TrimSpace(record[colIndex["creditor_iban"]]),
			},
			RemittanceInfo: reference,
		}

		// Optional BIC
		if idx, ok := colIndex["creditor_bic"]; ok && idx < len(record) {
			txn.CreditorAccount.BIC = strings.TrimSpace(record[idx])
		}

		transactions = append(transactions, txn)
	}

	return transactions, nil
}
