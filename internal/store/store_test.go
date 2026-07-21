package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/relentlessworks/contactkit/internal/model"
)

func tempStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := New(filepath.Join(dir, "test-data.json"))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	return s
}

func TestCreateAndGetWorkspace(t *testing.T) {
	s := tempStore(t)
	ws := &model.Workspace{
		Handle:    "ws_test1",
		Name:      "test@example.com",
		Plan:      "free",
		CreatedAt: time.Now(),
	}
	if err := s.CreateWorkspace(ws); err != nil {
		t.Fatalf("CreateWorkspace() error: %v", err)
	}
	got, ok := s.GetWorkspace("ws_test1")
	if !ok {
		t.Fatal("workspace not found")
	}
	if got.Name != "test@example.com" {
		t.Errorf("expected name test@example.com, got %s", got.Name)
	}
}

func TestCreateAndGetContact(t *testing.T) {
	s := tempStore(t)
	ws := "ws_test1"
	_ = s.CreateWorkspace(&model.Workspace{Handle: ws, Name: "test", Plan: "free", CreatedAt: time.Now()})

	c := &model.Contact{
		Handle:    "contact_abc12",
		Name:      "Jane Doe",
		Email:     "jane@acme.com",
		Company:   "Acme",
		Tags:      []string{"vip", "prospect"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.CreateContact(ws, c); err != nil {
		t.Fatalf("CreateContact() error: %v", err)
	}

	got, ok := s.GetContact(ws, "contact_abc12")
	if !ok {
		t.Fatal("contact not found")
	}
	if got.Name != "Jane Doe" {
		t.Errorf("expected name Jane Doe, got %s", got.Name)
	}
	if len(got.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(got.Tags))
	}
}

func TestUpdateContact(t *testing.T) {
	s := tempStore(t)
	ws := "ws_test1"
	_ = s.CreateWorkspace(&model.Workspace{Handle: ws, Name: "test", Plan: "free", CreatedAt: time.Now()})

	c := &model.Contact{
		Handle:    "contact_abc12",
		Name:      "Jane Doe",
		Email:     "jane@acme.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = s.CreateContact(ws, c)

	c.Title = "CTO"
	c.Company = "Acme Corp"
	if err := s.UpdateContact(ws, c); err != nil {
		t.Fatalf("UpdateContact() error: %v", err)
	}

	got, _ := s.GetContact(ws, "contact_abc12")
	if got.Title != "CTO" {
		t.Errorf("expected title CTO, got %s", got.Title)
	}
}

func TestDeleteContact(t *testing.T) {
	s := tempStore(t)
	ws := "ws_test1"
	_ = s.CreateWorkspace(&model.Workspace{Handle: ws, Name: "test", Plan: "free", CreatedAt: time.Now()})

	c := &model.Contact{
		Handle:    "contact_abc12",
		Name:      "Jane Doe",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = s.CreateContact(ws, c)

	if err := s.DeleteContact(ws, "contact_abc12"); err != nil {
		t.Fatalf("DeleteContact() error: %v", err)
	}

	_, ok := s.GetContact(ws, "contact_abc12")
	if ok {
		t.Fatal("contact should be deleted")
	}
}

func TestDeleteContactNotFound(t *testing.T) {
	s := tempStore(t)
	ws := "ws_test1"
	_ = s.CreateWorkspace(&model.Workspace{Handle: ws, Name: "test", Plan: "free", CreatedAt: time.Now()})

	err := s.DeleteContact(ws, "contact_nonexist")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestListContacts(t *testing.T) {
	s := tempStore(t)
	ws := "ws_test1"
	_ = s.CreateWorkspace(&model.Workspace{Handle: ws, Name: "test", Plan: "free", CreatedAt: time.Now()})

	_ = s.CreateContact(ws, &model.Contact{Handle: "contact_aaa1", Name: "Alice", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	_ = s.CreateContact(ws, &model.Contact{Handle: "contact_bbb2", Name: "Bob", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	contacts := s.ListContacts(ws)
	if len(contacts) != 2 {
		t.Errorf("expected 2 contacts, got %d", len(contacts))
	}
}

func TestSearchContacts(t *testing.T) {
	s := tempStore(t)
	ws := "ws_test1"
	_ = s.CreateWorkspace(&model.Workspace{Handle: ws, Name: "test", Plan: "free", CreatedAt: time.Now()})

	_ = s.CreateContact(ws, &model.Contact{Handle: "contact_aaa1", Name: "Alice Smith", Email: "alice@acme.com", Company: "Acme", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	_ = s.CreateContact(ws, &model.Contact{Handle: "contact_bbb2", Name: "Bob Jones", Email: "bob@globex.com", Company: "Globex", Tags: []string{"vip"}, CreatedAt: time.Now(), UpdatedAt: time.Now()})

	// Search by name
	results := s.SearchContacts(ws, "alice")
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'alice', got %d", len(results))
	}

	// Search by company
	results = s.SearchContacts(ws, "acme")
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'acme', got %d", len(results))
	}

	// Search by tag
	results = s.SearchContacts(ws, "vip")
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'vip', got %d", len(results))
	}

	// Search with no matches
	results = s.SearchContacts(ws, "nonexistent")
	if len(results) != 0 {
		t.Errorf("expected 0 results for 'nonexistent', got %d", len(results))
	}
}

func TestTokenOperations(t *testing.T) {
	s := tempStore(t)
	tok := &model.Token{
		Token:     "test-token-123",
		Workspace: "ws_test1",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
	}
	if err := s.SaveToken(tok); err != nil {
		t.Fatalf("SaveToken() error: %v", err)
	}

	got, ok := s.GetToken("test-token-123")
	if !ok {
		t.Fatal("token not found")
	}
	if got.Workspace != "ws_test1" {
		t.Errorf("expected workspace ws_test1, got %s", got.Workspace)
	}

	if err := s.DeleteToken("test-token-123"); err != nil {
		t.Fatalf("DeleteToken() error: %v", err)
	}

	_, ok = s.GetToken("test-token-123")
	if ok {
		t.Fatal("token should be deleted")
	}
}

func TestOTPOperations(t *testing.T) {
	s := tempStore(t)
	otp := &model.OTP{
		Email:     "test@example.com",
		Code:      "123456",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	if err := s.SaveOTP(otp); err != nil {
		t.Fatalf("SaveOTP() error: %v", err)
	}

	got, ok := s.GetOTP("test@example.com")
	if !ok {
		t.Fatal("OTP not found")
	}
	if got.Code != "123456" {
		t.Errorf("expected code 123456, got %s", got.Code)
	}

	if err := s.DeleteOTP("test@example.com"); err != nil {
		t.Fatalf("DeleteOTP() error: %v", err)
	}

	_, ok = s.GetOTP("test@example.com")
	if ok {
		t.Fatal("OTP should be deleted")
	}
}

func TestPersistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "persist-test.json")

	// Create and save data
	s1, err := New(path)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	_ = s1.CreateWorkspace(&model.Workspace{Handle: "ws_test1", Name: "test", Plan: "free", CreatedAt: time.Now()})
	_ = s1.CreateContact("ws_test1", &model.Contact{Handle: "contact_abc12", Name: "Jane", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	// Create a new store from the same file
	s2, err := New(path)
	if err != nil {
		t.Fatalf("New() second instance error: %v", err)
	}

	ws, ok := s2.GetWorkspace("ws_test1")
	if !ok {
		t.Fatal("workspace not persisted")
	}
	if ws.Name != "test" {
		t.Errorf("expected name 'test', got %s", ws.Name)
	}

	c, ok := s2.GetContact("ws_test1", "contact_abc12")
	if !ok {
		t.Fatal("contact not persisted")
	}
	if c.Name != "Jane" {
		t.Errorf("expected name 'Jane', got %s", c.Name)
	}
}

func TestNewStoreNonExistentFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.json")
	s, err := New(path)
	if err != nil {
		t.Fatalf("New() with non-existent file should not error: %v", err)
	}
	if s == nil {
		t.Fatal("store should not be nil")
	}
}

func TestNewStoreEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json")
	_ = os.WriteFile(path, []byte{}, 0600)
	s, err := New(path)
	if err != nil {
		t.Fatalf("New() with empty file should not error: %v", err)
	}
	if s == nil {
		t.Fatal("store should not be nil")
	}
}
