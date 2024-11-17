package proxy

import (
	"os"
	"time"
)

type AuthFiles struct{}

// Exists reports whether a user is registered.
// Error cases count as inexistent.
func (a AuthFiles) Exists(name string) bool {
	os.Mkdir(Path("auth"), 0700)

	_, err := os.Stat(Path("auth/", name))
	return err == nil
}

// Passwd returns the SRP salt and verifier of a user or an error.
func (a AuthFiles) Passwd(name string) (salt, verifier []byte, err error) {
	os.Mkdir(Path("auth"), 0700)

	salt, err = os.ReadFile(Path("auth/", name, "/salt"))
	if err != nil {
		return
	}

	verifier, err = os.ReadFile(Path("auth/", name, "/verifier"))
	if err != nil {
		return
	}

	a.updateTimestamp(name)
	return
}

// SetPasswd creates a password entry if necessary
// and sets the password of a user.
func (a AuthFiles) SetPasswd(name string, salt, verifier []byte) error {
	os.Mkdir(Path("auth"), 0700)
	os.Mkdir(Path("auth/", name), 0700)

	if err := os.WriteFile(Path("auth/", name, "/salt"), salt, 0600); err != nil {
		return err
	}

	if err := os.WriteFile(Path("auth/", name, "/verifier"), verifier, 0600); err != nil {
		return err
	}

	a.updateTimestamp(name)
	return nil
}

// LastSrv returns the last server a user was on.
func (a AuthFiles) LastSrv(name string) (string, error) {
	os.Mkdir(Path("auth"), 0700)
	os.Mkdir(Path("auth/", name), 0700)

	srv, err := os.ReadFile(Path("auth/", name, "/last_server"))
	return string(srv), err
}

// SetLastSrv sets the last server a user was on.
func (a AuthFiles) SetLastSrv(name, srv string) error {
	os.Mkdir(Path("auth"), 0700)
	os.Mkdir(Path("auth/", name), 0700)

	return os.WriteFile(Path("auth/", name, "/last_server"), []byte(srv), 0600)
}

// Timestamp returns the last time an authentication entry was accessed
// or an error.
func (a AuthFiles) Timestamp(name string) (time.Time, error) {
	os.Mkdir(Path("auth"), 0700)

	info, err := os.Stat(Path("auth/", name, "/timestamp"))
	if err != nil {
		return time.Time{}, err
	}

	return info.ModTime(), nil
}

// Import deletes all users and adds the passed users.
func (a AuthFiles) Import(in []User) error {
	os.Mkdir(Path("auth"), 0700)

	for _, u := range in {
		if err := a.SetPasswd(u.Name, u.Salt, u.Verifier); err != nil {
			return err
		}

		if err := os.Chtimes(Path("auth/", u.Name, "/timestamp"), u.Timestamp, u.Timestamp); err != nil {
			return err
		}
	}

	return nil
}

// Export returns data that can be processed by Import
// or an error.
func (a AuthFiles) Export() ([]User, error) {
	dir, err := os.ReadDir(Path("auth"))
	if err != nil {
		return nil, err
	}

	var out []User
	for _, f := range dir {
		u := User{Name: f.Name()}

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
func (a AuthFiles) Ban(addr, name string) error {
	os.Mkdir(Path("ban"), 0700)
	return os.WriteFile(Path("ban/", addr), []byte(name), 0600)
}

// Unban deletes a ban entry. It accepts both network addresses
// and player names.
func (a AuthFiles) Unban(id string) error {
	os.Mkdir(Path("ban"), 0700)

	if err := os.Remove(Path("ban/", id)); err != nil {
		if os.IsNotExist(err) {
			dir, err := os.ReadDir(Path("ban"))
			if err != nil {
				return err
			}

			for _, f := range dir {
				name, err := os.ReadFile(Path("ban/", f.Name()))
				if err != nil {
					return err
				}

				if string(name) == id {
					return os.Remove(Path("ban/", f.Name()))
				}
			}
		}
	}

	return nil
}

// Banned reports whether a network address is banned.
// Error cases count as banned.
func (a AuthFiles) Banned(addr, name string) bool {
	os.Mkdir(Path("ban"), 0700)

	_, err := os.Stat(Path("ban/", addr))
	if os.IsNotExist(err) {
		return false
	}

	return true
}

// RecordFail is a no-op.
func (a AuthFiles) RecordFail(addr, name string, sudo bool) error {
	return nil
}

// ImportBans deletes all ban entries and adds the passed entries.
func (a AuthFiles) ImportBans(in []Ban) error {
	os.Mkdir(Path("ban"), 0700)

	for _, b := range in {
		if err := a.Ban(b.Addr, b.Name); err != nil {
			return err
		}
	}

	return nil
}

// ExportBans returns data that can be processed by ImportBans
// or an error,
func (a AuthFiles) ExportBans() ([]Ban, error) {
	os.Mkdir(Path("ban"), 0700)

	dir, err := os.ReadDir(Path("ban"))
	if err != nil {
		return nil, err
	}

	var out []Ban
	for _, f := range dir {
		b := Ban{Addr: f.Name()}

		name, err := os.ReadFile(Path("ban/", f.Name()))
		if err != nil {
			return nil, err
		}

		b.Name = string(name)
		out = append(out, b)
	}

	return out, nil
}

func (a AuthFiles) updateTimestamp(name string) {
	os.Mkdir(Path("auth"), 0700)

	path := Path("auth/", name, "/timestamp")

	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0600)
			if err != nil {
				return
			}

			f.Close()
		}

		return
	}

	t := time.Now().Local()
	os.Chtimes(path, t, t)
}
