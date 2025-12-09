# Contracts: SEPA - Banking Integration

**Module**: sepa
**Date**: 2025-12-07

## 1. Overview

The SEPA module generates ISO 20022 compliant XML files for banking integration:

- **pain.001**: Customer Credit Transfer Initiation (outgoing payments)
- **pain.008**: Customer Direct Debit Initiation (incoming collections)
- **camt.053**: Bank to Customer Statement (account statements)
- **camt.054**: Bank to Customer Debit/Credit Notification

**Important**: From November 2025, structured addresses are mandatory.

---

## 2. pain.001 - Credit Transfer

### 2.1 Full Structure

```xml
<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:pain.001.001.09"
          xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
    <CstmrCdtTrfInitn>
        <!-- Group Header -->
        <GrpHdr>
            <MsgId>MSG-2025-001234</MsgId>
            <CreDtTm>2025-01-15T10:30:00</CreDtTm>
            <NbOfTxs>2</NbOfTxs>
            <CtrlSum>1500.00</CtrlSum>
            <InitgPty>
                <Nm>Auftraggeber GmbH</Nm>
            </InitgPty>
        </GrpHdr>

        <!-- Payment Information -->
        <PmtInf>
            <PmtInfId>PMT-2025-001234</PmtInfId>
            <PmtMtd>TRF</PmtMtd>
            <BtchBookg>true</BtchBookg>
            <NbOfTxs>2</NbOfTxs>
            <CtrlSum>1500.00</CtrlSum>
            <ReqdExctnDt>
                <Dt>2025-01-20</Dt>
            </ReqdExctnDt>

            <!-- Debtor (Sender) -->
            <Dbtr>
                <Nm>Auftraggeber GmbH</Nm>
                <PstlAdr>
                    <StrtNm>Musterstraße</StrtNm>
                    <BldgNb>1</BldgNb>
                    <PstCd>1010</PstCd>
                    <TwnNm>Wien</TwnNm>
                    <Ctry>AT</Ctry>
                </PstlAdr>
            </Dbtr>
            <DbtrAcct>
                <Id>
                    <IBAN>AT611904300234573201</IBAN>
                </Id>
            </DbtrAcct>
            <DbtrAgt>
                <FinInstnId>
                    <BICFI>BKAUATWW</BICFI>
                </FinInstnId>
            </DbtrAgt>

            <!-- Transaction 1 -->
            <CdtTrfTxInf>
                <PmtId>
                    <EndToEndId>E2E-001</EndToEndId>
                </PmtId>
                <Amt>
                    <InstdAmt Ccy="EUR">1000.00</InstdAmt>
                </Amt>
                <Cdtr>
                    <Nm>Empfänger 1 GmbH</Nm>
                    <PstlAdr>
                        <StrtNm>Empfängerweg</StrtNm>
                        <BldgNb>5</BldgNb>
                        <PstCd>8010</PstCd>
                        <TwnNm>Graz</TwnNm>
                        <Ctry>AT</Ctry>
                    </PstlAdr>
                </Cdtr>
                <CdtrAcct>
                    <Id>
                        <IBAN>AT021100000012345678</IBAN>
                    </Id>
                </CdtrAcct>
                <RmtInf>
                    <Ustrd>Rechnung RE-2025-001</Ustrd>
                </RmtInf>
            </CdtTrfTxInf>

            <!-- Transaction 2 -->
            <CdtTrfTxInf>
                <PmtId>
                    <EndToEndId>E2E-002</EndToEndId>
                </PmtId>
                <Amt>
                    <InstdAmt Ccy="EUR">500.00</InstdAmt>
                </Amt>
                <Cdtr>
                    <Nm>Empfänger 2 AG</Nm>
                    <PstlAdr>
                        <Ctry>AT</Ctry>
                    </PstlAdr>
                </Cdtr>
                <CdtrAcct>
                    <Id>
                        <IBAN>AT301200000098765432</IBAN>
                    </Id>
                </CdtrAcct>
                <RmtInf>
                    <Ustrd>Rechnung RE-2025-002</Ustrd>
                </RmtInf>
            </CdtTrfTxInf>
        </PmtInf>
    </CstmrCdtTrfInitn>
</Document>
```

### 2.2 Key Elements

