package account

import (
	"context"
	"fmt"

	"github.com/austrian-business-infrastructure/fo/internal/account/types"
	"github.com/austrian-business-infrastructure/fo/internal/fonws"
)

// FinanzOnlineConnector tests connections to FinanzOnline
type FinanzOnlineConnector struct {
	client *fonws.Client
}

// NewFinanzOnlineConnector creates a new FO connector
func NewFinanzOnlineConnector() *FinanzOnlineConnector {
	return &FinanzOnlineConnector{
		client: fonws.NewClient(),
	}
}

// TestConnection tests a FinanzOnline connection
func (c *FinanzOnlineConnector) TestConnection(ctx context.Context, creds interface{}) (*ConnectionTestResult, error) {
	foCreds, ok := creds.(*types.FinanzOnlineCredentials)
	if !ok {
		return &ConnectionTestResult{
			Success:      false,
			ErrorMessage: "invalid credential type",
		}, nil
	}

	// Attempt login
	req := fonws.LoginRequest{
		Xmlns: fonws.SessionNS,
		TID:   foCreds.TID,
		BenID: foCreds.BenID,
		PIN:   foCreds.PIN,
		Herst: "false",
	}

	var resp fonws.LoginResponse
	err := c.client.Call(fonws.SessionServiceURL, req, &resp)
	if err != nil {
		return &ConnectionTestResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("connection failed: %v", err),
		}, nil
	}

	// Check response code
	if foErr := fonws.CheckResponse(resp.RC, resp.Msg); foErr != nil {
		return &ConnectionTestResult{
			Success:      false,
			ErrorCode:    fmt.Sprintf("%d", resp.RC),
			ErrorMessage: foErr.Error(),
		}, nil
	}

	// Login successful, now logout
	if resp.ID != "" {
		logoutReq := fonws.LogoutRequest{
			Xmlns: fonws.SessionNS,
			ID:    resp.ID,
		}
		var logoutResp fonws.LogoutResponse
		_ = c.client.Call(fonws.SessionServiceURL, logoutReq, &logoutResp)
	}

	return &ConnectionTestResult{
		Success: true,
	}, nil
}
