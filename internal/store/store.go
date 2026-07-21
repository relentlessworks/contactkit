package store

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/relentlessworks/contactkit/internal/model"
)

// Store manages all data with a JSON file backend.
type Store struct {
	mu       sync.RWMutex
	filePath string
	data     *storeData
}

type storeData struct {
	Workspaces map[string]*model.Workspace `json:"workspaces"`
	Contacts   map[string]map[string]*model.Contact `json:"contacts"` // workspace -> handle -> contact
	Tokens     map[string]*model.Token     `json:"tokens"`
	OTPs       map[string]*model.OTP       `json:"otps"`
}

// New creates a new store backed by a JSON file.
func New(filePath string) (*Store, error) {
	s := &Store{
		filePath: filePath,
		data: &storeData{
			Workspaces: make(map[string]*model.Workspace),
			Contacts:   make(map[string]map[string]*model.Contact),
			Tokens:     make(map[string]*model.Token),
			OTPs:       make(map[string]*model.OTP),
		},
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	b, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // fresh start
		}
		return err
	}
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, s.data)
}

func (s *Store) save() error {
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, b, 0600)
}

// --- Workspace operations ---

func (s *Store) CreateWorkspace(ws *model.Workspace) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Workspaces[ws.Handle] = ws
	s.data.Contacts[ws.Handle] = make(map[string]*model.Contact)
	return s.save()
}

func (s *Store) GetWorkspace(handle string) (*model.Workspace, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ws, ok := s.data.Workspaces[handle]
	return ws, ok
}

func (s *Store) ListWorkspaces() []*model.Workspace {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*model.Workspace, 0, len(s.data.Workspaces))
	for _, ws := range s.data.Workspaces {
		result = append(result, ws)
	}
	return result
}

// --- Contact operations ---

func (s *Store) CreateContact(wsHandle string, c *model.Contact) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data.Contacts[wsHandle]; !ok {
		s.data.Contacts[wsHandle] = make(map[string]*model.Contact)
	}
	s.data.Contacts[wsHandle][c.Handle] = c
	return s.save()
}

func (s *Store) GetContact(wsHandle, handle string) (*model.Contact, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if contacts, ok := s.data.Contacts[wsHandle]; ok {
		c, ok := contacts[handle]
		return c, ok
	}
	return nil, false
}

func (s *Store) UpdateContact(wsHandle string, c *model.Contact) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if contacts, ok := s.data.Contacts[wsHandle]; ok {
		if _, ok := contacts[c.Handle]; ok {
			c.UpdatedAt = time.Now()
			contacts[c.Handle] = c
			return s.save()
		}
	}
	return ErrNotFound
}

func (s *Store) DeleteContact(wsHandle, handle string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if contacts, ok := s.data.Contacts[wsHandle]; ok {
		if _, ok := contacts[handle]; ok {
			delete(contacts, handle)
			return s.save()
		}
	}
	return ErrNotFound
}

func (s *Store) ListContacts(wsHandle string) []*model.Contact {
	s.mu.RLock()
	defer s.mu.RUnlock()
	contacts, ok := s.data.Contacts[wsHandle]
	if !ok {
		return nil
	}
	result := make([]*model.Contact, 0, len(contacts))
	for _, c := range contacts {
		result = append(result, c)
	}
	return result
}

func (s *Store) SearchContacts(wsHandle, query string) []*model.Contact {
	s.mu.RLock()
	defer s.mu.RUnlock()
	contacts, ok := s.data.Contacts[wsHandle]
	if !ok {
		return nil
	}
	query = toLower(query)
	var result []*model.Contact
	for _, c := range contacts {
		if contains(toLower(c.Name), query) ||
			contains(toLower(c.Email), query) ||
			contains(toLower(c.Company), query) ||
			contains(toLower(c.Phone), query) ||
			hasTag(c.Tags, query) {
			result = append(result, c)
		}
	}
	return result
}

// --- Token operations ---

func (s *Store) SaveToken(t *model.Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Tokens[t.Token] = t
	return s.save()
}

func (s *Store) GetToken(token string) (*model.Token, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.data.Tokens[token]
	return t, ok
}

func (s *Store) DeleteToken(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data.Tokens, token)
	return s.save()
}

// --- OTP operations ---

func (s *Store) SaveOTP(otp *model.OTP) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.OTPs[otp.Email] = otp
	return s.save()
}

func (s *Store) GetOTP(email string) (*model.OTP, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	otp, ok := s.data.OTPs[email]
	return otp, ok
}

func (s *Store) DeleteOTP(email string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data.OTPs, email)
	return s.save()
}
