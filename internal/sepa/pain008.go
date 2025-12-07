package sepa

import (
	"bytes"
	"encoding/xml"
	"fmt"
)

const (
	Pain008NS = "urn:iso:std:iso:20022:tech:xsd:pain.008.001.02"
)

// Pain008Document represents a pain.008 XML document
type Pain008Document struct {
	XMLName xml.Name               `xml:"Document"`
	XMLNS   string                 `xml:"xmlns,attr"`
	CstmrDrctDbtInitn *Pain008CstmrDrctDbtInitn `xml:"CstmrDrctDbtInitn"`
}

// Pain008CstmrDrctDbtInitn is the customer direct debit initiation
type Pain008CstmrDrctDbtInitn struct {
	GrpHdr *Pain008GrpHdr   `xml:"GrpHdr"`
	PmtInf []*Pain008PmtInf `xml:"PmtInf"`
}

// Pain008GrpHdr is the group header
type Pain008GrpHdr struct {
	MsgId    string        `xml:"MsgId"`
	CreDtTm  string        `xml:"CreDtTm"`
	NbOfTxs  int           `xml:"NbOfTxs"`
	CtrlSum  float64       `xml:"CtrlSum"`
	InitgPty *Pain008Party `xml:"InitgPty"`
}

// Pain008Party represents a party
type Pain008Party struct {
	Nm      string           `xml:"Nm,omitempty"`
	Id      *Pain008PartyId  `xml:"Id,omitempty"`
	PstlAdr *Pain008Address  `xml:"PstlAdr,omitempty"`
}

// Pain008PartyId represents party identification
type Pain008PartyId struct {
	OrgId  *Pain008OrgId  `xml:"OrgId,omitempty"`
	PrvtId *Pain008PrvtId `xml:"PrvtId,omitempty"`
}

// Pain008OrgId represents organization ID
type Pain008OrgId struct {
	Othr *Pain008OthrId `xml:"Othr,omitempty"`
}

// Pain008PrvtId represents private ID
type Pain008PrvtId struct {
	Othr *Pain008OthrId `xml:"Othr,omitempty"`
}

// Pain008OthrId represents other ID
type Pain008OthrId struct {
	Id      string          `xml:"Id"`
	SchmeNm *Pain008SchmeNm `xml:"SchmeNm,omitempty"`
}

// Pain008SchmeNm represents scheme name
type Pain008SchmeNm struct {
	Prtry string `xml:"Prtry,omitempty"`
}

// Pain008Address represents a postal address
type Pain008Address struct {
	StrtNm string `xml:"StrtNm,omitempty"`
	BldgNb string `xml:"BldgNb,omitempty"`
	PstCd  string `xml:"PstCd,omitempty"`
	TwnNm  string `xml:"TwnNm,omitempty"`
	Ctry   string `xml:"Ctry,omitempty"`
}

// Pain008PmtInf represents payment information
type Pain008PmtInf struct {
	PmtInfId    string                `xml:"PmtInfId"`
	PmtMtd      string                `xml:"PmtMtd"` // DD for direct debit
	BtchBookg   bool                  `xml:"BtchBookg,omitempty"`
	NbOfTxs     int                   `xml:"NbOfTxs"`
	CtrlSum     float64               `xml:"CtrlSum"`
	PmtTpInf    *Pain008PmtTpInf      `xml:"PmtTpInf"`
	ReqdColltnDt string               `xml:"ReqdColltnDt"`
	Cdtr        *Pain008Party         `xml:"Cdtr"`
	CdtrAcct    *Pain008Account       `xml:"CdtrAcct"`
	CdtrAgt     *Pain008FinInstn      `xml:"CdtrAgt"`
	CdtrSchmeId *Pain008CdtrSchmeId   `xml:"CdtrSchmeId,omitempty"`
	DrctDbtTxInf []*Pain008DrctDbtTxInf `xml:"DrctDbtTxInf"`
}