| Element | Required | Description |
|---------|----------|-------------|
| MsgId | Yes | Unique message identifier |
| CreDtTm | Yes | Creation date/time (ISO 8601) |
| NbOfTxs | Yes | Number of transactions |
| CtrlSum | Yes | Control sum (total amount) |
| PmtMtd | Yes | "TRF" for credit transfer |
| ReqdExctnDt | Yes | Requested execution date |
| EndToEndId | Yes | End-to-end identifier |
| InstdAmt | Yes | Amount with currency |
| IBAN | Yes | Account identifier |
| Ustrd | No | Unstructured remittance info (max 140 chars) |

---

## 3. pain.008 - Direct Debit

### 3.1 Structure

```xml
<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:pain.008.001.08">
    <CstmrDrctDbtInitn>
        <GrpHdr>
            <MsgId>DD-2025-001234</MsgId>
            <CreDtTm>2025-01-15T10:30:00</CreDtTm>
            <NbOfTxs>1</NbOfTxs>
            <CtrlSum>100.00</CtrlSum>
            <InitgPty>
                <Nm>Creditor GmbH</Nm>
            </InitgPty>
        </GrpHdr>

        <PmtInf>
            <PmtInfId>DD-PMT-001</PmtInfId>
            <PmtMtd>DD</PmtMtd>
            <NbOfTxs>1</NbOfTxs>
            <CtrlSum>100.00</CtrlSum>

            <!-- Payment Type -->
            <PmtTpInf>
                <SvcLvl>
                    <Cd>SEPA</Cd>
                </SvcLvl>
                <LclInstrm>
                    <Cd>CORE</Cd>
                </LclInstrm>
                <SeqTp>RCUR</SeqTp>
            </PmtTpInf>

            <ReqdColltnDt>2025-01-25</ReqdColltnDt>

            <!-- Creditor -->
            <Cdtr>
                <Nm>Creditor GmbH</Nm>
            </Cdtr>
            <CdtrAcct>
                <Id>
                    <IBAN>AT611904300234573201</IBAN>
                </Id>
            </CdtrAcct>
            <CdtrAgt>
                <FinInstnId>
                    <BICFI>BKAUATWW</BICFI>
                </FinInstnId>
            </CdtrAgt>
            <CdtrSchmeId>
                <Id>
                    <PrvtId>
                        <Othr>
                            <Id>AT98ZZZ00000012345</Id>
                            <SchmeNm>
                                <Prtry>SEPA</Prtry>
                            </SchmeNm>
                        </Othr>
                    </PrvtId>
                </Id>
            </CdtrSchmeId>

            <!-- Transaction -->
            <DrctDbtTxInf>
                <PmtId>
                    <EndToEndId>DD-E2E-001</EndToEndId>
                </PmtId>
                <InstdAmt Ccy="EUR">100.00</InstdAmt>

                <!-- Mandate -->
                <DrctDbtTx>
                    <MndtRltdInf>
                        <MndtId>MNDT-001234</MndtId>
                        <DtOfSgntr>2024-01-01</DtOfSgntr>
                    </MndtRltdInf>
                </DrctDbtTx>

                <!-- Debtor -->
                <DbtrAgt>
                    <FinInstnId>
                        <BICFI>GIBAATWW</BICFI>
                    </FinInstnId>
                </DbtrAgt>
                <Dbtr>
                    <Nm>Debtor Person</Nm>
                </Dbtr>
                <DbtrAcct>
                    <Id>
                        <IBAN>AT021100000012345678</IBAN>
                    </Id>
                </DbtrAcct>

                <RmtInf>
                    <Ustrd>Abo-Zahlung Januar 2025</Ustrd>
                </RmtInf>
            </DrctDbtTxInf>
        </PmtInf>
    </CstmrDrctDbtInitn>
</Document>
```

### 3.2 Sequence Types

| Code | Description |
|------|-------------|
| FRST | First collection |
| RCUR | Recurring collection |
| OOFF | One-off collection |
| FNAL | Final collection |

---

## 4. camt.053 - Account Statement

