package autocert

import (
	"sync"

	"golang.org/x/crypto/acme/autocert"
)

// Manager wraps autocert.Manager to support dynamic domain updates
type Manager struct {
	*autocert.Manager
	mu       sync.RWMutex
	domains  []string
	cacheDir string
}

// NewManager creates and configures a new autocert manager.
func NewManager(domains []string, cacheDir string) *Manager {
	manager := &Manager{
		domains:  domains,
		cacheDir: cacheDir,
	}
	manager.Manager = &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(domains...),
		Cache:      autocert.DirCache(cacheDir),
	}
	return manager
}

// UpdateDomains updates the allowed domains for certificate generation
func (m *Manager) UpdateDomains(domains []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.domains = domains
	// Create a new autocert manager with updated domains
	m.Manager = &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(domains...),
		Cache:      autocert.DirCache(m.cacheDir),
	}
}

// GetDomains returns the current list of allowed domains
func (m *Manager) GetDomains() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]string(nil), m.domains...)
}