// Pain008PmtTpInf represents payment type information
type Pain008PmtTpInf struct {
	SvcLvl  *Pain008SvcLvl  `xml:"SvcLvl"`
	LclInstrm *Pain008LclInstrm `xml:"LclInstrm,omitempty"`
	SeqTp   string          `xml:"SeqTp"`
}

// Pain008SvcLvl represents service level
type Pain008SvcLvl struct {
	Cd string `xml:"Cd"`
}

// Pain008LclInstrm represents local instrument
type Pain008LclInstrm struct {
	Cd string `xml:"Cd"`
}

// Pain008Account represents an account
type Pain008Account struct {
	Id  *Pain008AccountId `xml:"Id"`
	Ccy string            `xml:"Ccy,omitempty"`
}

// Pain008AccountId represents account identification
type Pain008AccountId struct {
	IBAN string `xml:"IBAN"`
}

// Pain008FinInstn represents a financial institution
type Pain008FinInstn struct {
	FinInstnId *Pain008FinInstnId `xml:"FinInstnId"`
}

// Pain008FinInstnId represents financial institution identification
type Pain008FinInstnId struct {
	BIC string `xml:"BIC,omitempty"`
}

// Pain008CdtrSchmeId represents creditor scheme identification
type Pain008CdtrSchmeId struct {
	Id *Pain008CdtrSchmeIdId `xml:"Id"`
}

// Pain008CdtrSchmeIdId represents creditor scheme ID inner element
type Pain008CdtrSchmeIdId struct {
	PrvtId *Pain008CdtrPrvtId `xml:"PrvtId"`
}

// Pain008CdtrPrvtId represents creditor private ID
type Pain008CdtrPrvtId struct {
	Othr *Pain008CdtrOthrId `xml:"Othr"`
}

// Pain008CdtrOthrId represents creditor other ID
type Pain008CdtrOthrId struct {
	Id      string          `xml:"Id"`
	SchmeNm *Pain008SchmeNm `xml:"SchmeNm"`
}

// Pain008DrctDbtTxInf represents a direct debit transaction
type Pain008DrctDbtTxInf struct {
	PmtId       *Pain008PmtId       `xml:"PmtId"`
	InstdAmt    *Pain008InstdAmt    `xml:"InstdAmt"`
	DrctDbtTx   *Pain008DrctDbtTx   `xml:"DrctDbtTx"`
	DbtrAgt     *Pain008FinInstn    `xml:"DbtrAgt,omitempty"`
	Dbtr        *Pain008Party       `xml:"Dbtr"`
	DbtrAcct    *Pain008Account     `xml:"DbtrAcct"`
	RmtInf      *Pain008RmtInf      `xml:"RmtInf,omitempty"`
}

// Pain008PmtId represents payment identification
type Pain008PmtId struct {
	InstrId    string `xml:"InstrId,omitempty"`
	EndToEndId string `xml:"EndToEndId"`
}

// Pain008InstdAmt represents an instructed amount
type Pain008InstdAmt struct {
	Ccy   string  `xml:"Ccy,attr"`
	Value float64 `xml:",chardata"`
}

// Pain008DrctDbtTx represents direct debit transaction details
type Pain008DrctDbtTx struct {
	MndtRltdInf *Pain008MndtRltdInf `xml:"MndtRltdInf"`
}

// Pain008MndtRltdInf represents mandate-related information
type Pain008MndtRltdInf struct {
	MndtId    string `xml:"MndtId"`
	DtOfSgntr string `xml:"DtOfSgntr"`
}

// Pain008RmtInf represents remittance information
type Pain008RmtInf struct {
	Ustrd string `xml:"Ustrd,omitempty"`
}

