package main

import (
	"fmt"
	"io"
	"net"

	log "github.com/nicholaskh/log4go"
)

type Processor struct {
}

func NewProcessor() (this *Processor) {
	this = new(Processor)
	return
}

func (this *Processor) Run(conn net.Conn) {
	cli := NewClient(conn)
	go func(cli *Client) {
		for {
			input := make([]byte, 1460)
			_, err := cli.conn.Read(input)

			if err != nil {
				if err == io.EOF {
					log.Info("Client shutdown: %s", cli.conn.RemoteAddr())
					cli.Close()
					log.Debug("client channels: %s", cli.channels)
					jsonChannels, err := pubsubChannels.MarshalJSON()
					if err != nil {
						log.Error("Json marshal error: %s", err.Error())
					}

					log.Debug("pubsub channels: %s", jsonChannels)
					return
				} else if nerr, ok := err.(net.Error); !ok || !nerr.Temporary() {
					log.Error("Read from client[%s] error: %s", cli.conn.RemoteAddr(), err.Error())
				}
			}

			log.Debug("input: %x", input)
			cl := NewCmdline(input, cli)
			ret, err := cl.processCmd()
			if err != nil {
				log.Error("Process cmd[%s %s] error: %s", cl.cmd, cl.params, err.Error())
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
