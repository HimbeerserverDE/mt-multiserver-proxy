package proxy

import ()

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

func (cc *ClientConn) HasPerms(want ...string) bool {
	has := make(map[string]struct{})
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
