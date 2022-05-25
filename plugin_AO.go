package proxy

import (
	"bufio"
	"fmt"
	"github.com/anon55555/mt"
	"os"
	"strings"
	"sync"
	"unicode/utf8"
)

type AOHandler struct {
	OnAOMsg       func(*ClientConn, *mt.ToSrvInteract) bool
	OnStopDigging func(*ClientConn, *mt.ToSrvInteract) bool
	OnDug         func(*ClientConn, *mt.ToSrvInteract) bool
	OnPlace       func(*ClientConn, *mt.ToSrvInteract) bool
	OnUse         func(*ClientConn, *mt.ToSrvInteract) bool
	OnActivate    func(*ClientConn, *mt.ToSrvInteract) bool
}

var AOHandlers []*NodeHandler
var AOHandlersMu sync.RWMutex

var AOCache map[string]*[]mt.AOInitData
var AOCacheMu sync.RWMutex

func handleAOMsg(sc *ServerConn, id mt.AOID, mg mt.AOMsg) bool {
	switch msg := mg.(type) {
	//case *mt.AOCmdPos:
	case *mt.AOCmdProps:
		p := &msg.Props
		if strings.Contains(p.Infotext, "mcl_signs") {
			p.Textures[0] = mt.Texture(generateTexture("PenisMomentPenisMomentPenisMomentPenisMomentPenisMomentPenisMomentPenisMomentPenisMomentPenisMomentPenisMomentPenisMoment\nsex", strings.Contains(p.Infotext, "mcl_signs:wall_sign")))
		}
	}

	return false
}

var charMap map[rune]string
var charMapMu sync.Once

const (
	SIGN_WIDTH         = 115
	LINE_LENGTH        = 15
	NUMBER_OF_LINES    = 4
	LINE_HEIGHT        = 14
	CHAR_WIDTH         = 5
	PRINTED_CHAR_WIDTH = CHAR_WIDTH + 1
)

func loadCharMap() {
	charMapMu.Do(func() {
		charMap = make(map[rune]string)
		f, err := os.Open("characters.txt")
		if err != nil {
			fmt.Println("[MT_SIGNS] Cant read characters.txt file!")
			os.Exit(-1)
		}

		s := bufio.NewScanner(f)
		var state uint8 = 1
		var char string
		var eChar string

		for s.Scan() {
			switch state {
			case 1:
				char = s.Text()
			case 2:
				eChar = s.Text()
			case 3:
				fmt.Println(char, eChar)
				ru, _ := utf8.DecodeRuneInString(char)
				charMap[ru] = "micl2_" + eChar
				state = 0
			}

			state++
		}
	})
}

func generateTexture(text string, wall bool) string {
	loadCharMap()

	texture := fmt.Sprintf("[combine:%dx%d", SIGN_WIDTH, SIGN_WIDTH)
	ypos := 0
	if wall {
		ypos = 30
	}

	for _, line := range strings.Split(text, "\n") {
		xpos := 0
		for _, letter := range line {
			if charMap[letter] != "" {
				texture += fmt.Sprintf(":%d,%d=%s.png", xpos, ypos, charMap[letter])

				xpos += PRINTED_CHAR_WIDTH
			}
		}

		ypos += LINE_HEIGHT
	}

	return texture
}
