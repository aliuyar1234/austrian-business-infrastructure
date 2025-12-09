package types

// ELDACredentials holds ELDA API credentials
type ELDACredentials struct {
	DienstgeberNr       string `json:"dienstgeber_nr"`
	PIN                 string `json:"pin"`
	CertificatePath     string `json:"certificate_path"`
	CertificatePassword string `json:"certificate_password"`
}

// Masked returns credentials with sensitive fields masked
func (c *ELDACredentials) Masked() *ELDACredentials {
	return &ELDACredentials{
		DienstgeberNr:       c.DienstgeberNr,
		PIN:                 "****",
		CertificatePath:     c.CertificatePath,
		CertificatePassword: "****",
	}
}

// IsComplete checks if all required fields are present
func (c *ELDACredentials) IsComplete() bool {
	return c.DienstgeberNr != "" && c.PIN != "" && c.CertificatePath != ""
}
