package fonws

import (
	"encoding/xml"
	"time"
)

// LoginRequest represents a SOAP Login request
type LoginRequest struct {
	XMLName xml.Name `xml:"Login"`
	Xmlns   string   `xml:"xmlns,attr"`
	TID     string   `xml:"tid"`
	BenID   string   `xml:"benid"`
	PIN     string   `xml:"pin"`
	Herst   string   `xml:"heression"`
}

// LoginResponse represents a SOAP Login response
type LoginResponse struct {
	XMLName xml.Name `xml:"LoginResponse"`
	RC      int      `xml:"rc"`
	Msg     string   `xml:"msg"`
	ID      string   `xml:"id"`
}

// LogoutRequest represents a SOAP Logout request
type LogoutRequest struct {
	XMLName xml.Name `xml:"Logout"`
	Xmlns   string   `xml:"xmlns,attr"`
	ID      string   `xml:"id"`
	TID     string   `xml:"tid"`
	BenID   string   `xml:"benid"`
}

// LogoutResponse represents a SOAP Logout response
type LogoutResponse struct {
	XMLName xml.Name `xml:"LogoutResponse"`
	RC      int      `xml:"rc"`
	Msg     string   `xml:"msg"`
}

// Session represents an active FinanzOnline session
type Session struct {
	Token       string
	AccountName string
	TID         string
	BenID       string
	CreatedAt   time.Time
	Valid       bool
}

// SessionService handles session operations
type SessionService struct {
	client *Client
}

// NewSessionService creates a new session service
func NewSessionService(client *Client) *SessionService {
	return &SessionService{client: client}
}

// Login authenticates with FinanzOnline and returns a session
func (s *SessionService) Login(tid, benid, pin string) (*Session, error) {
	req := LoginRequest{
		Xmlns:  SessionNS,
		TID:    tid,
		BenID:  benid,
		PIN:    pin,
		Herst:  "false",
	}

	var resp LoginResponse
	if err := s.client.Call(SessionServiceURL, req, &resp); err != nil {
		return nil, err
	}

	// Check for errors
	if err := CheckResponse(resp.RC, resp.Msg); err != nil {
		return nil, err
	}

	return &Session{
		Token:     resp.ID,
		TID:       tid,
		BenID:     benid,
		CreatedAt: time.Now(),
		Valid:     true,
	}, nil
}

// Logout terminates an active session
func (s *SessionService) Logout(session *Session) error {
	if session == nil || !session.Valid {
		return ErrNoActiveSession
	}

	req := LogoutRequest{
		Xmlns:  SessionNS,
		ID:     session.Token,
		TID:    session.TID,
		BenID:  session.BenID,
	}

	var resp LogoutResponse
	if err := s.client.Call(SessionServiceURL, req, &resp); err != nil {
		return err
	}

	// Check for errors (ignore session expired on logout)
	if resp.RC != ErrCodeNone && resp.RC != ErrCodeSessionExpired {
		return CheckResponse(resp.RC, resp.Msg)
	}

	session.Valid = false
	return nil
}

// Invalidate marks the session as invalid
func (session *Session) Invalidate() {
	if session != nil {
		session.Valid = false
	}
}
