package proxy

import ()

func (cc *ClientConn) Perms() []string {
	if cc.name == "" {
		return []string{}
	}

	grp := Conf().UserGroups[cc.name]
	if perms, ok := Conf().Groups[grp]; ok {
		return perms
	}

	return []string{}
}
