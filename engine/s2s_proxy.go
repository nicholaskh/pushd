package engine

import (
	"fmt"
	"net"
	"reflect"
	"strings"

	"github.com/nicholaskh/golib/cache"
	"github.com/nicholaskh/golib/set"
	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/config"
)

const (
	S2S_PORT = 2223

	RETRY_CNT = 2

	S2S_SUB_CMD   = "sub"
	S2S_PUB_CMD   = "pub"
	S2S_UNSUB_CMD = "unsub"
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
	//bind local addr since we will use this for peer addr on remote server
	laddr, err := net.ResolveTCPAddr("tcp", config.PushdConf.TcpListenAddr)
	if err != nil {
		panic(err)
	}
	laddr.Port = 0
	dialer := &net.Dialer{LocalAddr: laddr}

	this.Conn, err = dialer.Dial("tcp", this.addr)
	if err != nil {
		log.Warn("s2s connect to %s error: %s", this.addr, err.Error())
	}
	if this.Conn != nil {
		// just wait for s2s server close the connection
		go func() {
			for {
				input := make([]byte, 1460)
				_, err = this.Conn.Read(input)
				if err != nil {
					this.Conn.Close()
					return
				}
			}
		}()
	}
	return
}

func (this *Peer) writeMsg(msg string) {
	var err error

	if this.Conn != nil {
		_, err = this.Write([]byte(msg))
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

	ChannelPeers *cache.LruCache
	PubMsgChan   chan *PubTuple
	SubMsgChan   chan string
	UnsubMsgChan chan string

	Stats *ProxyStats
}

func NewS2sProxy() (this *S2sProxy) {
	this = new(S2sProxy)
	this.SubMsgChan = make(chan string, 100)
	this.UnsubMsgChan = make(chan string, 100)
	this.PubMsgChan = make(chan *PubTuple, 100)
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
	this.ChannelPeers = cache.NewLruCache(config.PushdConf.S2sChannelPeersMaxItems)

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
				go peer.writeMsg(fmt.Sprintf("%s %s %s %d\n", S2S_PUB_CMD, tuple.channel, tuple.msg, tuple.ts))
			}

		case channel := <-this.SubMsgChan:
			this.Stats.subCalls.Mark(1)
			this.Stats.outChannels.Mark(1)
			for _, peer := range this.peers {
				go peer.writeMsg(fmt.Sprintf("%s %s %s\n", S2S_SUB_CMD, channel))
			}

		case channel := <-this.UnsubMsgChan:
			this.Stats.unsubCalls.Mark(1)
			this.Stats.outChannels.Mark(-1)
			for _, peer := range this.peers {
				go peer.writeMsg(fmt.Sprintf("%s %s %s\n", S2S_UNSUB_CMD, channel))
			}
		}
	}
}

func (this *S2sProxy) GetPeersByChannel(channel string) (peers set.Set, exists bool) {
	peersInterface, exists := this.ChannelPeers.Get(channel)
	// TODO
	// !exists=>read from mongodb and write to cache
	// empty => delete from cache
	// else use it
	if peersInterface != nil {
		peers, _ = peersInterface.(set.Set)
	} else {
		peers = nil
	}
	return
}

func GetS2sAddr(servAddr string) (s2sAddr string) {
	parts := strings.Split(servAddr, ":")
	ip := parts[0]
	s2sPort := S2S_PORT
	s2sAddr = fmt.Sprintf("%s:%d", ip, s2sPort)
	return
}

type PubTuple struct {
	peers   set.Set
	msg     string
	channel string
	ts      int64
}

func NewPubTuple(peers set.Set, msg, channel string, ts int64) (this *PubTuple) {
	this = &PubTuple{peers: peers, msg: msg, channel: channel, ts: ts}
	return
}
