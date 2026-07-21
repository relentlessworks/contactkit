package model

import (
	"strings"
	"testing"
)

func TestGenerateHandle(t *testing.T) {
	h, err := GenerateHandle()
	if err != nil {
		t.Fatalf("GenerateHandle() error: %v", err)
	}
	if !strings.HasPrefix(h, "contact_") {
		t.Errorf("handle should start with 'contact_', got %s", h)
	}
	if len(h) != len("contact_")+5 {
		t.Errorf("handle should be contact_ + 5 chars, got %s (len %d)", h, len(h))
	}
}

func TestGenerateHandleUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		h, err := GenerateHandle()
		if err != nil {
			t.Fatalf("GenerateHandle() error: %v", err)
		}
		if seen[h] {
			t.Fatalf("duplicate handle generated: %s", h)
		}
		seen[h] = true
	}
}

func TestGenerateWorkspaceHandle(t *testing.T) {
	h, err := GenerateWorkspaceHandle()
	if err != nil {
		t.Fatalf("GenerateWorkspaceHandle() error: %v", err)
	}
	if !strings.HasPrefix(h, "ws_") {
		t.Errorf("workspace handle should start with 'ws_', got %s", h)
	}
	if len(h) != len("ws_")+5 {
		t.Errorf("workspace handle should be ws_ + 5 chars, got %s (len %d)", h, len(h))
	}
}

func TestGenerateToken(t *testing.T) {
	tok, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken() error: %v", err)
	}
	if len(tok) < 16 {
		t.Errorf("token too short: %s (len %d)", tok, len(tok))
	}
}

func TestGenerateTokenUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		tok, err := GenerateToken()
		if err != nil {
			t.Fatalf("GenerateToken() error: %v", err)
		}
		if seen[tok] {
			t.Fatalf("duplicate token generated: %s", tok)
		}
		seen[tok] = true
	}
}

func TestGenerateOTP(t *testing.T) {
	code, err := GenerateOTP()
	if err != nil {
		t.Fatalf("GenerateOTP() error: %v", err)
	}
	if len(code) != 6 {
		t.Errorf("OTP should be 6 digits, got %s (len %d)", code, len(code))
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			t.Errorf("OTP should be all digits, got %s", code)
			break
		}
	}
}
