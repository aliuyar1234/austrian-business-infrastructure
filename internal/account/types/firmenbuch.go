package types

// FirmenbuchCredentials holds Firmenbuch API credentials
type FirmenbuchCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Masked returns credentials with password masked
func (c *FirmenbuchCredentials) Masked() *FirmenbuchCredentials {
	return &FirmenbuchCredentials{
		Username: c.Username,
		Password: "****",
	}
}

// IsComplete checks if all required fields are present
func (c *FirmenbuchCredentials) IsComplete() bool {
	return c.Username != "" && c.Password != ""
}
