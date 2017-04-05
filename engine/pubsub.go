package engine

import (
	"fmt"
	"time"

	"github.com/nicholaskh/golib/cache"
	cmap "github.com/nicholaskh/golib/concurrent/map"
	"github.com/nicholaskh/golib/set"
	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/config"
	"github.com/nicholaskh/pushd/engine/storage"
)

var (
	PubsubChannels *PubsubChans
)

type PubsubChans struct {
	*cache.LruCache
}

func NewPubsubChannels(maxChannelItems int) (this *PubsubChans) {
	this = new(PubsubChans)
	this.LruCache = cache.NewLruCache(maxChannelItems)
	return
}

func (this *PubsubChans) Get(channel string) (clients cmap.ConcurrentMap, exists bool) {
	clientsInterface, exists := PubsubChannels.LruCache.Get(channel)
	clients, _ = clientsInterface.(cmap.ConcurrentMap)
	return
}

func subscribe(cli *Client, channel string) string {
	log.Debug("%x", channel)
	_, exists := cli.Channels[channel]
	if exists {
		return fmt.Sprintf("%s %s", OUTPUT_ALREADY_SUBSCRIBED, channel)
	} else {
		cli.Channels[channel] = 1
		clients, exists := PubsubChannels.Get(channel)
		if exists {
			clients.Set(cli.RemoteAddr().String(), cli)
		} else {
			clients = cmap.New()
			clients.Set(cli.RemoteAddr().String(), cli)

			//s2s
			if config.PushdConf.IsDistMode() {
				Proxy.SubMsgChan <- channel
			}
		}
		PubsubChannels.Set(channel, clients)

		return fmt.Sprintf("%s %s", OUTPUT_SUBSCRIBED, channel)
	}

}

func unsubscribe(cli *Client, channel string) string {
	_, exists := cli.Channels[channel]
	if exists {
		delete(cli.Channels, channel)
		clients, exists := PubsubChannels.Get(channel)
		if exists {
			clients.Remove(cli.RemoteAddr().String())
		}
		clients, exists = PubsubChannels.Get(channel)

		if clients.Count() == 0 {
			PubsubChannels.Del(channel)

			//s2s
			if config.PushdConf.IsDistMode() {
				Proxy.UnsubMsgChan <- channel
			}
		}
		return fmt.Sprintf("%s %s", OUTPUT_UNSUBSCRIBED, channel)
	} else {
		return fmt.Sprintf("%s %s", OUTPUT_NOT_SUBSCRIBED, channel)
	}
}

func UnsubscribeAllChannels(cli *Client) {
	for channel, _ := range cli.Channels {
		clients, _ := PubsubChannels.Get(channel)
		clients.Remove(cli.RemoteAddr().String())
		if clients.Count() == 0 {
			PubsubChannels.Del(channel)
		}
	}
	cli.Channels = nil
}

func publish(channel, msg , uuid string, fromS2s bool) string {
	clients, exists := PubsubChannels.Get(channel)
	ts := time.Now().UnixNano()
	if exists {
		log.Debug("channel %s subscribed by %d clients", channel, clients.Count())
		for ele := range clients.Iter() {
			cli := ele.Val.(*Client)
			cli.Mutex.Lock()
			if cli.IsConnected() {
				if !fromS2s {
					go cli.WriteMsg(fmt.Sprintf("%s %d", msg, ts))
				} else {
					go cli.WriteMsg(msg)
				}
			}
			cli.Mutex.Unlock()
		}
	}

	storage.MsgCache.Store(&storage.MsgTuple{Channel: channel, Msg: msg, Ts: ts, Uuid: uuid})
	if !fromS2s && config.PushdConf.EnableStorage() {
		storage.EnqueueMsg(channel, msg, uuid, ts)
	}

	if !fromS2s {
		//s2s
		if config.PushdConf.IsDistMode() {
			var peers set.Set
			peers, exists = Proxy.Router.LookupPeersByChannel(channel)
			log.Debug("now peers %s", peers)
			if exists {
				Proxy.PubMsgChan <- NewPubTuple(peers, msg, channel, uuid, ts)
			}
		}

		return OUTPUT_PUBLISHED
	} else {
		return ""
	}
}
