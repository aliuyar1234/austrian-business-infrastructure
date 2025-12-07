package sepa

import (
	"time"
)

// CreditDebitIndicator indicates if an entry is a credit or debit
type CreditDebitIndicator string

const (
	CreditDebitCredit CreditDebitIndicator = "CRDT"
	CreditDebitDebit  CreditDebitIndicator = "DBIT"
)

// SequenceType indicates the type of sequence for direct debits
type SequenceType string

const (
	SequenceTypeFirst     SequenceType = "FRST" // First in series
	SequenceTypeRecurrent SequenceType = "RCUR" // Recurring
	SequenceTypeFinal     SequenceType = "FNAL" // Final in series
	SequenceTypeOneOff    SequenceType = "OOFF" // One-off
)

// SEPAParty represents a party in a SEPA transaction
type SEPAParty struct {
	Name    string       `json:"name"`
	ID      string       `json:"id,omitempty"`
	Address *SEPAAddress `json:"address,omitempty"`
}

// SEPAAddress represents a postal address
type SEPAAddress struct {
	StreetName string `json:"street_name,omitempty"`
	BuildingNo string `json:"building_no,omitempty"`
	PostCode   string `json:"post_code,omitempty"`
	TownName   string `json:"town_name,omitempty"`
	Country    string `json:"country"`
}

// SEPAAccount represents a bank account
type SEPAAccount struct {
	IBAN     string `json:"iban"`
	BIC      string `json:"bic,omitempty"`
	Name     string `json:"name,omitempty"`
	Currency string `json:"currency,omitempty"`
}

// SEPACreditTransfer represents a SEPA credit transfer batch (pain.001)
type SEPACreditTransfer struct {
	MessageID       string                    `json:"message_id"`
	CreationTime    time.Time                 `json:"creation_time"`
	NumberOfTxs     int                       `json:"number_of_txs"`
	ControlSum      int64                     `json:"control_sum"` // In cents
	InitiatingParty SEPAParty                 `json:"initiating_party"`
	Debtor          SEPAParty                 `json:"debtor"`
	DebtorAccount   SEPAAccount               `json:"debtor_account"`
	Transactions    []*SEPACreditTransaction  `json:"transactions"`
}

// NewCreditTransfer creates a new credit transfer with default values
func NewCreditTransfer(messageID, initiatorName string) *SEPACreditTransfer {
	return &SEPACreditTransfer{
		MessageID:    messageID,
		CreationTime: time.Now(),
		InitiatingParty: SEPAParty{
			Name: initiatorName,
		},
		Transactions: make([]*SEPACreditTransaction, 0),
	}
}

// ControlSumEUR returns the control sum in EUR
func (ct *SEPACreditTransfer) ControlSumEUR() float64 {
	return float64(ct.ControlSum) / 100
}

// CalculateTotals calculates NumberOfTxs and ControlSum from transactions
func (ct *SEPACreditTransfer) CalculateTotals() {
	ct.NumberOfTxs = len(ct.Transactions)
	ct.ControlSum = 0
	for _, txn := range ct.Transactions {
		ct.ControlSum += txn.Amount
	}
}

// SEPACreditTransaction represents a single credit transfer transaction
type SEPACreditTransaction struct {
	InstructionID   string      `json:"instruction_id"`
	EndToEndID      string      `json:"end_to_end_id"`
	Amount          int64       `json:"amount"` // In cents
	Currency        string      `json:"currency"`
	Creditor        SEPAParty   `json:"creditor"`
	CreditorAccount SEPAAccount `json:"creditor_account"`
	RemittanceInfo  string      `json:"remittance_info,omitempty"`
	Purpose         string      `json:"purpose,omitempty"` // Purpose code
}

// AmountEUR returns the amount in EUR
func (t *SEPACreditTransaction) AmountEUR() float64 {
	return float64(t.Amount) / 100
}

// SEPADirectDebit represents a SEPA direct debit batch (pain.008)
type SEPADirectDebit struct {
	MessageID       string                        `json:"message_id"`
	CreationTime    time.Time                     `json:"creation_time"`
	NumberOfTxs     int                           `json:"number_of_txs"`
	ControlSum      int64                         `json:"control_sum"` // In cents
	Creditor        SEPAParty                     `json:"creditor"`
	CreditorAccount SEPAAccount                   `json:"creditor_account"`
	CreditorID      string                        `json:"creditor_id"` // SEPA Creditor Identifier
	Transactions    []*SEPADirectDebitTransaction `json:"transactions"`
}

// NewDirectDebit creates a new direct debit with default values
func NewDirectDebit(messageID string, creditor SEPAParty) *SEPADirectDebit {
	return &SEPADirectDebit{
		MessageID:    messageID,
		CreationTime: time.Now(),
		Creditor:     creditor,
		Transactions: make([]*SEPADirectDebitTransaction, 0),
	}
}

// CalculateTotals calculates NumberOfTxs and ControlSum from transactions
func (dd *SEPADirectDebit) CalculateTotals() {
	dd.NumberOfTxs = len(dd.Transactions)
	dd.ControlSum = 0
	for _, txn := range dd.Transactions {
		dd.ControlSum += txn.Amount
	}
}

// SEPADirectDebitTransaction represents a single direct debit transaction
type SEPADirectDebitTransaction struct {
	InstructionID  string       `json:"instruction_id"`
	EndToEndID     string       `json:"end_to_end_id"`
	Amount         int64        `json:"amount"` // In cents
	Currency       string       `json:"currency"`
	Debtor         SEPAParty    `json:"debtor"`
	DebtorAccount  SEPAAccount  `json:"debtor_account"`
	MandateID      string       `json:"mandate_id"`
	MandateDate    time.Time    `json:"mandate_date"`
	SequenceType   SequenceType `json:"sequence_type"`
	RemittanceInfo string       `json:"remittance_info,omitempty"`
}

// AmountEUR returns the amount in EUR
func (t *SEPADirectDebitTransaction) AmountEUR() float64 {
	return float64(t.Amount) / 100
}

// SEPAStatement represents a bank statement (camt.053)
type SEPAStatement struct {
	ID              string                 `json:"id"`
	CreationTime    time.Time              `json:"creation_time"`
	Account         SEPAAccount            `json:"account"`
	OpeningBalance  int64                  `json:"opening_balance"` // In cents
	ClosingBalance  int64                  `json:"closing_balance"` // In cents
	Entries         []*SEPAStatementEntry  `json:"entries"`
}

// SEPAStatementEntry represents a single statement entry
type SEPAStatementEntry struct {
	Amount          int64                `json:"amount"` // In cents
	Currency        string               `json:"currency"`
	CreditDebit     CreditDebitIndicator `json:"credit_debit"`
	BookingDate     time.Time            `json:"booking_date"`
	ValueDate       time.Time            `json:"value_date,omitempty"`
	Reference       string               `json:"reference,omitempty"`
	EndToEndID      string               `json:"end_to_end_id,omitempty"`
	RemittanceInfo  string               `json:"remittance_info,omitempty"`
	CounterpartyName string              `json:"counterparty_name,omitempty"`
	CounterpartyIBAN string              `json:"counterparty_iban,omitempty"`
}

// AmountEUR returns the amount in EUR
func (e *SEPAStatementEntry) AmountEUR() float64 {
	return float64(e.Amount) / 100
}

// IsCredit returns true if this is a credit entry
func (e *SEPAStatementEntry) IsCredit() bool {
	return e.CreditDebit == CreditDebitCredit
}

// IsDebit returns true if this is a debit entry
func (e *SEPAStatementEntry) IsDebit() bool {
	return e.CreditDebit == CreditDebitDebit
}
