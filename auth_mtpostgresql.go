package proxy

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// A handle to a PostgreSQL authentication database.
// The upstream Minetest schema is used.
//
// Table info:
//
//	                           Table "public.auth"
//	Column   |  Type   | Collation | Nullable |              Default
//
// ------------+---------+-----------+----------+------------------------------------
//
//	id         | integer |           | not null | nextval('auth_id_seq'::regclass)
//	name       | text    |           |          |
//	password   | text    |           |          |
//	last_login | integer |           | not null | 0
//
// Indexes:
//
//	"auth_pkey" PRIMARY KEY, btree (id)
//	"auth_name_key" UNIQUE CONSTRAINT, btree (name)
//
// Sequence info:
//
//	                 Sequence "public.auth_id_seq"
//	Type   | Start | Minimum |  Maximum   | Increment | Cycles? | Cache
//
// ---------+-------+---------+------------+-----------+---------+-------
//
//	integer |     1 |       1 | 2147483647 |         1 | no      |     1
//
// Owned by: public.auth.id
type AuthMTPostgreSQL struct {
	db *sql.DB
}

// NewAuthMTPostgreSQL opens the PostgreSQL authentication database
// at the specified connection string.
func NewAuthMTPostgreSQL(conn string) (*AuthMTPostgreSQL, error) {
	db, err := sql.Open("postgres", conn)
	if err != nil {
		return nil, err
	}

	// Check if the connection is working.
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	// Initialize the database if necessary.

	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS public.auth (id integer NOT NULL, name text, password text, last_login integer DEFAULT 0 NOT NULL);"); err != nil {
		db.Close()
		return nil, err
	}

	if _, err := db.Exec("CREATE SEQUENCE IF NOT EXISTS public.auth_id_seq AS integer START WITH 1 INCREMENT BY 1 NO MINVALUE NO MAXVALUE CACHE 1;"); err != nil {
		db.Close()
		return nil, err
	}

	if _, err := db.Exec("ALTER SEQUENCE public.auth_id_seq OWNED BY public.auth.id;"); err != nil {
		db.Close()
		return nil, err
	}

	if _, err := db.Exec("ALTER TABLE ONLY public.auth ALTER COLUMN id SET DEFAULT nextval('public.auth_id_seq'::regclass);"); err != nil {
		db.Close()
		return nil, err
	}

	if _, err := db.Exec("BEGIN TRANSACTION; ALTER TABLE ONLY public.auth DROP CONSTRAINT IF EXISTS auth_pkey; ALTER TABLE ONLY public.auth ADD CONSTRAINT auth_pkey PRIMARY KEY (id); COMMIT;"); err != nil {
		db.Close()
		return nil, err
	}

	if _, err := db.Exec("BEGIN TRANSACTION; ALTER TABLE ONLY public.auth DROP CONSTRAINT IF EXISTS auth_name_key; ALTER TABLE ONLY public.auth ADD CONSTRAINT auth_name_key UNIQUE (name); COMMIT;"); err != nil {
		db.Close()
		return nil, err
	}

	return &AuthMTPostgreSQL{db}, nil
}

// Close closes the underlying PostgreSQL database handle.
func (a *AuthMTPostgreSQL) Close() error {
	return a.db.Close()
}

// Exists reports whether a user is registered.
// Error cases count as inexistent.
func (a *AuthMTPostgreSQL) Exists(name string) bool {
	result := a.db.QueryRow("SELECT COUNT(1) FROM auth WHERE name = $1;", name)

	var count int
	if err := result.Scan(&count); err != nil {
		return false
	}

	return count == 1
}

// Passwd returns the SRP salt and verifier of a user or an error.
func (a *AuthMTPostgreSQL) Passwd(name string) (salt, verifier []byte, err error) {
	result := a.db.QueryRow("SELECT password FROM auth WHERE name = $1;", name)

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
func (a *AuthMTPostgreSQL) SetPasswd(name string, salt, verifier []byte) error {
	encodedPasswd := EncodeVerifierAndSalt(salt, verifier)

	_, err := a.db.Exec("INSERT INTO auth (name, password, last_login) VALUES ($1, $2, extract(epoch from now())) ON CONFLICT (name) DO UPDATE SET password = EXCLUDED.password, last_login = extract(epoch from now());", name, encodedPasswd)
	return err
}

// LastSrv always returns an error
// since the Minetest database schema cannot store this information.
func (a *AuthMTPostgreSQL) LastSrv(_ string) (string, error) {
	return "", ErrLastSrvNotSupported
}

// SetLastSrv is a no-op
// since the Minetest database schema cannot store this information.
func (a *AuthMTPostgreSQL) SetLastSrv(_, _ string) error {
	return nil
}

// Timestamp returns the last time an authentication entry was accessed
// or an error.
func (a *AuthMTPostgreSQL) Timestamp(name string) (time.Time, error) {
	result := a.db.QueryRow("SELECT last_login FROM auth WHERE name = $1;", name)

	var timestamp int64
	if err := result.Scan(&timestamp); err != nil {
		return time.Time{}, err
	}

	return time.Unix(timestamp, 0), nil
}

// Import adds the passed users.
func (a *AuthMTPostgreSQL) Import(in []User) error {
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
func (a *AuthMTPostgreSQL) Export() ([]User, error) {
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
func (a *AuthMTPostgreSQL) Ban(addr, name string) error {
	bans, err := a.readBans()
	if err != nil {
		return err
	}

	bans[addr] = name
	return a.writeBans(bans)
}

// Unban deletes a ban entry. It accepts bot network addresses
// and player names.
func (a *AuthMTPostgreSQL) Unban(id string) error {
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
func (a *AuthMTPostgreSQL) Banned(addr, name string) bool {
	bans, err := a.readBans()
	if err != nil {
		return true
	}

	_, ok := bans[addr]
	return ok
}

// RecordFail is a no-op.
func (a *AuthMTPostgreSQL) RecordFail(addr, name string, sudo bool) error {
	return nil
}

// ImportBans adds the passed entries.
func (a *AuthMTPostgreSQL) ImportBans(in []Ban) error {
	for _, b := range in {
		if err := a.Ban(b.Addr, b.Name); err != nil {
			return err
		}
	}

	return nil
}

// ExportBans returns data that can be processed by ImportBans
// or an error.
func (a *AuthMTPostgreSQL) ExportBans() ([]Ban, error) {
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

func (a *AuthMTPostgreSQL) setTimestamp(name string, t time.Time) {
	timestamp := t.Unix()
	a.db.Exec("UPDATE auth SET last_login = $1 WHERE name = $2;", timestamp, name)
}

func (a *AuthMTPostgreSQL) updateTimestamp(name string) {
	a.setTimestamp(name, time.Now().Local())
}

func (a *AuthMTPostgreSQL) readBans() (map[string]string, error) {
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

func (a *AuthMTPostgreSQL) writeBans(bans map[string]string) error {
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
