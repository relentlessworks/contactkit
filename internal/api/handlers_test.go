package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/relentlessworks/contactkit/internal/auth"
	"github.com/relentlessworks/contactkit/internal/model"
	"github.com/relentlessworks/contactkit/internal/store"
)

func setupTestHandler(t *testing.T) (*Handler, string) {
	t.Helper()
	dir := t.TempDir()
	s, err := store.New(dir + "/test-api.json")
	if err != nil {
		t.Fatalf("store.New() error: %v", err)
	}
	a := auth.New(s, "") // dev mode, no SMTP
	h := NewHandler(s, a)

	// Create a workspace and token for authenticated requests
	ws := &model.Workspace{
		Handle: "ws_test1",
		Name:   "test@example.com",
		Plan:   "free",
	}
	_ = s.CreateWorkspace(ws)

	tok := &model.Token{
		Token:     "test-bearer-token",
		Workspace: "ws_test1",
		Email:     "test@example.com",
	}
	_ = s.SaveToken(tok)

	return h, "test-bearer-token"
}

func TestHelp(t *testing.T) {
	h, _ := setupTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/help", nil)
	w := httptest.NewRecorder()
	h.handleHelp(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "ContactKit") {
		t.Error("help should mention ContactKit")
	}
	if !strings.Contains(body, "POST /contacts") {
		t.Error("help should mention POST /contacts")
	}
	if !strings.Contains(body, "auth/request") {
		t.Error("help should mention auth flow")
	}
}

func TestAuthRequestMissingEmail(t *testing.T) {
	h, _ := setupTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/auth/request", nil)
	w := httptest.NewRecorder()
	h.handleAuthRequest(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "missing email") {
		t.Errorf("error should mention missing email, got: %s", body)
	}
	if !strings.Contains(body, "hint:") {
		t.Error("error should include a hint")
	}
}

