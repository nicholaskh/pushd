package client

import (
	"net"
	"time"

	"github.com/nicholaskh/golib/server"
	log "github.com/nicholaskh/log4go"
)

type PushdClient struct {
	serverAddr string
	net.Conn
	proto       server.Protocol
	connTimeout time.Duration
	readTimeout time.Duration
}

func NewPushdClient(serverAddr string, proto server.Protocol, connTimeout, readTimeout time.Duration) *PushdClient {
	this := new(PushdClient)
	this.serverAddr = serverAddr
	this.proto = proto
	this.connTimeout = connTimeout
	this.readTimeout = readTimeout

	return this
}

func (this *PushdClient) Connect() error {
	var err error
	this.Conn, err = net.Dial("tcp", this.serverAddr)
	if err != nil {
		log.Error("Connect server[%s] error: %s", this.serverAddr, err.Error())
		this.Conn = nil
		return err
	}
	this.proto.SetConn(this.Conn)
	return nil
}

func (this *PushdClient) Write(msg []byte) (int, error) {
	data := this.proto.Marshal(msg)
	return this.Conn.Write(data)
}

func (this *PushdClient) Read() ([]byte, error) {
	return this.proto.Read()
}

func (this *PushdClient) IsConnected() bool {
	return this.Conn != nil
}

func (this *PushdClient) Close() {
	this.Conn.Close()
	this.Conn = nil
	this.proto.SetConn(nil)
}
