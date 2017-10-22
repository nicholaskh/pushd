package engine

import (
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
	S2S_PUSH_BINARY_MESSAGE = "push_bin_msg"			// 广播内容为二进制的消息
	S2S_PUSH_STRING_MESSAGE = "push_str_msg"			// 广播内容为字符型的消息

)

var (
	Proxy *S2sProxy
)

type S2sProxy struct {
	Router       *Router
	ForwardTuple chan *ForwardTuple
	Stats *ProxyStats
}

func NewS2sProxy() (this *S2sProxy) {
	this = new(S2sProxy)
	this.ForwardTuple = make(chan *ForwardTuple, 100)

	go watchPeers(this)

	this.Router = NewRouter(config.PushdConf.S2sChannelPeersMaxItems)

	this.Stats = newProxyStats()
	this.Stats.registerMetrics()

	return
}

func (this *S2sProxy) WaitMsg() {
	for {
		select {
		case tuple := <-this.ForwardTuple:
		// this.Stats.pubCalls.Mark(1)
			for peerInterface := range tuple.peers.Iter() {
				log.Debug("peer was %s %s", peerInterface, reflect.TypeOf(peerInterface))
				peer, _ := peerInterface.(*Peer)
				log.Debug("peer is %s %s", peer, reflect.TypeOf(peer))

				go peer.writeFormatMsg(tuple.cmd, tuple.msg)
			}
		}

	}
}


type ForwardTuple struct {
	peers set.Set
	msg []byte
	cmd string
}

func NewForwardTuple(peers set.Set, msg []byte, cmd string) (this *ForwardTuple) {
	this = &ForwardTuple{peers: peers, msg: msg, cmd: cmd}
	return
}