func TestAuthRequestSuccess(t *testing.T) {
	h, _ := setupTestHandler(t)
	form := url.Values{"email": {"newuser@example.com"}}
	req := httptest.NewRequest(http.MethodPost, "/auth/request", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.handleAuthRequest(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAuthVerifyMissingParams(t *testing.T) {
	h, _ := setupTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/auth/verify", nil)
	w := httptest.NewRecorder()
	h.handleAuthVerify(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateContactMissingName(t *testing.T) {
	h, token := setupTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/contacts", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.handleContacts(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "missing name") {
		t.Errorf("error should mention missing name, got: %s", body)
	}
}

func TestCreateContactSuccess(t *testing.T) {
	h, token := setupTestHandler(t)
	form := url.Values{
		"name":    {"Jane Doe"},
		"email":   {"jane@acme.com"},
		"company": {"Acme"},
		"tags":    {"vip,prospect"},
	}
	req := httptest.NewRequest(http.MethodPost, "/contacts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.handleContacts(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "handle=contact_") {
		t.Errorf("response should contain handle, got: %s", body)
	}
	if !strings.Contains(body, "name=Jane Doe") {
		t.Errorf("response should contain name, got: %s", body)
	}
	if !strings.Contains(body, "tags=vip,prospect") {
		t.Errorf("response should contain tags, got: %s", body)
	}
}

func TestCreateContactJSON(t *testing.T) {
	h, token := setupTestHandler(t)
	form := url.Values{
		"name":  {"JSON Person"},
		"email": {"json@test.com"},
	}
	req := httptest.NewRequest(http.MethodPost, "/contacts?format=json", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.handleContacts(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
	var c model.Contact
	if err := json.NewDecoder(w.Body).Decode(&c); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if c.Name != "JSON Person" {
		t.Errorf("expected name 'JSON Person', got %s", c.Name)
	}
}

func TestListContacts(t *testing.T) {
	h, token := setupTestHandler(t)

	// Create two contacts
	for _, name := range []string{"Alice", "Bob"} {
		form := url.Values{"name": {name}}
		req := httptest.NewRequest(http.MethodPost, "/contacts", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		h.handleContacts(w, req)
	}

	// List
	req := httptest.NewRequest(http.MethodGet, "/contacts", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.handleContacts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "name=Alice") {
		t.Errorf("should contain Alice, got: %s", body)
	}
	if !strings.Contains(body, "name=Bob") {
		t.Errorf("should contain Bob, got: %s", body)
	}
}

func TestListContactsEmpty(t *testing.T) {
	h, token := setupTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/contacts", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.handleContacts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "no contacts found") {
		t.Errorf("should say no contacts found, got: %s", body)
	}
}

func TestGetContactNotFound(t *testing.T) {
	h, token := setupTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/contacts/contact_nonexist", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.handleContact(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestUpdateContact(t *testing.T) {
	h, token := setupTestHandler(t)

	// Create a contact
	form := url.Values{"name": {"Jane"}, "email": {"jane@old.com"}}
	req := httptest.NewRequest(http.MethodPost, "/contacts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.handleContacts(w, req)

	// Extract handle from response
	body := w.Body.String()
	handleStart := strings.Index(body, "handle=") + 7
	handleEnd := strings.Index(body[handleStart:], " ")
	handle := body[handleStart : handleStart+handleEnd]

	// Update
	form = url.Values{"email": {"jane@new.com"}, "title": {"CTO"}}
	req = httptest.NewRequest(http.MethodPut, "/contacts/"+handle, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	h.handleContact(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body = w.Body.String()
	if !strings.Contains(body, "email=jane@new.com") {
		t.Errorf("should have updated email, got: %s", body)
	}
	if !strings.Contains(body, "title=CTO") {
		t.Errorf("should have updated title, got: %s", body)
	}
	if !strings.Contains(body, "name=Jane") {
		t.Errorf("should still have original name, got: %s", body)
	}
}

func TestDeleteContact(t *testing.T) {
	h, token := setupTestHandler(t)

	// Create a contact
	form := url.Values{"name": {"ToDelete"}}
	req := httptest.NewRequest(http.MethodPost, "/contacts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.handleContacts(w, req)

	body := w.Body.String()
	handleStart := strings.Index(body, "handle=") + 7
	handleEnd := strings.Index(body[handleStart:], " ")
	handle := body[handleStart : handleStart+handleEnd]

	// Delete
	req = httptest.NewRequest(http.MethodDelete, "/contacts/"+handle, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	h.handleContact(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "deleted") {
		t.Errorf("should say deleted, got: %s", w.Body.String())
	}

	// Verify it's gone
	req = httptest.NewRequest(http.MethodGet, "/contacts/"+handle, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	h.handleContact(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", w.Code)
	}
}

func TestSearchContacts(t *testing.T) {
	h, token := setupTestHandler(t)

	// Create contacts
	form := url.Values{"name": {"Alice Smith"}, "company": {"Acme"}}
	req := httptest.NewRequest(http.MethodPost, "/contacts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.handleContacts(w, req)

	form = url.Values{"name": {"Bob Jones"}, "company": {"Globex"}}
	req = httptest.NewRequest(http.MethodPost, "/contacts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	h.handleContacts(w, req)

	// Search
	req = httptest.NewRequest(http.MethodGet, "/contacts/search?q=acme", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	h.handleSearchContacts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Alice") {
		t.Errorf("should find Alice, got: %s", body)
	}
	if strings.Contains(body, "Bob") {
		t.Errorf("should not find Bob, got: %s", body)
	}
}

func TestSearchContactsMissingQuery(t *testing.T) {
	h, token := setupTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/contacts/search", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.handleSearchContacts(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUnauthorizedRequest(t *testing.T) {
	h, _ := setupTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/contacts", nil)
	w := httptest.NewRecorder()
	h.mw.RequireAuth(func(w http.ResponseWriter, r *http.Request) {})(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "missing auth token") {
		t.Errorf("should say missing auth token, got: %s", body)
	}
	if !strings.Contains(body, "hint:") {
		t.Error("error should include a hint")
	}
}

func TestInvalidToken(t *testing.T) {
	h, _ := setupTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/contacts", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	h.mw.RequireAuth(func(w http.ResponseWriter, r *http.Request) {})(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestWorkspaceInfo(t *testing.T) {
	h, token := setupTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/workspace", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Workspace", "ws_test1")
	w := httptest.NewRecorder()
	h.handleWorkspace(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "handle=ws_test1") {
		t.Errorf("should contain workspace handle, got: %s", body)
	}
}

func TestFullAuthFlow(t *testing.T) {
	h, _ := setupTestHandler(t)

	// Request OTP
	form := url.Values{"email": {"flowtest@example.com"}}
	req := httptest.NewRequest(http.MethodPost, "/auth/request", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.handleAuthRequest(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("auth request failed: %d", w.Code)
	}

	// We can't easily get the OTP from stderr, so let's test the verify
	// with a wrong code to ensure it fails properly
	form = url.Values{"email": {"flowtest@example.com"}, "code": {"000000"}}
	req = httptest.NewRequest(http.MethodPost, "/auth/verify", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	h.handleAuthVerify(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong code, got %d", w.Code)
	}
}

func TestParseTags(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"vip", []string{"vip"}},
		{"vip,prospect", []string{"vip", "prospect"}},
		{"  vip  ,  prospect  ", []string{"vip", "prospect"}},
		{"vip,,prospect", []string{"vip", "prospect"}},
	}
	for _, tt := range tests {
		got := parseTags(tt.input)
		if len(got) != len(tt.expected) {
			t.Errorf("parseTags(%q) = %v, expected %v", tt.input, got, tt.expected)
			continue
		}
		for i, tag := range got {
			if tag != tt.expected[i] {
				t.Errorf("parseTags(%q)[%d] = %q, expected %q", tt.input, i, tag, tt.expected[i])
			}
		}
	}
}
