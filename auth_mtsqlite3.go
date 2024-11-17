package proxy

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
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
	db, err := sql.Open("sqlite3", Path("auth.sqlite"))
	if err != nil {
		return nil, err
	}

	// Initialize the database if necessary.
	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS auth (id INTEGER PRIMARY KEY AUTOINCREMENT, name VARCHAR(32) UNIQUE, password VARCHAR(512), last_login INTEGER);"); err != nil {
		db.Close()
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

	salt, verifier, err = DecodeVerifierAndSalt(encodedPasswd)

	a.updateTimestamp(name)
	return
}

// SetPasswd creates a password entry if necessary
// and sets the password of a user.
func (a *AuthMTSQLite3) SetPasswd(name string, salt, verifier []byte) error {
	encodedPasswd := EncodeVerifierAndSalt(salt, verifier)

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

	result, err := a.db.Query("SELECT name FROM auth;")
	if err != nil {
		return nil, err
	}

	for result.Next() {
		var name string
		if err := result.Scan(&name); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				break
			}

			return nil, err
		}

		names = append(names, name)
	}

	if err := result.Err(); err != nil {
		return nil, err
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

// Ban adds a ban entry for a network address and an associated name.
func (a *AuthMTSQLite3) Ban(addr, name string) error {
	bans, err := a.readBans()
	if err != nil {
		return err
	}

	bans[addr] = name
	return a.writeBans(bans)
}

// Unban deletes a ban entry. It accepts bot network addresses
// and player names.
func (a *AuthMTSQLite3) Unban(id string) error {
	bans, err := a.readBans()
	if err != nil {
		return err
	}

	delete(bans, id)

	var flagged []string
	for addr, name := range bans {
		if name == id {
			flagged = append(flagged, addr)
		}
	}

	for _, addr := range flagged {
		delete(bans, addr)
	}

	return a.writeBans(bans)
}

// Banned reports whether a network address is banned.
// Error cases count as banned.
func (a *AuthMTSQLite3) Banned(addr, name string) bool {
	bans, err := a.readBans()
	if err != nil {
		return true
	}

	_, ok := bans[addr]
	return ok
}

// RecordFail is a no-op.
func (a *AuthMTSQLite3) RecordFail(addr, name string, sudo bool) error {
	return nil
}

// ImportBans adds the passed entries.
func (a *AuthMTSQLite3) ImportBans(in []Ban) error {
	for _, b := range in {
		if err := a.Ban(b.Addr, b.Name); err != nil {
			return err
		}
	}

	return nil
}

// ExportBans returns data that can be processed by ImportBans
// or an error.
func (a *AuthMTSQLite3) ExportBans() ([]Ban, error) {
	bans, err := a.readBans()
	if err != nil {
		return nil, err
	}

	var unmapped []Ban
	for addr, name := range bans {
		unmapped = append(unmapped, Ban{
			Addr: addr,
			Name: name,
		})
	}

	return unmapped, nil
}

func (a *AuthMTSQLite3) setTimestamp(name string, t time.Time) {
	timestamp := t.Unix()
	a.db.Exec("UPDATE auth SET last_login = ? WHERE name = ?;", timestamp, name)
}

func (a *AuthMTSQLite3) updateTimestamp(name string) {
	a.setTimestamp(name, time.Now().Local())
}

func (a *AuthMTSQLite3) readBans() (map[string]string, error) {
	f, err := os.OpenFile(Path("ipban.txt"), os.O_RDONLY|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	bans := make(map[string]string)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		ban := scanner.Text()

		addr := strings.Split(ban, "|")[0]
		name := strings.Split(ban, "|")[1]

		bans[addr] = name
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return bans, nil
}

func (a *AuthMTSQLite3) writeBans(bans map[string]string) error {
	f, err := os.OpenFile(Path("ipban.txt"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	for addr, name := range bans {
		if _, err := fmt.Fprintln(f, addr+"|"+name); err != nil {
			return err
		}
	}

	return nil
}
