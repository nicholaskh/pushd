package engine

import (
	"fmt"

	cmap "github.com/nicholaskh/golib/concurrent/map"
	"github.com/nicholaskh/golib/set"
	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/client"
)

var (
	PubsubChannels *PubsubChans = NewPubsubChannels()
)

type PubsubChans struct {
	cmap.ConcurrentMap
}

func NewPubsubChannels() (this *PubsubChans) {
	this = new(PubsubChans)
	this.ConcurrentMap = cmap.New()
	return
}

func (this *PubsubChans) Get(channel string) (clients map[*client.Client]int, exists bool) {
	clientsInterface, exists := PubsubChannels.ConcurrentMap.Get(channel)
	clients, _ = clientsInterface.(map[*client.Client]int)
	return
}

// TODO subscribe count of channel
func subscribe(cli *client.Client, channel string) string {
	_, exists := cli.Channels[channel]
	if exists {
		return fmt.Sprintf("%s %s", OUTPUT_ALREADY_SUBSCRIBED, channel)
	} else {
		cli.Channels[channel] = 1
		clients, exists := PubsubChannels.Get(channel)
		if exists {
			clients[cli] = 1
		} else {
			clients = map[*client.Client]int{cli: 1}

			//s2s
			Proxy.SubMsgChan <- channel
		}
		PubsubChannels.Set(channel, clients)

		return fmt.Sprintf("%s %s", OUTPUT_SUBSCRIBED, channel)
	}

}

func unsubscribe(cli *client.Client, channel string) string {
	_, exists := cli.Channels[channel]
	if exists {
		delete(cli.Channels, channel)
		clients, exists := PubsubChannels.Get(channel)
		if exists {
			delete(clients, cli)
		}
		clients, exists = PubsubChannels.Get(channel)
		if len(clients) == 0 {
			PubsubChannels.Remove(channel)
		}
		return fmt.Sprintf("%s %s", OUTPUT_UNSUBSCRIBED, channel)
	} else {
		return fmt.Sprintf("%s %s", OUTPUT_NOT_SUBSCRIBED, channel)
	}
}

func UnsubscribeAllChannels(cli *client.Client) {
	for channel, _ := range cli.Channels {
		clients, _ := PubsubChannels.Get(channel)
		delete(clients, cli)
		if len(clients) == 0 {
			PubsubChannels.Remove(channel)
		}
	}
	cli.Channels = nil
}

func publish(channel, msg string, fromS2s bool) string {
	clients, exists := PubsubChannels.Get(channel)
	if exists {
		log.Debug("channel %s subscribed by clients%s", channel, clients)
		for cli, _ := range clients {
			cli.MsgQueue <- msg
		}
	}

	if !fromS2s {
		//s2s
		var peers set.Set
		peers, exists = Proxy.GetPeersByChannel(channel)
		log.Debug("now peers %s", peers)
		if exists {
			Proxy.PubMsgChan <- NewPubTuple(peers, msg, channel)
		}

		return OUTPUT_PUBLISHED
	} else {
		return ""
	}
}
