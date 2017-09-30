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

	S2S_PUSH_CMD = "push"			// 广播用户发送的消息
	S2S_ADD_USER_INFO = "add_user_info" 	// 某用户上线，状态表中更新或增加用户信息
	S2S_DISABLE_NOTIFY = "disable_notify"   // 关闭某用户的离线消息推送
	S2S_ENABLE_NOTIFY = "enable_notify"	// 激活某用户的离线消息推送
	S2S_USER_OFFLINE = "user_offline"	// 某用户下线
)

var (
	Proxy *S2sProxy
)

type S2sProxy struct {
	Router       *Router
	PubMsgChan   chan *PubTuple
	PubMsgChan2   chan *PubTuple2
	SubMsgChan   chan string
	UnsubMsgChan chan string

	Stats *ProxyStats
}

func NewS2sProxy() (this *S2sProxy) {
	this = new(S2sProxy)
	this.SubMsgChan = make(chan string, 100)
	this.UnsubMsgChan = make(chan string, 100)
	this.PubMsgChan = make(chan *PubTuple, 100)
	this.PubMsgChan2 = make(chan *PubTuple2, 100)

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
				go peer.writeMsg(fmt.Sprintf("%s %s %s %d %d %s\n", S2S_PUB_CMD, tuple.channel, tuple.uuid, tuple.ts, tuple.msgId, tuple.msg))
			}

		case tuple2 := <-this.PubMsgChan2:
			this.Stats.pubCalls.Mark(1)
			for peerInterface := range tuple2.peers.Iter() {
				log.Debug("peer was %s %s", peerInterface, reflect.TypeOf(peerInterface))
				peer, _ := peerInterface.(*Peer)
				log.Debug("peer is %s %s", peer, reflect.TypeOf(peer))
				go peer.writeMsg(tuple2.msg)
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

type PubTuple struct {
	peers   set.Set
	msg     string
	channel string
	uuid	string
	ts      int64
	msgId   int64
}

func NewPubTuple(peers set.Set, msg, channel , uuid string, ts, msgId int64) (this *PubTuple) {
	this = &PubTuple{peers: peers, msg: msg, channel: channel, uuid: uuid, ts: ts, msgId:msgId}
	return
}

// It maybe replace of PubTuple in the future
type PubTuple2 struct {
	peers   set.Set
	msg     string
}

func NewPubTuple2(peers set.Set, msg string) (this *PubTuple2) {
	this = &PubTuple2{peers: peers, msg: msg}
	return
}