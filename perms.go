package proxy

import "strings"

// Perms returns the raw permissions of the ClientConn.
func (cc *ClientConn) Perms() []string {
	if cc.Name() == "" {
		return []string{}
	}

	grp, ok := Conf().UserGroups[cc.Name()]
	if !ok {
		grp = "default"
	}

	if perms, ok := Conf().Groups[grp]; ok {
		return perms
	}

	return []string{}
}

// HasPerms returns true if the ClientConn has all
// of the specified permissions. Otherwise it returns false.
// This method matches wildcards, but they may only be used
// at the end of a raw permission. Asterisks in other places
// will be treated as regular characters.
func (cc *ClientConn) HasPerms(want ...string) bool {
	has := map[string]struct{}{
		"": struct{}{},
	}

	for _, perm := range cc.Perms() {
		if strings.HasSuffix(perm, "*") {
			perm = perm[:len(perm)-1]

			for _, wperm := range want {
				if strings.HasPrefix(wperm, perm) {
					has[wperm] = struct{}{}
				}
			}
		} else {
			has[perm] = struct{}{}
		}
	}

	for _, perm := range want {
		if _, ok := has[perm]; !ok {
			return false
		}
	}

	return true
}
