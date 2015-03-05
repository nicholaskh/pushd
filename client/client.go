package client

import (
	"github.com/nicholaskh/golib/server"
)

type Client struct {
	Channels map[string]int
	MsgQueue chan string
	Output   chan string
	Uname    string
	*server.Client
}

//TODO config of backlog of msgQueue
func NewClient() (this *Client) {
	this = new(Client)
	this.Channels = make(map[string]int)
	this.MsgQueue = make(chan string, 20)
	this.Output = make(chan string)
	return
}

func (this *Client) WaitMsg() {
	for {
		select {
		case msg := <-this.MsgQueue:
			this.Conn.Write([]byte(msg))

		case msg := <-this.Output:
			this.Conn.Write([]byte(msg))
		}
	}
}

func (this *Client) Close() {
	close(this.MsgQueue)
	close(this.Output)
	this.Conn.Close()
}
