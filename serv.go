package main

import (
	"fmt"
	"io"
	"net"
)

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

		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	for {
		buf := make([]byte, 1024)
		_, err := conn.Read(buf)

		if err != nil {
			if err == io.EOF {
				// TODO log debug
				conn.Close()
				return
			} else {
				// TODO log
			}
		}

		fmt.Println(buf)
		input := string(buf)

		ret, err := processReq(input)
		// TODO log
		if err != nil {

		}

		if input[0:4] == "quit" {
			// TODO log
			conn.Close()
			break
		}

		conn.Write([]byte("Received: " + ret + "\n"))
	}
}