### 4.1 Structure (Simplified)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.08">
    <BkToCstmrStmt>
        <GrpHdr>
            <MsgId>STMT-2025-001</MsgId>
            <CreDtTm>2025-01-16T06:00:00</CreDtTm>
        </GrpHdr>

        <Stmt>
            <Id>STMT-001</Id>
            <CreDtTm>2025-01-16T06:00:00</CreDtTm>

            <!-- Account -->
            <Acct>
                <Id>
                    <IBAN>AT611904300234573201</IBAN>
                </Id>
            </Acct>

            <!-- Opening Balance -->
            <Bal>
                <Tp>
                    <CdOrPrtry>
                        <Cd>OPBD</Cd>
                    </CdOrPrtry>
                </Tp>
                <Amt Ccy="EUR">10000.00</Amt>
                <CdtDbtInd>CRDT</CdtDbtInd>
                <Dt>
                    <Dt>2025-01-15</Dt>
                </Dt>
            </Bal>

            <!-- Closing Balance -->
            <Bal>
                <Tp>
                    <CdOrPrtry>
                        <Cd>CLBD</Cd>
                    </CdOrPrtry>
                </Tp>
                <Amt Ccy="EUR">9500.00</Amt>
                <CdtDbtInd>CRDT</CdtDbtInd>
                <Dt>
                    <Dt>2025-01-15</Dt>
                </Dt>
            </Bal>

            <!-- Entry -->
            <Ntry>
                <Amt Ccy="EUR">500.00</Amt>
                <CdtDbtInd>DBIT</CdtDbtInd>
                <Sts>
                    <Cd>BOOK</Cd>
                </Sts>
                <BookgDt>
                    <Dt>2025-01-15</Dt>
                </BookgDt>
                <ValDt>
                    <Dt>2025-01-15</Dt>
                </ValDt>
                <NtryDtls>
                    <TxDtls>
                        <Refs>
                            <EndToEndId>E2E-001</EndToEndId>
                        </Refs>
                        <RltdPties>
                            <Cdtr>
                                <Nm>Empfänger GmbH</Nm>
                            </Cdtr>
                            <CdtrAcct>
                                <Id>
                                    <IBAN>AT021100000012345678</IBAN>
                                </Id>
                            </CdtrAcct>
                        </RltdPties>
                        <RmtInf>
                            <Ustrd>Rechnung RE-2025-001</Ustrd>
                        </RmtInf>
                    </TxDtls>
                </NtryDtls>
            </Ntry>
        </Stmt>
    </BkToCstmrStmt>
</Document>
```

---

## 5. IBAN Validation

### 5.1 Structure

Austrian IBAN: AT + 2 check digits + 5 bank code + 11 account number = 20 chars

```
AT61 1904 3002 3457 3201
│  │  │    │
│  │  │    └── Account number (11 digits)
│  │  └── Bank code (5 digits)
│  └── Check digits (2 digits)
└── Country code
```

### 5.2 Validation Algorithm (ISO 7064 Mod 97)

```go
func ValidateIBAN(iban string) bool {
    // Remove spaces
    iban = strings.ReplaceAll(iban, " ", "")

    // Check length
    if len(iban) < 15 || len(iban) > 34 {
        return false
    }

    // Move first 4 chars to end
    rearranged := iban[4:] + iban[:4]

    // Convert letters to numbers (A=10, B=11, ..., Z=35)
    var numeric strings.Builder
    for _, c := range rearranged {
        if c >= 'A' && c <= 'Z' {
            numeric.WriteString(strconv.Itoa(int(c - 'A' + 10)))
        } else {
            numeric.WriteRune(c)
        }
    }

    // Mod 97 check
    n := new(big.Int)
    n.SetString(numeric.String(), 10)
    remainder := new(big.Int)
    remainder.Mod(n, big.NewInt(97))

    return remainder.Int64() == 1
}
```

### 5.3 BIC Lookup

Austrian bank codes to BIC mapping:

| Bank Code | BIC | Bank Name |
|-----------|-----|-----------|
| 19043 | BKAUATWW | Bank Austria |
| 11000 | BKAUATWW | Bank Austria |
| 12000 | BAWAATWW | BAWAG PSK |
| 20111 | GIBAATWW | Erste Bank |
| 32000 | RLNWATWW | Raiffeisen |
| 60000 | OPSKATWW | Sparkasse |

---

## 6. Go Struct Definitions

```go
// internal/sepa/pain001.go

// Pain001Document is the root structure
type Pain001Document struct {
    XMLName xml.Name        `xml:"Document"`
    XMLNS   string          `xml:"xmlns,attr"`
    Body    Pain001Content  `xml:"CstmrCdtTrfInitn"`
}

