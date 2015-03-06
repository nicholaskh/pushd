package engine

import (
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"

	"github.com/nicholaskh/golib/cache"
	"github.com/nicholaskh/golib/set"
	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/config"
)

const (
	RETRY_CNT = 2

	S2S_SUB_CMD = "sub"
	S2S_PUB_CMD = "pub"
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
	this.connect()
	return
}

func (this *Peer) connect() (err error) {
	this.Conn, err = net.Dial("tcp", this.addr)
	if err != nil {
		log.Warn("s2s connect to %s error: %s", this.addr, err.Error())
	}

	// just wait for s2s server close the connection
	if this.Conn != nil {
		go func() {
			for {
				input := make([]byte, 1460)
				_, err = this.Conn.Read(input)
				if err != nil {
					this.Conn.Close()
				}
			}
		}()
	}
	return
}

func (this *Peer) writeMsg(msg string) {
	var err error

	if this.Conn != nil {
		var num int
		num, err = this.Write([]byte(msg))
	}
	if err != nil || this.Conn == nil {
		// retry
		for i := 0; i < RETRY_CNT; i++ {
			err = this.connect()
			if err != nil {
				log.Warn("write to peer %s error: %s", this.addr, err.Error())
			} else {
				if this.Conn != nil {
					_, err = this.Write([]byte(msg))
					if err == nil {
						break
					}
				}
			}
		}
	}
}

type S2sProxy struct {
	peers map[string]*Peer

	// TODO lru cache
	channelPeers *cache.LruCache
	PubMsgChan   chan *PubTuple
	SubMsgChan   chan string
	UnsubMsgChan chan string

	Stats *ProxyStats
}

func NewS2sProxy() (this *S2sProxy) {
	this = new(S2sProxy)
	// TODO s2s channel backlog
	this.SubMsgChan = make(chan string, 10)
	this.PubMsgChan = make(chan *PubTuple, 10)
	this.peers = make(map[string]*Peer)

	var peer *Peer
	for _, server := range config.PushdConf.Servers {
		if server != config.PushdConf.TcpListenAddr {
			s2sServer := GetS2sAddr(server)
			peer = NewPeer(s2sServer)
			this.peers[s2sServer] = peer
		}
	}
	log.Debug("%s", this.peers)
	// TODO
	this.channelPeers = cache.NewLruCache(200000)

	this.Stats = newProxyStats()
	this.Stats.registerMetrics()

	return
}

func (this *S2sProxy) WaitMsg() {
	for {
		select {
		case tuple := <-this.PubMsgChan:
			this.Stats.pubCalls.Mark(1)
			for peerInterface := range tuple.peers.Iter() {
				log.Debug("peer was %s %s", peerInterface, reflect.TypeOf(peerInterface))
				peer, _ := peerInterface.(*Peer)
				log.Debug("peer is %s %s", peer, reflect.TypeOf(peer))
				go peer.writeMsg(fmt.Sprintf("%s %s %s", S2S_PUB_CMD, tuple.channel, tuple.msg))
			}

		case channel := <-this.SubMsgChan:
			this.Stats.subCalls.Mark(1)
			this.Stats.outChannels.Mark(1)
			for _, peer := range this.peers {
				go peer.writeMsg(fmt.Sprintf("%s %s %s", S2S_SUB_CMD, channel, config.PushdConf.TcpListenAddr))
			}
			// TODO save into redis

			//case channel := <-this.UnsubMsgChan:
			//TODO
		}
	}
}

func (this *S2sProxy) GetPeersByChannel(channel string) (peers set.Set, exists bool) {
	peersInterface, exists := this.channelPeers.Get(channel)
	// TODO
	// !exists=>read from redis and write to cache
	// empty => delete from cache
	// else use it
	if peersInterface != nil {
		peers, _ = peersInterface.(set.Set)
	} else {
		peers = nil
	}
	return
}

// TODO port should fixed
func GetS2sAddr(servAddr string) (s2sAddr string) {
	parts := strings.Split(servAddr, ":")
	ip := parts[0]
	port := parts[1]
	intPort, _ := strconv.Atoi(port)
	s2sPort := strconv.Itoa((intPort + 1))
	s2sAddr = fmt.Sprintf("%s:%s", ip, s2sPort)
	return
}

type PubTuple struct {
	peers   set.Set
	msg     string
	channel string
}

func NewPubTuple(peers set.Set, msg, channel string) (this *PubTuple) {
	this = &PubTuple{peers: peers, msg: msg, channel: channel}
	return
}
