package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
)

var (
	ErrAccountNotFound   = errors.New("account not found")
	ErrDuplicateAccount  = errors.New("account with this name already exists")
	ErrInvalidVersion    = errors.New("invalid credential store version")
	ErrInvalidAccountType = errors.New("invalid account type")
	tidPattern           = regexp.MustCompile(`^\d{12}$`)
	dienstgeberNrPattern = regexp.MustCompile(`^\d{8}$`)
)

// AccountType distinguishes credential types
type AccountType string

const (
	AccountTypeFinanzOnline AccountType = "finanzonline"
	AccountTypeELDA         AccountType = "elda"
	AccountTypeFirmenbuch   AccountType = "firmenbuch"
)

// ValidAccountTypes returns all valid account types
func ValidAccountTypes() []AccountType {
	return []AccountType{AccountTypeFinanzOnline, AccountTypeELDA, AccountTypeFirmenbuch}
}

// IsValidAccountType checks if the given type is valid
func IsValidAccountType(t AccountType) bool {
	for _, valid := range ValidAccountTypes() {
		if t == valid {
			return true
		}
	}
	return false
}

// Account represents a stored FinanzOnline WebService account
// Kept for backward compatibility - use ExtendedAccount for new code
type Account struct {
	Name  string `json:"name"`
	TID   string `json:"tid"`
	BenID string `json:"benid"`
	PIN   string `json:"pin"`
}

// ExtendedAccount extends the base account with type-specific fields
type ExtendedAccount struct {
	Name string      `json:"name"`
	Type AccountType `json:"type"`

	// FinanzOnline fields (Type == AccountTypeFinanzOnline)
	TID   string `json:"tid,omitempty"`
	BenID string `json:"benid,omitempty"`
	PIN   string `json:"pin,omitempty"`

	// ELDA fields (Type == AccountTypeELDA)
	DienstgeberNr string `json:"dienstgeber_nr,omitempty"`
	ELDABenutzer  string `json:"elda_benutzer,omitempty"`
	ELDAPIN       string `json:"elda_pin,omitempty"`

	// Firmenbuch fields (Type == AccountTypeFirmenbuch)
	APIKey string `json:"api_key,omitempty"`
}

// ToAccount converts ExtendedAccount to legacy Account (for FinanzOnline only)
func (ea *ExtendedAccount) ToAccount() *Account {
	return &Account{
		Name:  ea.Name,
		TID:   ea.TID,
		BenID: ea.BenID,
		PIN:   ea.PIN,
	}
}

// FromAccount creates an ExtendedAccount from a legacy Account
func FromAccount(a *Account) *ExtendedAccount {
	return &ExtendedAccount{
		Name:  a.Name,
		Type:  AccountTypeFinanzOnline,
		TID:   a.TID,
		BenID: a.BenID,
		PIN:   a.PIN,
	}
}

// Validate checks if the ExtendedAccount fields are valid based on type
func (ea *ExtendedAccount) Validate() error {
	if ea.Name == "" {
		return errors.New("name must not be empty")
	}
	if len(ea.Name) > 100 {
		return errors.New("name must not exceed 100 characters")
	}
	if !IsValidAccountType(ea.Type) {
		return ErrInvalidAccountType
	}

	switch ea.Type {
	case AccountTypeFinanzOnline:
		return ea.validateFinanzOnline()
	case AccountTypeELDA:
		return ea.validateELDA()
	case AccountTypeFirmenbuch:
		return ea.validateFirmenbuch()
	}

	return nil
}

func (ea *ExtendedAccount) validateFinanzOnline() error {
	if !tidPattern.MatchString(ea.TID) {
		return errors.New("tid must be exactly 12 digits")
	}
	if ea.BenID == "" {
		return errors.New("benid must not be empty")
	}
	if ea.PIN == "" {
		return errors.New("pin must not be empty")
	}
	return nil
}