// GeneratePain008 generates a pain.008 XML document from a direct debit
func GeneratePain008(dd *SEPADirectDebit) ([]byte, error) {
	// Calculate totals if not set
	if dd.NumberOfTxs == 0 {
		dd.CalculateTotals()
	}

	doc := &Pain008Document{
		XMLNS: Pain008NS,
		CstmrDrctDbtInitn: &Pain008CstmrDrctDbtInitn{
			GrpHdr: &Pain008GrpHdr{
				MsgId:   dd.MessageID,
				CreDtTm: dd.CreationTime.Format("2006-01-02T15:04:05"),
				NbOfTxs: dd.NumberOfTxs,
				CtrlSum: float64(dd.ControlSum) / 100,
				InitgPty: &Pain008Party{
					Nm: dd.Creditor.Name,
				},
			},
		},
	}

	// Determine sequence type from first transaction (or default)
	seqType := string(SequenceTypeRecurrent)
	if len(dd.Transactions) > 0 {
		seqType = string(dd.Transactions[0].SequenceType)
	}

	// Payment information block
	pmtInf := &Pain008PmtInf{
		PmtInfId:     dd.MessageID + "-001",
		PmtMtd:       "DD",
		BtchBookg:    true,
		NbOfTxs:      dd.NumberOfTxs,
		CtrlSum:      float64(dd.ControlSum) / 100,
		ReqdColltnDt: dd.CreationTime.AddDate(0, 0, 5).Format("2006-01-02"), // 5 days in future
		PmtTpInf: &Pain008PmtTpInf{
			SvcLvl: &Pain008SvcLvl{Cd: "SEPA"},
			LclInstrm: &Pain008LclInstrm{Cd: "CORE"},
			SeqTp:  seqType,
		},
		Cdtr: &Pain008Party{
			Nm: dd.Creditor.Name,
		},
		CdtrAcct: &Pain008Account{
			Id: &Pain008AccountId{IBAN: dd.CreditorAccount.IBAN},
		},
		CdtrAgt: &Pain008FinInstn{
			FinInstnId: &Pain008FinInstnId{BIC: dd.CreditorAccount.BIC},
		},
	}

	// Add creditor scheme ID if available
	if dd.CreditorID != "" {
		pmtInf.CdtrSchmeId = &Pain008CdtrSchmeId{
			Id: &Pain008CdtrSchmeIdId{
				PrvtId: &Pain008CdtrPrvtId{
					Othr: &Pain008CdtrOthrId{
						Id: dd.CreditorID,
						SchmeNm: &Pain008SchmeNm{
							Prtry: "SEPA",
						},
					},
				},
			},
		}
	}

	// Add transactions
	for _, txn := range dd.Transactions {
		drctDbtTxInf := &Pain008DrctDbtTxInf{
			PmtId: &Pain008PmtId{
				InstrId:    txn.InstructionID,
				EndToEndId: txn.EndToEndID,
			},
			InstdAmt: &Pain008InstdAmt{
				Ccy:   txn.Currency,
				Value: txn.AmountEUR(),
			},
			DrctDbtTx: &Pain008DrctDbtTx{
				MndtRltdInf: &Pain008MndtRltdInf{
					MndtId:    txn.MandateID,
					DtOfSgntr: txn.MandateDate.Format("2006-01-02"),
				},
			},
			Dbtr: &Pain008Party{
				Nm: txn.Debtor.Name,
			},
			DbtrAcct: &Pain008Account{
				Id: &Pain008AccountId{IBAN: txn.DebtorAccount.IBAN},
			},
		}

		if txn.DebtorAccount.BIC != "" {
			drctDbtTxInf.DbtrAgt = &Pain008FinInstn{
				FinInstnId: &Pain008FinInstnId{BIC: txn.DebtorAccount.BIC},
			}
		}

		if txn.RemittanceInfo != "" {
			drctDbtTxInf.RmtInf = &Pain008RmtInf{
				Ustrd: txn.RemittanceInfo,
			}
		}

		pmtInf.DrctDbtTxInf = append(pmtInf.DrctDbtTxInf, drctDbtTxInf)
	}

	doc.CstmrDrctDbtInitn.PmtInf = []*Pain008PmtInf{pmtInf}

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
