package engine

import (
	"errors"
	"net"

	"github.com/nicholaskh/golib/cache"
	"github.com/nicholaskh/golib/set"
	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/config"
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

func (this *Peer) writeFormatMsg(op string, msg []byte) {
	var err error

	if this.Conn != nil {

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