type Pain001Content struct {
    GroupHeader    GroupHeader    `xml:"GrpHdr"`
    PaymentInfo    []PaymentInfo  `xml:"PmtInf"`
}

type GroupHeader struct {
    MessageID       string    `xml:"MsgId"`
    CreationTime    string    `xml:"CreDtTm"`
    NumberOfTx      int       `xml:"NbOfTxs"`
    ControlSum      string    `xml:"CtrlSum"`
    InitiatingParty Party     `xml:"InitgPty"`
}

type PaymentInfo struct {
    PaymentInfoID   string              `xml:"PmtInfId"`
    PaymentMethod   string              `xml:"PmtMtd"`
    BatchBooking    bool                `xml:"BtchBookg"`
    NumberOfTx      int                 `xml:"NbOfTxs"`
    ControlSum      string              `xml:"CtrlSum"`
    ExecutionDate   ExecutionDate       `xml:"ReqdExctnDt"`
    Debtor          Party               `xml:"Dbtr"`
    DebtorAccount   Account             `xml:"DbtrAcct"`
    DebtorAgent     Agent               `xml:"DbtrAgt"`
    Transactions    []CreditTransaction `xml:"CdtTrfTxInf"`
}

type ExecutionDate struct {
    Date string `xml:"Dt"`
}

type Party struct {
    Name    string       `xml:"Nm"`
    Address *PostalAddr  `xml:"PstlAdr,omitempty"`
}

type PostalAddr struct {
    Street   string `xml:"StrtNm,omitempty"`
    Building string `xml:"BldgNb,omitempty"`
    PostCode string `xml:"PstCd,omitempty"`
    Town     string `xml:"TwnNm,omitempty"`
    Country  string `xml:"Ctry"`
}

type Account struct {
    ID AccountID `xml:"Id"`
}

type AccountID struct {
    IBAN string `xml:"IBAN"`
}

type Agent struct {
    FinancialInstitution FinInst `xml:"FinInstnId"`
}

type FinInst struct {
    BIC string `xml:"BICFI"`
}

type CreditTransaction struct {
    PaymentID      PaymentID `xml:"PmtId"`
    Amount         Amount    `xml:"Amt"`
    Creditor       Party     `xml:"Cdtr"`
    CreditorAccount Account  `xml:"CdtrAcct"`
    RemittanceInfo *RemInfo  `xml:"RmtInf,omitempty"`
}

type PaymentID struct {
    EndToEndID string `xml:"EndToEndId"`
}

type Amount struct {
    InstructedAmount InstAmt `xml:"InstdAmt"`
}

type InstAmt struct {
    Currency string `xml:"Ccy,attr"`
    Value    string `xml:",chardata"`
}

type RemInfo struct {
    Unstructured string `xml:"Ustrd"`
}
```

---

## 7. CLI Commands

```bash
# Credit Transfer
sepa pain001 payments.csv --output payments.xml
sepa pain001 payments.csv \
    --debtor-name "Auftraggeber GmbH" \
    --debtor-iban AT611904300234573201 \
    --execution-date 2025-01-20

# Direct Debit
sepa pain008 collections.csv --output collections.xml

# Parse Statement
sepa camt053 statement.xml
sepa camt053 statement.xml --json
sepa camt053 statement.xml --output transactions.csv

# IBAN Validation
sepa validate AT611904300234573201
sepa validate --file ibans.txt

# Batch from CSV
sepa pain001 --input payments.csv --output payments.xml
```

---

## 8. Input CSV Format (pain.001)

```csv
end_to_end_id,creditor_name,creditor_iban,amount,currency,remittance_info
E2E-001,Empfänger 1 GmbH,AT021100000012345678,1000.00,EUR,Rechnung RE-2025-001
E2E-002,Empfänger 2 AG,AT301200000098765432,500.00,EUR,Rechnung RE-2025-002
```

---

## 9. Version Compatibility

| Version | Support Until | Notes |
|---------|---------------|-------|
| pain.001.001.03 | Nov 2026 | Unstructured addresses allowed |
| pain.001.001.09 | Current | Structured addresses required from Nov 2025 |
| camt.053.001.02 | Oct 2025 | Legacy format |
| camt.053.001.08 | Current | Current format |

**Recommendation**: Generate pain.001.001.09 with structured addresses to be future-proof.
