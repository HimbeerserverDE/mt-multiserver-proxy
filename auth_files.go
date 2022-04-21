package proxy

import (
	"net"
	"os"
	"time"
)

type authFiles struct{}

// Exists reports whether a user is registered.
func (a authFiles) Exists(name string) bool {
	os.Mkdir(Path("auth"), 0700)

	_, err := os.Stat(Path("auth/", name))
	return err == nil
}

// Passwd returns the SRP salt and verifier of a user or an error.
func (a authFiles) Passwd(name string) (salt, verifier []byte, err error) {
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
func (a authFiles) SetPasswd(name string, salt, verifier []byte) error {
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
func (a authFiles) LastSrv(name string) (string, error) {
	os.Mkdir(Path("auth"), 0700)
	os.Mkdir(Path("auth/", name), 0700)

	srv, err := os.ReadFile(Path("auth/", name, "/last_server"))
	return string(srv), err
}

// SetLastSrv sets the last server a user was on.
func (a authFiles) SetLastSrv(name, srv string) error {
	os.Mkdir(Path("auth"), 0700)
	os.Mkdir(Path("auth/", name), 0700)

	return os.WriteFile(Path("auth/", name, "/last_server"), []byte(srv), 0600)
}

// Timestamp returns the last time an authentication entry was accessed
// or an error.
func (a authFiles) Timestamp(name string) (time.Time, error) {
	os.Mkdir(Path("auth"), 0700)

	info, err := os.Stat(Path("auth/", name, "/timestamp"))
	if err != nil {
		return time.Time{}, err
	}

	return info.ModTime(), nil
}

// Import deletes all users and adds the passed users.
func (a authFiles) Import(in []user) {
	os.Mkdir(Path("auth"), 0700)

	for _, u := range in {
		a.SetPasswd(u.name, u.salt, u.verifier)
		os.Chtimes(Path("auth/", u.name, "/timestamp"), u.timestamp, u.timestamp)
	}
}

// Export returns data that can be processed by Import
// or an error.
func (a authFiles) Export() ([]user, error) {
	dir, err := os.ReadDir(Path("auth"))
	if err != nil {
		return nil, err
	}

	var out []user
	for _, f := range dir {
		u := user{name: f.Name()}

		u.timestamp, err = a.Timestamp(u.name)
		if err != nil {
			return nil, err
		}

		u.salt, u.verifier, err = a.Passwd(u.name)
		if err != nil {
			return nil, err
		}

		out = append(out, u)
	}

	return out, nil
}

// Ban adds a ban entry for a network address and an associated name.
func (a authFiles) Ban(addr, name string) error {
	os.Mkdir(Path("ban"), 0700)
	return os.WriteFile(Path("ban/", addr), []byte(name), 0600)
}

// Unban deletes a ban entry. It accepts both network addresses
// and player names.
func (a authFiles) Unban(id string) error {
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
func (a authFiles) Banned(addr *net.UDPAddr) bool {
	os.Mkdir(Path("ban"), 0700)

	_, err := os.Stat(Path("ban/", addr.IP.String()))
	if os.IsNotExist(err) {
		return false
	}

	return true
}

// ImportBans deletes all ban entries and adds the passed entries.
func (a authFiles) ImportBans(in []ban) {
	os.Mkdir(Path("ban"), 0700)

	for _, b := range in {
		a.Ban(b.addr, b.name)
	}
}

// ExportBans returns data that can be processed by ImportBans
// or an error,
func (a authFiles) ExportBans() ([]ban, error) {
	os.Mkdir(Path("ban"), 0700)

	dir, err := os.ReadDir(Path("ban"))
	if err != nil {
		return nil, err
	}

	var out []ban
	for _, f := range dir {
		b := ban{addr: f.Name()}

		name, err := os.ReadFile(Path("ban/", f.Name()))
		if err != nil {
			return nil, err
		}

		b.name = string(name)
		out = append(out, b)
	}

	return out, nil
}

func (a authFiles) updateTimestamp(name string) {
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
