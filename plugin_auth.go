package proxy

import (
	"errors"
	"fmt"
	"sync"
)

var (
	ErrEmptyAuthBackendName = errors.New("auth backend name is empty")
	ErrNilAuthBackend       = errors.New("auth backend is nil")
	ErrBuiltinAuthBackend   = errors.New("auth backend name collision with builtin")
)

var (
	authBackends     map[string]AuthBackend
	authBackendsMu   sync.Mutex
	authBackendsOnce sync.Once
)

// Auth returns an authentication backend by name or false if it doesn't exist.
func Auth(name string) (AuthBackend, bool) {
	authBackendsMu.Lock()
	defer authBackendsMu.Unlock()

	ab, ok := authBackends[name]
	return ab, ok
}

// RegisterAuthBackend registers a new authentication backend implementation.
// The name must be unique, non-empty and must not collide
// with a builtin authentication backend.
// The authentication backend must be non-nil.
// Registered backends can be enabled by specifying their name
// in the AuthBackend config option.
// Backend-specific configuration is handled by the calling plugin's
// configuration mechanism at initialization time.
// Backends must be registered at initialization time
// (before the init functions return).
// Backends registered after initialization time will not be available
// to the user.
func RegisterAuthBackend(name string, ab AuthBackend) error {
	initAuthBackends()

	if name == "" {
		return ErrEmptyAuthBackendName
	}
	if ab == nil {
		return ErrNilAuthBackend
	}

	if name == "files" || name == "mtsqlite3" || name == "mtpostgresql" {
		return ErrBuiltinAuthBackend
	}

	if _, ok := authBackends[name]; ok {
		return fmt.Errorf("duplicate auth backend %s", name)
	}

	authBackendsMu.Lock()
	defer authBackendsMu.Unlock()

	authBackends[name] = ab
	return nil
}

func initAuthBackends() {
	authBackendsOnce.Do(func() {
		authBackendsMu.Lock()
		defer authBackendsMu.Unlock()

		authBackends = make(map[string]AuthBackend)
	})
}
