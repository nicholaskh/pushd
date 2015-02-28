package main

import (
	"fmt"
	"github.com/nicholaskh/golib/server"
	"io"
	"net"
)

const (
	SERV_ADDR = ":2222"
)

type Processor struct {
}

func NewProcessor() (this *Processor) {
	this = new(Processor)
	return
}

func (this *Processor) run(conn net.Conn) {
	cli := NewClient(conn)
	go func(cli) {
		for {
			input := make([]byte, 1024)
			_, err := cli.conn.Read(input)

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

			//fmt.Println(input)
			cl := NewCmdline(input, cli)
			ret, err := cl.processCmd()
			// TODO log
			if err != nil {
				cli.conn.Write([]byte(err.Error()))
				continue
			}

			if ret == "" {
				return
			} else {
				cli.conn.Write([]byte("Received: " + ret + "\n"))
			}
		}
	}(cli)

	go cli.waitMsg()
}

type Client struct {
	conn     net.Conn
	cl       *Cmdline
	channels map[string]int
	msgQueue chan []byte
	output   chan []byte
}

func NewClient(conn net.Conn) (this *Client) {
	this = new(Client)
	this.conn = conn
	this.channels = make(map[string]int)
	this.msgQueue = make(chan []byte, 20)
	this.output = make(chan []byte)
	return
}

func (this *Client) waitMsg() {
	for {
		select {
		case msg := <-this.msgQueue:
			this.conn.Write(msg)

		case msg := <-this.output:
			this.conn.Write([]byte(fmt.Sprintf("Received: %s\n", msg)))
		}
	}
}

func (this *Client) Close() {
	unsubscribeAllChannels(this)
	close(this.msgQueue)
	this.conn.Close()
}
