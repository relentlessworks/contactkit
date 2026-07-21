package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/relentlessworks/contactkit/internal/model"
)

// wantsJSON checks if the client wants JSON response.
func wantsJSON(r *http.Request) bool {
	if r.URL.Query().Get("format") == "json" {
		return true
	}
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "application/json")
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeText writes a plain text response.
func writeText(w http.ResponseWriter, status int, s string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprintln(w, s)
}

// writeError writes an error response in the appropriate format.
func writeError(w http.ResponseWriter, r *http.Request, status int, msg, hint string) {
	if wantsJSON(r) {
		writeJSON(w, status, map[string]string{
			"error": msg,
			"hint":  hint,
		})
		return
	}
	writeText(w, status, fmt.Sprintf("error: %s | hint: %s", msg, hint))
}

// writeOK writes a success response.
func writeOK(w http.ResponseWriter, r *http.Request, msg string) {
	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]string{"ok": msg})
		return
	}
	writeText(w, http.StatusOK, msg)
}

// formatContact formats a contact as a single grepable line.
func formatContact(c *model.Contact) string {
	parts := []string{
		"handle=" + c.Handle,
		"name=" + c.Name,
	}
	if c.Email != "" {
		parts = append(parts, "email="+c.Email)
	}
	if c.Phone != "" {
		parts = append(parts, "phone="+c.Phone)
	}
	if c.Company != "" {
		parts = append(parts, "company="+c.Company)
	}
	if c.Title != "" {
		parts = append(parts, "title="+c.Title)
	}
	if len(c.Tags) > 0 {
		parts = append(parts, "tags="+strings.Join(c.Tags, ","))
	}
	parts = append(parts, "created="+c.CreatedAt.Format("2006-01-02T15:04:05Z"))
	parts = append(parts, "updated="+c.UpdatedAt.Format("2006-01-02T15:04:05Z"))
	return strings.Join(parts, " ")
}

// formatWorkspace formats a workspace as a single grepable line.
func formatWorkspace(ws *model.Workspace) string {
	return fmt.Sprintf("handle=%s name=%s plan=%s created=%s",
		ws.Handle, ws.Name, ws.Plan, ws.CreatedAt.Format("2006-01-02T15:04:05Z"))
}
