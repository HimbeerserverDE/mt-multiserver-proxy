package proxy

import (
	"bufio"
	"errors"
	"io"
	"log"
	"net"
)

var telnetCh = make(chan struct{})

func telnetServer() error {
	ln, err := net.Listen("tcp", Conf().TelnetAddr)
	if err != nil {
		return err
	}
	defer ln.Close()

	for {
		select {
		case <-telnetCh:
			return nil
		default:
			conn, err := ln.Accept()
			if err != nil {
				log.Print("{←|⇶} ", err)
				continue
			}

			go handleTelnet(conn)
		}
	}
}

func handleTelnet(conn net.Conn) {
	defer conn.Close()

	r, w := bufio.NewReader(conn), bufio.NewWriter(conn)
	b := bufio.NewReadWriter(r, w)

	b.WriteString("mt-multiserver-proxy console\n")
	for {
		b.WriteString(">")

		_, err := b.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}

			log.Print("{←|⇶} ", err)
			continue
		}
	}
}
