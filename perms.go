package proxy

// Perms returns the permissions of the ClientConn.
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
func (cc *ClientConn) HasPerms(want ...string) bool {
	has := map[string]struct{}{
		"": struct{}{},
	}

	for _, perm := range cc.Perms() {
		has[perm] = struct{}{}
	}

	for _, perm := range want {
		if _, ok := has[perm]; !ok {
			return false
		}
	}

	return true
}
