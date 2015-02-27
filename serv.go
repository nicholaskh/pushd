package main

import (
	"fmt"
	"io"
	"net"
)

type client struct {
	conn  net.Conn
	input string
	cl    *cmdline
}

const (
	SERV_PORT = ":2222"
)

func startServ() {
	ln, err := net.Listen("tcp", SERV_PORT)

	// TODO log
	if err != nil {

	}

	defer ln.Close()

	// TODO log
	fmt.Println("Listening on " + ":" + SERV_PORT)

	for {
		conn, err := ln.Accept()

		// TODO log
		if err != nil {

		}

		go handleRequest(&client{conn: conn})
	}
}

func handleRequest(cli *client) {
	for {
		buf := make([]byte, 1024)
		_, err := cli.conn.Read(buf)

		if err != nil {
			if err == io.EOF {
				// TODO log debug
				cli.conn.Close()
				return
			} else {
				// TODO log
			}
		}

		fmt.Println(buf)
		cli.input = string(buf)

		ret, err := processReq(cli)
		// TODO log
		if err != nil {

		}

		if cli.input[0:4] == "quit" {
			// TODO log
			cli.conn.Close()
			break
		}

		cli.conn.Write([]byte("Received: " + ret + "\n"))
	}
}
