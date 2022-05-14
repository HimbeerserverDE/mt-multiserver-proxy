package proxy

import (
	"bytes"
	"encoding/json"
	"log"
	"math"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"sync"
	"time"
)

const (
	listAdd    = "start"
	listUpdate = "update"
	listRm     = "delete"
)

var announceMu sync.Mutex

func announce(action string) error {
	announceMu.Lock()
	defer announceMu.Unlock()

	addr, err := net.ResolveUDPAddr("udp", Conf().BindAddr)
	if err != nil {
		return err
	}

	a := map[string]interface{}{
		"action":  action,
		"address": addr.IP.String(),
		"port":    addr.Port,
	}

	if action != listRm {
		a["name"] = Conf().List.Name
		a["description"] = Conf().List.Desc
		a["version"] = versionString
		a["proto_min"] = protoVer
		a["proto_max"] = protoVer
		a["url"] = Conf().List.URL
		a["creative"] = Conf().List.Creative
		a["damage"] = Conf().List.Dmg
		a["password"] = Conf().RequirePasswd
		a["pvp"] = Conf().List.PvP
		a["uptime"] = math.Floor(Uptime().Seconds())
		a["game_time"] = 0

		playersMu.RLock()
		a["clients"] = len(players)
		clts := make([]string, 0, len(players))
		for player := range players {
			clts = append(clts, player)
		}
		playersMu.RUnlock()

		a["clients_max"] = Conf().UserLimit
		a["clients_list"] = clts
		a["gameid"] = Conf().List.Game
	}

	if action == listAdd {
		a["can_see_far_names"] = Conf().List.FarNames
		a["mods"] = Conf().List.Mods
	}

	s, err := json.Marshal(a)
	if err != nil {
		return err
	}

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)

	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", "form-data; name=\"json\"")

	part, _ := w.CreatePart(header)
	part.Write(s)
	w.Close()

	_, err = http.Post(Conf().List.Addr+"/announce", "multipart/form-data; boundary="+w.Boundary(), body)
	if err != nil {
		return err
	}

	log.Println("announce", action)
	return nil
}

func init() {
	if Conf().List.Enable {
		go func() {
			var added bool
			t := time.NewTicker(time.Duration(Conf().List.Interval) * time.Second)
			for {
				<-t.C
				if !added {
					if err := announce(listAdd); err != nil {
						log.Print(err)
					}

					added = true
					continue
				}

				if err := announce(listUpdate); err != nil {
					log.Print(err)
				}
			}
		}()
	}
}
