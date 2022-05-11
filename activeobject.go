package proxy

import "github.com/anon55555/mt"

func (sc *ServerConn) swapAOID(ao *mt.AOID) {
	if sc.client() != nil {
		if *ao == sc.client().playerCAO {
			*ao = sc.client().currentCAO
		} else if *ao == sc.client().currentCAO {
			*ao = sc.client().playerCAO
		}
	}
}

func (sc *ServerConn) handleAOMsg(aoMsg mt.AOMsg) {
	switch msg := aoMsg.(type) {
	case *mt.AOCmdAttach:
		sc.swapAOID(&msg.Attach.ParentID)
	case *mt.AOCmdProps:
		for j := range msg.Props.Textures {
			prependTexture(sc.mediaPool, &msg.Props.Textures[j])
		}
		prepend(sc.mediaPool, &msg.Props.Mesh)
		prepend(sc.mediaPool, &msg.Props.Itemstring)
		prependTexture(sc.mediaPool, &msg.Props.DmgTextureMod)
	case *mt.AOCmdSpawnInfant:
		sc.swapAOID(&msg.ID)
	case *mt.AOCmdTextureMod:
		prependTexture(sc.mediaPool, &msg.Mod)
	}
}