func (ea *ExtendedAccount) validateELDA() error {
	if !dienstgeberNrPattern.MatchString(ea.DienstgeberNr) {
		return errors.New("dienstgeber_nr must be exactly 8 digits")
	}
	if ea.ELDABenutzer == "" {
		return errors.New("elda_benutzer must not be empty")
	}
	if ea.ELDAPIN == "" {
		return errors.New("elda_pin must not be empty")
	}
	return nil
}

func (ea *ExtendedAccount) validateFirmenbuch() error {
	if ea.APIKey == "" {
		return errors.New("api_key must not be empty")
	}
	return nil
}

// Validate checks if the account fields are valid
func (a *Account) Validate() error {
	if a.Name == "" {
		return errors.New("name must not be empty")
	}
	if len(a.Name) > 100 {
		return errors.New("name must not exceed 100 characters")
	}
	if !tidPattern.MatchString(a.TID) {
		return errors.New("tid must be exactly 12 digits")
	}
	if a.BenID == "" {
		return errors.New("benid must not be empty")
	}
	if a.PIN == "" {
		return errors.New("pin must not be empty")
	}
	return nil
}

// CredentialStore represents the encrypted file containing all accounts
type CredentialStore struct {
	Version  int       `json:"version"`
	Accounts []Account `json:"accounts"`
}

// NewCredentialStore creates a new empty credential store
func NewCredentialStore() *CredentialStore {
	return &CredentialStore{
		Version:  1,
		Accounts: []Account{},
	}
}

// ToJSON serializes the credential store to JSON
func (cs *CredentialStore) ToJSON() ([]byte, error) {
	return json.Marshal(cs)
}

// FromJSON deserializes a credential store from JSON
func FromJSON(data []byte) (*CredentialStore, error) {
	var cs CredentialStore
	if err := json.Unmarshal(data, &cs); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if cs.Version != 1 {
		return nil, ErrInvalidVersion
	}
	return &cs, nil
}

// EncryptStore encrypts the credential store with the master password
func (cs *CredentialStore) EncryptStore(masterPassword string) ([]byte, error) {
	plaintext, err := cs.ToJSON()
	if err != nil {
		return nil, err
	}
	return Encrypt(plaintext, masterPassword)
}

// DecryptStore decrypts data and returns a credential store
func DecryptStore(data []byte, masterPassword string) (*CredentialStore, error) {
	plaintext, err := Decrypt(data, masterPassword)
	if err != nil {
		return nil, err
	}
	return FromJSON(plaintext)
}

// Load reads and decrypts a credential store from a file
func Load(path string, masterPassword string) (*CredentialStore, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read credential file: %w", err)
	}

	return DecryptStore(data, masterPassword)
}

// Save encrypts and writes the credential store to a file
func (cs *CredentialStore) Save(path string, masterPassword string) error {
	encrypted, err := cs.EncryptStore(masterPassword)
	if err != nil {
		return err
	}
	return os.WriteFile(path, encrypted, 0600)
}

// AddAccount adds a new account to the store
func (cs *CredentialStore) AddAccount(account Account) error {
	if err := account.Validate(); err != nil {
		return err
	}

	// Check for duplicate name
	for _, existing := range cs.Accounts {
		if existing.Name == account.Name {
			return ErrDuplicateAccount
		}
	}

	cs.Accounts = append(cs.Accounts, account)
	return nil
}

// RemoveAccount removes an account by name
func (cs *CredentialStore) RemoveAccount(name string) error {
	for i, acc := range cs.Accounts {
		if acc.Name == name {
			cs.Accounts = append(cs.Accounts[:i], cs.Accounts[i+1:]...)
			return nil
		}
	}
	return ErrAccountNotFound
}

// GetAccount retrieves an account by name
func (cs *CredentialStore) GetAccount(name string) (*Account, error) {
	for i := range cs.Accounts {
		if cs.Accounts[i].Name == name {
			return &cs.Accounts[i], nil
		}
	}
	return nil, ErrAccountNotFound
}

// ListAccounts returns a list of account names (without sensitive data)
func (cs *CredentialStore) ListAccounts() []string {
	names := make([]string, len(cs.Accounts))
	for i, acc := range cs.Accounts {
		names[i] = acc.Name
	}
	return names
}
