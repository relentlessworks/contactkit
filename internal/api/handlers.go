package api

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/relentlessworks/contactkit/internal/auth"
	"github.com/relentlessworks/contactkit/internal/model"
	"github.com/relentlessworks/contactkit/internal/store"
)

// Handler holds dependencies for HTTP handlers.
type Handler struct {
	store *store.Store
	auth  *auth.Auth
	mw    *Middleware
}

// NewHandler creates a new API handler.
func NewHandler(s *store.Store, a *auth.Auth) *Handler {
	h := &Handler{
		store: s,
		auth:  a,
	}
	h.mw = NewMiddleware(a)
	return h
}

// Routes returns the HTTP mux with all routes registered.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/help", h.handleHelp)
	mux.HandleFunc("/.well-known/agent.md", h.handleHelp)
	mux.HandleFunc("/auth/request", h.handleAuthRequest)
	mux.HandleFunc("/auth/verify", h.handleAuthVerify)

	// Protected routes
	mux.HandleFunc("/contacts", h.mw.RequireAuth(h.handleContacts))
	mux.HandleFunc("/contacts/", h.mw.RequireAuth(h.handleContact))
	mux.HandleFunc("/contacts/search", h.mw.RequireAuth(h.handleSearchContacts))
	mux.HandleFunc("/workspace", h.mw.RequireAuth(h.handleWorkspace))
	mux.HandleFunc("/auth/revoke", h.mw.RequireAuth(h.handleAuthRevoke))

	return mux
}

// --- Help ---

func (h *Handler) handleHelp(w http.ResponseWriter, r *http.Request) {
	help := `ContactKit — Agentic-First Contact Management & CRM

DESCRIPTION
  ContactKit is a headless CRM service designed for AI agents.
  Manage contacts, search, tag, and organize your network over plain HTTP.

AUTH FLOW
  1. POST /auth/request   body: email=user@example.com
     Sends a 6-digit OTP to the email (or logs to stderr in dev mode).
  2. POST /auth/verify    body: email=user@example.com code=123456
     Returns: token=BASE32TOKEN workspace=ws_abc12
  3. Use token in all subsequent requests: Authorization: Bearer TOKEN

ENDPOINTS

  POST /contacts
    Create a contact. Body params: name (required), email, phone, company, title, notes, tags (comma-separated)
    Returns: handle=contact_xxxxx name=... email=... company=...
    Example: curl -X POST -H "Authorization: Bearer TOKEN" -d "name=Jane Doe&email=jane@acme.com&company=Acme&tags=vip,prospect" http://localhost:7700/contacts

  GET /contacts
    List all contacts in your workspace.
    Returns one line per contact: handle=... name=... email=... company=...

  GET /contacts/{handle}
    Get a single contact by handle.
    Returns: handle=contact_xxxxx name=... email=... phone=... company=... title=... tags=... created=... updated=...

  PUT /contacts/{handle}
    Update a contact. Body params: name, email, phone, company, title, notes, tags (comma-separated)
    Only provided fields are updated.
    Returns: handle=contact_xxxxx name=... updated=...

  DELETE /contacts/{handle}
    Delete a contact.
    Returns: ok: contact contact_xxxxx deleted

  GET /contacts/search?q=query
    Search contacts by name, email, company, phone, or tags.
    Returns matching contacts, one per line.

  GET /workspace
    Get your workspace info.
    Returns: handle=ws_xxxxx name=... plan=...

  POST /auth/revoke
    Revoke your current token.
    Returns: ok: token revoked

RESPONSE FORMAT
  Plain text by default (one labeled line per record).
  JSON available via Accept: application/json or ?format=json.
  Errors include a hint: error: message | hint: what to do next

CONFIG
  -addr    Listen address (default :7700, env CONTACTKIT_ADDR)
  -data    Data file path (default ./contactkit-data.json, env CONTACTKIT_DATA)
  -secret  Token signing secret (auto-generated, env CONTACTKIT_SECRET)
  -smtp    SMTP server for OTP emails (empty = log to stderr, env CONTACTKIT_SMTP)
`
	writeText(w, http.StatusOK, strings.TrimSpace(help))
}

