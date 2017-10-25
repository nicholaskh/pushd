package engine

import (
	"errors"
	"net"

	"github.com/nicholaskh/golib/cache"
	"github.com/nicholaskh/golib/set"
	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/config"
	"github.com/nicholaskh/golib/server"
	"fmt"
)

type Router struct {
	Peers map[string]*Peer
	cache *cache.LruCache
}

func NewRouter(maxCacheItems int) *Router {
	this := new(Router)
	this.cache = cache.NewLruCache(maxCacheItems)
	this.Peers = make(map[string]*Peer)

	return this
}

func (this *Router) LookupPeersByChannel(channel string) (peers set.Set, exists bool) {
	peersI, exists := this.cache.Get(channel)

	if exists {
		peers = peersI.(set.Set)
	} else {
		peers = nil
	}

	return
}

func (this *Router) AddPeerToChannel(peerAddr, channel string) error {
	peers, exists := this.LookupPeersByChannel(channel)
	if !exists {
		peers = set.NewSet()
	}
	succ := peers.Add(this.Peers[peerAddr])
	if succ {
		this.cache.Set(channel, peers)
	}

	return nil
}

func (this *Router) RemovePeerFromChannel(peerAddr, channel string) error {
	peers, exists := this.LookupPeersByChannel(channel)
	if !exists {
		return errors.New("specified channel not exists")
	}
	peers.Remove(this.Peers[peerAddr])

	return nil
}

func (this *Router) connectPeer(server string) {
	if server == config.PushdConf.S2sListenAddr {
		return
	}
	peer := NewPeer(server)
	this.Peers[server] = peer
}

type Peer struct {
	addr string
	client *server.Client
}

func NewPeer(addr string) (this *Peer) {
	this = new(Peer)
	this.addr = addr

	this.initClientConn()
	return
}

func (this *Peer) initClientConn() (err error) {
	//bind local addr since we will use this for peer addr on remote server
	laddr, err := net.ResolveTCPAddr("tcp", config.PushdConf.TcpListenAddr)
	if err != nil {
		panic(err)
	}
	laddr.Port = 0
	dialer := &net.Dialer{LocalAddr: laddr}

	conn, err := dialer.Dial("tcp", this.addr)
	if err != nil {
		log.Warn("s2s connect to %s error: %s", this.addr, err.Error())
		return
	}

	this.client = server.NewClient(conn, server.CONN_TYPE_TCP, server.NewFixedLengthProtocol())

	// just wait for s2s server close the connection
	go func() {
		for {
			input := make([]byte, 1460)
			_, err = this.client.Read(input)
			if err != nil {
				this.client.Close()
				return
			}
		}
	}()

	return
}

func (this *Peer) WriteFormatMsg(op string, msg []byte) {
       if this.client.IsConnected() {
		err := this.client.WriteFormatBinMsg(op, msg)
	       if err == nil {
		       return
	       }
       }
       for i := 0; i < RETRY_CNT; i++ {
	       err := this.initClientConn()
	       if err == nil {
		       err = this.client.WriteFormatBinMsg(op, msg)
		       if err == nil {
			       log.Info("do reconnect peer(%s) success", this.addr)
			       return
		       }
	       }
       }
       log.Info(fmt.Sprintf("do reconnect peer(%s) failed, times retry is: %d %s", this.addr, RETRY_CNT))
       return
}
