package auth

import (
	"fmt"
	"log"
	"net/smtp"
	"time"

	"github.com/relentlessworks/contactkit/internal/model"
	"github.com/relentlessworks/contactkit/internal/store"
)

// Auth handles OTP-based authentication.
type Auth struct {
	store    *store.Store
	smtpAddr string
}

// New creates a new auth handler.
func New(s *store.Store, smtpAddr string) *Auth {
	return &Auth{
		store:    s,
		smtpAddr: smtpAddr,
	}
}

// RequestOTP generates and sends/stores an OTP for the given email.
func (a *Auth) RequestOTP(email string) error {
	code, err := model.GenerateOTP()
	if err != nil {
		return fmt.Errorf("generate otp: %w", err)
	}

	otp := &model.OTP{
		Email:     email,
		Code:      code,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	if err := a.store.SaveOTP(otp); err != nil {
		return fmt.Errorf("save otp: %w", err)
	}

	// Send OTP via email or log to stderr
	if a.smtpAddr != "" {
		if err := a.sendEmail(email, code); err != nil {
			return fmt.Errorf("send email: %w", err)
		}
	} else {
		log.Printf("[OTP] %s -> code: %s (dev mode: no SMTP configured)", email, code)
	}

	return nil
}

// VerifyOTP validates an OTP and returns a bearer token.
func (a *Auth) VerifyOTP(email, code string) (*model.Token, error) {
	otp, ok := a.store.GetOTP(email)
	if !ok {
		return nil, fmt.Errorf("no OTP found for %s", email)
	}

	if time.Now().After(otp.ExpiresAt) {
		_ = a.store.DeleteOTP(email)
		return nil, fmt.Errorf("OTP expired")
	}

	if otp.Code != code {
		return nil, fmt.Errorf("invalid OTP code")
	}

	_ = a.store.DeleteOTP(email)

	token, err := model.GenerateToken()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	// Find or create a workspace for this user
	wsHandle := a.findOrCreateWorkspace(email)

	t := &model.Token{
		Token:     token,
		Workspace: wsHandle,
		Email:     email,
		CreatedAt: time.Now(),
	}

	if err := a.store.SaveToken(t); err != nil {
		return nil, fmt.Errorf("save token: %w", err)
	}

	return t, nil
}

// ValidateToken checks if a token is valid and returns the associated workspace.
func (a *Auth) ValidateToken(token string) (*model.Token, bool) {
	t, ok := a.store.GetToken(token)
	return t, ok
}

// RevokeToken removes a token.
func (a *Auth) RevokeToken(token string) error {
	return a.store.DeleteToken(token)
}

func (a *Auth) sendEmail(email, code string) error {
	// Basic SMTP send - in production this would be more robust
	addr := a.smtpAddr
	from := "noreply@contactkit.local"
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: Your ContactKit OTP\r\n\r\nYour verification code is: %s\r\n", from, email, code)
	return smtp.SendMail(addr, nil, from, []string{email}, []byte(msg))
}

func (a *Auth) findOrCreateWorkspace(email string) string {
	// Check if user already has a workspace via existing tokens
	for _, ws := range a.store.ListWorkspaces() {
		// Simple approach: use workspace name as email for first-time users
		if ws.Name == email {
			return ws.Handle
		}
	}

	// Create a new workspace
	wsHandle, err := model.GenerateWorkspaceHandle()
	if err != nil {
		// This should never happen, but we need a fallback
		wsHandle = "ws_error"
	}

	ws := &model.Workspace{
		Handle:    wsHandle,
		Name:      email,
		Plan:      "free",
		CreatedAt: time.Now(),
	}

	_ = a.store.CreateWorkspace(ws)
	return wsHandle
}
