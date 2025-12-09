package types

import "encoding/json"

// Credentials is a union type for all credential types
type Credentials struct {
	Type        string `json:"type"`
	FinanzOnline *FinanzOnlineCredentials `json:"finanzonline,omitempty"`
	ELDA        *ELDACredentials         `json:"elda,omitempty"`
	Firmenbuch  *FirmenbuchCredentials   `json:"firmenbuch,omitempty"`
}

// MarshalCredentials converts typed credentials to JSON for encryption
func MarshalCredentials(accountType string, creds interface{}) ([]byte, error) {
	return json.Marshal(creds)
}

// UnmarshalCredentials converts decrypted JSON to typed credentials
func UnmarshalCredentials(accountType string, data []byte) (interface{}, error) {
	switch accountType {
	case "finanzonline":
		var creds FinanzOnlineCredentials
		if err := json.Unmarshal(data, &creds); err != nil {
			return nil, err
		}
		return &creds, nil
	case "elda":
		var creds ELDACredentials
		if err := json.Unmarshal(data, &creds); err != nil {
			return nil, err
		}
		return &creds, nil
	case "firmenbuch":
		var creds FirmenbuchCredentials
		if err := json.Unmarshal(data, &creds); err != nil {
			return nil, err
		}
		return &creds, nil
	default:
		return nil, nil
	}
}

// MaskCredentials returns masked version of credentials
func MaskCredentials(accountType string, creds interface{}) interface{} {
	switch c := creds.(type) {
	case *FinanzOnlineCredentials:
		return c.Masked()
	case *ELDACredentials:
		return c.Masked()
	case *FirmenbuchCredentials:
		return c.Masked()
	default:
		return nil
	}
}
