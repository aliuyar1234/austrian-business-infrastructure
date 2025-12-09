package types

// FinanzOnlineCredentials holds FO API credentials
type FinanzOnlineCredentials struct {
	TID   string `json:"tid"`
	BenID string `json:"ben_id"`
	PIN   string `json:"pin"`
}

// Masked returns credentials with PIN masked
func (c *FinanzOnlineCredentials) Masked() *FinanzOnlineCredentials {
	return &FinanzOnlineCredentials{
		TID:   c.TID,
		BenID: c.BenID,
		PIN:   "****",
	}
}

// IsComplete checks if all required fields are present
func (c *FinanzOnlineCredentials) IsComplete() bool {
	return c.TID != "" && c.BenID != "" && c.PIN != ""
}
