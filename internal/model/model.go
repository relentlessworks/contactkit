package model

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"time"
)

// Contact represents a person or organization in the CRM.
type Contact struct {
	Handle    string    `json:"handle"`
	Name      string    `json:"name"`
	Email     string    `json:"email,omitempty"`
	Phone     string    `json:"phone,omitempty"`
	Company   string    `json:"company,omitempty"`
	Title     string    `json:"title,omitempty"`
	Notes     string    `json:"notes,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Workspace represents a tenant in the system.
type Workspace struct {
	Handle    string    `json:"handle"`
	Name      string    `json:"name"`
	Plan      string    `json:"plan"`
	CreatedAt time.Time `json:"created_at"`
}

// Token represents an auth token.
type Token struct {
	Token     string    `json:"token"`
	Workspace string    `json:"workspace"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// OTP represents a one-time password for auth.
type OTP struct {
	Email     string    `json:"email"`
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expires_at"`
}

const handleChars = "abcdefghijklmnopqrstuvwxyz234567"

// generateID generates a random base32 string of the given length.
func generateID(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	for i := range b {
		b[i] = handleChars[int(b[i])%len(handleChars)]
	}
	return string(b), nil
}

// GenerateHandle creates a workspace-scoped handle for a contact.
// Format: contact_5char (e.g. contact_k7m2q)
func GenerateHandle() (string, error) {
	id, err := generateID(5)
	if err != nil {
		return "", err
	}
	return "contact_" + id, nil
}

// GenerateWorkspaceHandle creates a handle for a workspace.
// Format: ws_5char (e.g. ws_abc12)
func GenerateWorkspaceHandle() (string, error) {
	id, err := generateID(5)
	if err != nil {
		return "", err
	}
	return "ws_" + id, nil
}

// GenerateToken creates a random bearer token.
func GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base32.StdEncoding.EncodeToString(b), nil
}

// GenerateOTP creates a 6-digit one-time password.
func GenerateOTP() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate otp: %w", err)
	}
	code := (uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])) % 1000000
	return fmt.Sprintf("%06d", code), nil
}
