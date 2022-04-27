package proxy

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
)

// A TelnetWriter can be used to print something at the other end
// of a telnet connection. It implements the io.Writer interface.
type TelnetWriter struct {
	conn net.Conn
}

// Write writes its parameter to the telnet connection.
// A trailing newline is always appended.
// It returns the number of bytes written and an error.
func (tw *TelnetWriter) Write(p []byte) (n int, err error) {
	return tw.conn.Write(append(p, '\n'))
}

var telnetCh = make(chan struct{})

func telnetServer() error {
	ln, err := net.Listen("tcp", Conf().TelnetAddr)
	if err != nil {
		return err
	}
	defer ln.Close()

	log.Println("listen telnet", ln.Addr())

	for {
		select {
		case <-telnetCh:
			return nil
		default:
			conn, err := ln.Accept()
			if err != nil {
				log.Print(err)
				continue
			}

			go handleTelnet(conn)
		}
	}
}

func handleTelnet(conn net.Conn) {
	tlog := func(dir string, v ...interface{}) {
		prefix := fmt.Sprintf("[telnet %s] ", conn.RemoteAddr())
		l := log.New(logWriter, prefix, log.LstdFlags|log.Lmsgprefix)
		l.Println(append([]interface{}{dir}, v...)...)
	}

	tlog("<->", "connect")

	defer tlog("<->", "disconnect")
	defer conn.Close()

	readString := func(delim byte) (string, error) {
		s, err := bufio.NewReader(conn).ReadString(delim)
		if err != nil || len(s) == 0 {
			return s, err
		}

		i := int(math.Max(float64(len(s)-1), 1))
		s = s[:i]
		return s, nil
	}

	writeString := func(s string) (n int, err error) {
		return io.WriteString(conn, s)
	}

	writeString("mt-multiserver-proxy console\n")
	writeString("Type \\quit or \\q to disconnect.\n")

	for {
		writeString(Conf().CmdPrefix)

		s, err := readString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}

			log.Print(err)
			continue
		}

		tlog("->", "command", s)

		if s == "\\quit" || s == "\\q" {
			return
		}

		result := onTelnetMsg(tlog, &TelnetWriter{conn: conn}, s)
		if result != "\n" {
			writeString(result)
		}
	}
}
