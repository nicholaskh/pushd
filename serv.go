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

		go handleRequest(&client{conn: conn, channels: make(map[string]int)})
	}
}

func handleRequest(cli *client) {
	for {
		buf := make([]byte, 1024)
		_, err := cli.conn.Read(buf)

		if err != nil {
			if err == io.EOF {
				// TODO log debug
				cli.Close()
				fmt.Println(cli.channels)
				fmt.Println(pubsubChannels)
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
			cli.conn.Write([]byte(err.Error()))
			continue
		}

		if cli.input[0:4] == "quit" {
			// TODO log
			cli.Close()
			break
		}

		cli.conn.Write([]byte("Received: " + ret + "\n"))
	}
}

type client struct {
	conn     net.Conn
	input    string
	cl       *cmdline
	channels map[string]int
}

func (this *client) Close() {
	unsubscribeAllChannels(this)
	this.conn.Close()
}
