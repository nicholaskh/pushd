package client

import (
	"github.com/nicholaskh/golib/server"
	"github.com/nicholaskh/golib/sync2"
)

const (
	TYPE_CLIENT = 1
	TYPE_SERVER = 2
)

type Client struct {
	Channels map[string]int
	MsgQueue chan string
	Output   chan string
	Uname    string
	Type     uint8
	Authed   bool
	Closed   bool
	Mutex    *sync2.Semaphore
	*server.Client
}

//TODO config of backlog of msgQueue
func NewClient() (this *Client) {
	this = new(Client)
	this.Channels = make(map[string]int)
	this.MsgQueue = make(chan string, 20)
	this.Output = make(chan string)
	this.Authed = false
	this.Closed = false
	this.Mutex = sync2.NewSemaphore(1, 0)
	return
}

func (this *Client) WaitMsg() {
	for {
		select {
		case msg, ok := <-this.MsgQueue:
			if !ok {
				return
			}
			this.Conn.Write([]byte(msg))

		case msg, ok := <-this.Output:
			if !ok {
				return
			}
			this.Conn.Write([]byte(msg))
		}
	}
}

func (this *Client) Close() {
	this.Mutex.Acquire()
	this.Closed = true
	this.Mutex.Release()
	close(this.Output)
	close(this.MsgQueue)
	this.Conn.Close()
}