// --- Auth ---

func (h *Handler) handleAuthRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed", "use POST to request an OTP")
		return
	}

	email := r.FormValue("email")
	if email == "" {
		writeError(w, r, http.StatusBadRequest, "missing email", "provide email parameter: POST /auth/request with body email=user@example.com")
		return
	}

	if err := h.auth.RequestOTP(email); err != nil {
		writeError(w, r, http.StatusInternalServerError, "failed to send OTP", "check that the email is valid and SMTP is configured (or run without -smtp for dev mode)")
		return
	}

	writeOK(w, r, "OTP sent to "+email+" (check stderr if no SMTP configured)")
}

func (h *Handler) handleAuthVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed", "use POST to verify an OTP")
		return
	}

	email := r.FormValue("email")
	code := r.FormValue("code")
	if email == "" || code == "" {
		writeError(w, r, http.StatusBadRequest, "missing email or code", "provide both email and code: POST /auth/verify with body email=user@example.com code=123456")
		return
	}

	token, err := h.auth.VerifyOTP(email, code)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, "verification failed: "+err.Error(), "request a new OTP via POST /auth/request with email, then verify with the new code")
		return
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]string{
			"token":     token.Token,
			"workspace": token.Workspace,
			"email":     token.Email,
		})
		return
	}
	writeText(w, http.StatusOK, fmt.Sprintf("token=%s workspace=%s email=%s", token.Token, token.Workspace, token.Email))
}

func (h *Handler) handleAuthRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed", "use POST to revoke your token")
		return
	}

	token := extractToken(r)
	if err := h.auth.RevokeToken(token); err != nil {
		writeError(w, r, http.StatusInternalServerError, "failed to revoke token", "try again or the token may already be invalid")
		return
	}

	writeOK(w, r, "token revoked")
}

// --- Contacts ---

func (h *Handler) handleContacts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.handleCreateContact(w, r)
	case http.MethodGet:
		h.handleListContacts(w, r)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed", "use POST to create a contact or GET to list contacts")
	}
}

func (h *Handler) handleCreateContact(w http.ResponseWriter, r *http.Request) {
	ws := getWorkspace(r)

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		writeError(w, r, http.StatusBadRequest, "missing name", "provide at least a name: POST /contacts with body name=Jane Doe&email=jane@acme.com")
		return
	}

	handle, err := model.GenerateHandle()
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "failed to generate handle", "try again")
		return
	}

	tags := parseTags(r.FormValue("tags"))

	c := &model.Contact{
		Handle:    handle,
		Name:      name,
		Email:     strings.TrimSpace(r.FormValue("email")),
		Phone:     strings.TrimSpace(r.FormValue("phone")),
		Company:   strings.TrimSpace(r.FormValue("company")),
		Title:     strings.TrimSpace(r.FormValue("title")),
		Notes:     strings.TrimSpace(r.FormValue("notes")),
		Tags:      tags,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.store.CreateContact(ws, c); err != nil {
		writeError(w, r, http.StatusInternalServerError, "failed to create contact", "try again or check your input")
		return
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusCreated, c)
		return
	}
	writeText(w, http.StatusCreated, formatContact(c))
}

func (h *Handler) handleListContacts(w http.ResponseWriter, r *http.Request) {
	ws := getWorkspace(r)
	contacts := h.store.ListContacts(ws)

	// Sort by created time (newest first)
	sort.Slice(contacts, func(i, j int) bool {
		return contacts[i].CreatedAt.After(contacts[j].CreatedAt)
	})

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, contacts)
		return
	}

	if len(contacts) == 0 {
		writeText(w, http.StatusOK, "no contacts found")
		return
	}

	var lines []string
	for _, c := range contacts {
		lines = append(lines, formatContact(c))
	}
	writeText(w, http.StatusOK, strings.Join(lines, "\n"))
}

