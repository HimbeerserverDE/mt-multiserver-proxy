package proxy

import (
	"errors"
	"strings"
	"time"
)

var authIface AuthBackend

var (
	ErrAuthBackendExists   = errors.New("auth backend already set")
	ErrInvalidSRPHeader    = errors.New("encoded password is not SRP")
	ErrLastSrvNotSupported = errors.New("auth backend does not support server information")
)

type User struct {
	Name      string
	Salt      []byte
	Verifier  []byte
	Timestamp time.Time
}

type Ban struct {
	Addr string
	Name string
}

// An AuthBackend provides authentication and moderation functionality.
// This typically includes persistent storage.
// It does not handle authorization, i.e. permission checks.
// All methods are safe for concurrent use.
type AuthBackend interface {
	// Exists reports whether a user with the specified name exists.
	// The result is false if an error is encountered.
	Exists(name string) bool
	// Passwd returns the SRP verifier and salt of the user
	// with the specified name or an error.
	Passwd(name string) (salt, verifier []byte, err error)
	// SetPasswd sets the SRP verifier and salt of the user
	// with the specified name.
	SetPasswd(name string, salt, verifier []byte) error
	// LastSrv returns the name of the last server the user
	// with the specified name intentionally connected to or an error.
	// This method should return an error if this feature is unsupported.
	// Errors are handled gracefully (by connecting the user
	// to the default server or group) and aren't logged.
	LastSrv(name string) (string, error)
	// SetLastSrv sets the name of the last server the user
	// with the specified name intentionally connected to.
	// This method should not return an error if this feature is unsupported.
	// Errors will make server hopping fail.
	SetLastSrv(name, srv string) error
	// Timestamp returns the last time the user with the specified name
	// connected to the proxy or an error.
	Timestamp(name string) (time.Time, error)
	// Import adds or modifies authentication entries in bulk.
	Import(in []User) error
	// Export returns all authentication entries or an error.
	Export() ([]User, error)

	// Ban adds a ban entry for a network address and an associated name.
	// Only the specified network address is banned from connecting.
	// Existing connections are not kicked.
	Ban(addr, name string) error
	// Unban deletes a ban entry by network address or username.
	Unban(id string) error
	// Banned reports whether a network address or username is banned.
	// The result is true if either identifier is banned
	// or if an error is encountered.
	Banned(addr, name string) bool
	// RecordFail records an authentication failure regarding a certain
	// network address and username. The implementation is not required
	// to process this event in any way, but the intent is to allow
	// rate limiting / brute-force protection to be implemented by plugins.
	RecordFail(addr, name string, sudo bool) error
	// ImportBans adds or modifies ban entries in bulk.
	ImportBans(in []Ban) error
	// Export returns all ban entries or an error.
	ExportBans() ([]Ban, error)
}

// DefaultAuth returns the authentication backend that is currently in use
// or nil during initialization time.
func DefaultAuth() AuthBackend {
	return authIface
}

func setAuthBackend(ab AuthBackend) error {
	if authIface != nil {
		return ErrAuthBackendExists
	}

	authIface = ab
	return nil
}

func EncodeVerifierAndSalt(salt, verifier []byte) string {
	return "#1#" + b64.EncodeToString(salt) + "#" + b64.EncodeToString(verifier)
}

func DecodeVerifierAndSalt(encodedPasswd string) ([]byte, []byte, error) {
	if !strings.HasPrefix(encodedPasswd, "#1#") {
		return nil, nil, ErrInvalidSRPHeader
	}

	salt, err := b64.DecodeString(strings.Split(encodedPasswd, "#")[2])
	if err != nil {
		return nil, nil, err
	}

	verifier, err := b64.DecodeString(strings.Split(encodedPasswd, "#")[3])
	if err != nil {
		return nil, nil, err
	}

	return salt, verifier, nil
}
