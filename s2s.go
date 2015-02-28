package main

import (
	cmap "github.com/nicholaskh/golib/concurrent/map"
	"net"
)

type S2sProxy struct {
	channelConns cmap.ConcurrentMap
}

func (this *S2sProxy) Get(channel string) (conns []net.Conn, exists bool) {
	connsInterface, exists := this.channelConns.Get(channel)
	conns, _ = connsInterface.([]net.Conn)
	return
}

func (this *S2sProxy) RegisterChannel(channel string) {
	servers := []string{"127.0.0.1:2000"}
	this.connect(servers)
}

func (this *S2sProxy) UnregisterChannel(channel string) {
	conns, exists := this.Get(channel)
	if exists {
		this.disconnect(conns)
	}
}

func (this *S2sProxy) connect(servers []string) {

}

func (this *S2sProxy) disconnect(conns []net.Conn) {
	for _, conn := range conns {
		conn.Close()
	}
}