func (h *Handler) handleContact(w http.ResponseWriter, r *http.Request) {
	handle := strings.TrimPrefix(r.URL.Path, "/contacts/")
	if handle == "" {
		writeError(w, r, http.StatusBadRequest, "missing contact handle", "provide a handle: GET /contacts/contact_xxxxx")
		return
	}

	// Handle search path
	if handle == "search" {
		h.handleSearchContacts(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGetContact(w, r, handle)
	case http.MethodPut:
		h.handleUpdateContact(w, r, handle)
	case http.MethodDelete:
		h.handleDeleteContact(w, r, handle)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed", "use GET to view, PUT to update, or DELETE to remove a contact")
	}
}

func (h *Handler) handleGetContact(w http.ResponseWriter, r *http.Request, handle string) {
	ws := getWorkspace(r)
	c, ok := h.store.GetContact(ws, handle)
	if !ok {
		writeError(w, r, http.StatusNotFound, "contact not found", "check the handle or call GET /contacts to list all contacts")
		return
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, c)
		return
	}
	writeText(w, http.StatusOK, formatContact(c))
}

func (h *Handler) handleUpdateContact(w http.ResponseWriter, r *http.Request, handle string) {
	ws := getWorkspace(r)
	c, ok := h.store.GetContact(ws, handle)
	if !ok {
		writeError(w, r, http.StatusNotFound, "contact not found", "check the handle or call GET /contacts to list all contacts")
		return
	}

	// Update only provided fields
	if v := strings.TrimSpace(r.FormValue("name")); v != "" {
		c.Name = v
	}
	if v := strings.TrimSpace(r.FormValue("email")); v != "" {
		c.Email = v
	}
	if v := strings.TrimSpace(r.FormValue("phone")); v != "" {
		c.Phone = v
	}
	if v := strings.TrimSpace(r.FormValue("company")); v != "" {
		c.Company = v
	}
	if v := strings.TrimSpace(r.FormValue("title")); v != "" {
		c.Title = v
	}
	if v := strings.TrimSpace(r.FormValue("notes")); v != "" {
		c.Notes = v
	}
	if v := r.FormValue("tags"); v != "" {
		c.Tags = parseTags(v)
	}

	if err := h.store.UpdateContact(ws, c); err != nil {
		writeError(w, r, http.StatusInternalServerError, "failed to update contact", "try again or check the handle")
		return
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, c)
		return
	}
	writeText(w, http.StatusOK, formatContact(c))
}

func (h *Handler) handleDeleteContact(w http.ResponseWriter, r *http.Request, handle string) {
	ws := getWorkspace(r)
	_, ok := h.store.GetContact(ws, handle)
	if !ok {
		writeError(w, r, http.StatusNotFound, "contact not found", "check the handle or call GET /contacts to list all contacts")
		return
	}

	if err := h.store.DeleteContact(ws, handle); err != nil {
		writeError(w, r, http.StatusInternalServerError, "failed to delete contact", "try again")
		return
	}

	writeOK(w, r, "contact "+handle+" deleted")
}

func (h *Handler) handleSearchContacts(w http.ResponseWriter, r *http.Request) {
	ws := getWorkspace(r)
	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, r, http.StatusBadRequest, "missing search query", "provide a query: GET /contacts/search?q=jane")
		return
	}

	contacts := h.store.SearchContacts(ws, q)

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, contacts)
		return
	}

	if len(contacts) == 0 {
		writeText(w, http.StatusOK, "no contacts found matching: "+q)
		return
	}

	var lines []string
	for _, c := range contacts {
		lines = append(lines, formatContact(c))
	}
	writeText(w, http.StatusOK, strings.Join(lines, "\n"))
}

// --- Workspace ---

func (h *Handler) handleWorkspace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed", "use GET to view your workspace")
		return
	}

	ws := getWorkspace(r)
	wsp, ok := h.store.GetWorkspace(ws)
	if !ok {
		writeError(w, r, http.StatusNotFound, "workspace not found", "your token may be invalid. Re-authenticate via POST /auth/request")
		return
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, wsp)
		return
	}
	writeText(w, http.StatusOK, formatWorkspace(wsp))
}

// --- Helpers ---

func parseTags(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var tags []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			tags = append(tags, p)
		}
	}
	return tags
}
