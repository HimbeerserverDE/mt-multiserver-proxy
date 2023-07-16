package proxy

import (
	"database/sql"
	"errors"
	"net"
	"time"
)

// A handle to a SQLite3 authentication database.
// The upstream Minetest schema is used.
//
// Table info:
//
// 0|id|INTEGER|0||1
// 1|name|VARCHAR(32)|0||0
// 2|password|VARCHAR(512)|0||0
// 3|last_login|INTEGER|0||0
type AuthMTSQLite3 struct {
	db *sql.DB
}

// NewAuthMTSQLite3 opens the SQLite3 authentication database at auth.sqlite.
func NewAuthMTSQLite3() (*AuthMTSQLite3, error) {
	db, err := sql.Open("sqlite3", "auth.sqlite")
	if err != nil {
		return nil, err
	}

	// Initialize the database if necessary.
	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS auth (id INTEGER PRIMARY KEY AUTOINCREMENT, name VARCHAR(32) UNIQUE, password VARCHAR(512), last_login INTEGER);"); err != nil {
		return nil, err
	}

	return &AuthMTSQLite3{db}, nil
}

// Close closes the underlying SQLite3 database handle.
func (a *AuthMTSQLite3) Close() error {
	return a.db.Close()
}

// Exists reports whether a user is registered.
// Error cases count as inexistent.
func (a *AuthMTSQLite3) Exists(name string) bool {
	result := a.db.QueryRow("SELECT COUNT(1) FROM auth WHERE name = ?;", name)

	var count int
	if err := result.Scan(&count); err != nil {
		return false
	}

	return count == 1
}

// Passwd returns the SRP salt and verifier of a user or an error.
func (a *AuthMTSQLite3) Passwd(name string) (salt, verifier []byte, err error) {
	result := a.db.QueryRow("SELECT password FROM auth WHERE name = ?;", name)

	var encodedPasswd string
	if err = result.Scan(&encodedPasswd); err != nil {
		return
	}

	salt, verifier, err = decodeVerifierAndSalt(encodedPasswd)

	a.updateTimestamp(name)
	return
}

// SetPasswd creates a password entry if necessary
// and sets the password of a user.
func (a *AuthMTSQLite3) SetPasswd(name string, salt, verifier []byte) error {
	encodedPasswd := encodeVerifierAndSalt(salt, verifier)

	_, err := a.db.Exec("REPLACE INTO auth (name, password, last_login) VALUES (?, ?, unixepoch());", name, encodedPasswd)
	return err
}

// LastSrv always returns an error
// since the Minetest database schema cannot store this information.
func (a *AuthMTSQLite3) LastSrv(_ string) (string, error) {
	return "", ErrLastSrvNotSupported
}

// SetLastSrv is a no-op
// since the Minetest database schema cannot store this information.
func (a *AuthMTSQLite3) SetLastSrv(_, _ string) error {
	return nil
}

// Timestamp returns the last time an authentication entry was accessed
// or an error.
func (a *AuthMTSQLite3) Timestamp(name string) (time.Time, error) {
	result := a.db.QueryRow("SELECT last_login FROM auth WHERE name = ?;", name)

	var timestamp int64
	if err := result.Scan(&timestamp); err != nil {
		return time.Time{}, err
	}

	return time.Unix(timestamp, 0), nil
}

// Import adds the passed users.
func (a *AuthMTSQLite3) Import(in []User) error {
	for _, u := range in {
		if err := a.SetPasswd(u.Name, u.Salt, u.Verifier); err != nil {
			return err
		}

		a.setTimestamp(u.Name, u.Timestamp)
	}

	return nil
}

// Export returns data that can be processed by Import
// or an error.
func (a *AuthMTSQLite3) Export() ([]User, error) {
	var names []string
	result := a.db.QueryRow("SELECT name FROM auth;")

	for {
		var name string
		if err := result.Scan(&name); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				break
			}

			return nil, err
		}

		names = append(names, name)
	}

	var out []User
	for _, name := range names {
		u := User{Name: name}

		var err error
		u.Timestamp, err = a.Timestamp(u.Name)
		if err != nil {
			return nil, err
		}

		u.Salt, u.Verifier, err = a.Passwd(u.Name)
		if err != nil {
			return nil, err
		}

		out = append(out, u)
	}

	return out, nil
}

// Ban always returns an error
// since the Minetest database schema cannot store this information.
func (a *AuthMTSQLite3) Ban(_, _ string) error {
	return ErrBanNotSupported
}

// Unban always returns an error
// since the Minetest database schema cannot store this information.
func (a *AuthMTSQLite3) Unban(_ string) error {
	return ErrBanNotSupported
}

// Banned always reports that the user is not banned
// since the Minetest database schema cannot store this information.
func (a *AuthMTSQLite3) Banned(_ *net.UDPAddr) bool {
	return false
}

// ImportBans always returns an error
// since the Minetest database schema cannot store this information.
func (a *AuthMTSQLite3) ImportBans(in []Ban) error {
	return ErrBanNotSupported
}

// ExportBans always returns an empty list of ban entries
// since the Minetest database schema cannot store this information.
func (a *AuthMTSQLite3) ExportBans() ([]Ban, error) {
	return []Ban{}, nil
}

func (a *AuthMTSQLite3) setTimestamp(name string, t time.Time) {
	timestamp := t.Unix()
	a.db.Exec("UPDATE auth SET last_login = ? WHERE name = ?;", timestamp, name)
}

func (a *AuthMTSQLite3) updateTimestamp(name string) {
	a.setTimestamp(name, time.Now().Local())
}
