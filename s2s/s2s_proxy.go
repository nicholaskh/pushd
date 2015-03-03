package s2s

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	cmap "github.com/nicholaskh/golib/concurrent/map"
	"github.com/nicholaskh/golib/ip"
	"github.com/nicholaskh/golib/set"
	"github.com/nicholaskh/pushd/config"
)

const (
	S2S_SUB_CMD = "subscribe %s"
	S2S_PUB_CMD = "publish %s %s"
)

var (
	Proxy *S2sProxy
)

type Peer struct {
	addr string
	net.Conn
}

func NewPeer(addr string) (this *Peer) {
	this = new(Peer)
	this.addr = addr
	return
}

func (this *Peer) sendSubscribe(channel string) {
	this.Write([]byte(fmt.Sprintf(S2S_SUB_CMD, channel)))
}

func (this *Peer) sendPublish(channel, msg string) {
	this.Write([]byte(fmt.Sprintf(S2S_SUB_CMD, channel, msg)))
}

type S2sProxy struct {
	servers      []string
	peers        []*Peer
	channelPeers cmap.ConcurrentMap
}

func NewS2sProxy() (this *S2sProxy) {
	this = new(S2sProxy)
	selfIps := ip.LocalIpv4Addrs()
	selfPort := strings.Split(config.PushdConf.TcpListenAddr, ":")[1]
	selfAddr := make(map[string]int)
	for _, selfIp := range selfIps {
		selfAddr[fmt.Sprintf("%s:%s", selfIp, selfPort)] = 1
	}
	var peer *Peer
	for _, server := range config.PushdConf.Servers {
		_, exists := selfAddr[server]
		if !exists {
			peer = NewPeer(this.getS2sAddr(server))
			this.peers = append(this.peers, peer)
		}
	}
	this.channelPeers = cmap.New()
	return
}

func (this *S2sProxy) RegisterChannel(channel string) {
	for _, peer := range this.peers {
		peer.sendSubscribe(channel)
	}
}

func (this *S2sProxy) UnregisterChannel(channel string) {

}

func (this *S2sProxy) Publish(peers set.Set, channel, msg string) {
	for peerInterface := range peers {
		peer, _ := peerInterface.(Peer)
		peer.sendPublish(channel, msg)
	}
}

func (this *S2sProxy) startProxyServ() {
	s := NewS2sServ()
	//TODO
	//s.LaunchTcpServ(confPushd.s2sAddr, s, confPushd.s2sPingInterval)
	s.LaunchTcpServ(this.getS2sAddr(config.PushdConf.TcpListenAddr), s, config.PushdConf.S2sPingInterval)
}

func (this *S2sProxy) connect(servers string) {

}

func (this *S2sProxy) getPeersByChannel(channel string) (peers set.Set, exists bool) {
	var peersInterface interface{}
	peersInterface, exists = this.channelPeers.Get(channel)
	peers, _ = peersInterface.(set.Set)
	return
}

// TODO port should fixed
func (this *S2sProxy) getS2sAddr(servAddr string) (s2sAddr string) {
	parts := strings.Split(servAddr, ":")
	ip := parts[0]
	port := parts[1]
	intPort, _ := strconv.Atoi(port)
	s2sPort := strconv.Itoa((intPort + 1))
	s2sAddr = fmt.Sprintf("%s:%s", ip, s2sPort)
	return
}
