package autocert

import (
	"golang.org/x/crypto/acme/autocert"
)

// NewManager creates and configures a new autocert manager.
func NewManager(domains []string, cacheDir string) *autocert.Manager {
	return &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(domains...),
		Cache:      autocert.DirCache(cacheDir),
	}
}
