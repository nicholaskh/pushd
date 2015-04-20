package engine

import (
	"fmt"
	"reflect"

	"github.com/nicholaskh/golib/set"
	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/config"
)

const (
	RETRY_CNT = 2

	S2S_SUB_CMD   = "sub"
	S2S_PUB_CMD   = "pub"
	S2S_UNSUB_CMD = "unsub"
)

var (
	Proxy *S2sProxy
)

type S2sProxy struct {
	Router       *Router
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

	go watchPeers(this)

	this.Router = NewRouter(config.PushdConf.S2sChannelPeersMaxItems)

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
			for _, peer := range this.Router.Peers {
				go peer.writeMsg(fmt.Sprintf("%s %s\n", S2S_SUB_CMD, channel))
			}

		case channel := <-this.UnsubMsgChan:
			this.Stats.unsubCalls.Mark(1)
			this.Stats.outChannels.Mark(-1)
			for _, peer := range this.Router.Peers {
				go peer.writeMsg(fmt.Sprintf("%s %s\n", S2S_UNSUB_CMD, channel))
			}
		}
	}
}

func (this *S2sProxy) ForwardMsg(msg, channel string, peers set.Set) {

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
