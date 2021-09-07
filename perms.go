package proxy

import ()

func (cc *ClientConn) Perms() []string {
	if cc.name == "" {
		return []string{}
	}

	grp, ok := Conf().UserGroups[cc.name]
	if !ok {
		grp = "default"
	}

	if perms, ok := Conf().Groups[grp]; ok {
		return perms
	}

	return []string{}
}
