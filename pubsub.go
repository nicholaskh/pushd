package main

import (
	"fmt"
	cmap "github.com/nicholaskh/golib/concurrent/map"
	log "github.com/nicholaskh/log4go"
)

var (
	pubsubChannels *PubsubChannels = NewPubsubChannels()
)

type PubsubChannels struct {
	cmap.ConcurrentMap
}

func NewPubsubChannels() (this *PubsubChannels) {
	this = new(PubsubChannels)
	this.ConcurrentMap = cmap.New()
	return
}

func (this *PubsubChannels) Get(channel string) (clients map[*Client]int, exists bool) {
	clientsInterface, exists := pubsubChannels.ConcurrentMap.Get(channel)
	clients, _ = clientsInterface.(map[*Client]int)
	return
}

// TODO subscribe count of channel
func subscribe(cli *Client, channel string) string {
	_, exists := cli.channels[channel]
	if exists {
		return fmt.Sprintf("%s %s", OUTPUT_ALREADY_SUBSCRIBED, channel)
	} else {
		cli.channels[channel] = 1
		clients, exists := pubsubChannels.Get(channel)
		if exists {
			clients[cli] = 1
		} else {
			clients = map[*Client]int{cli: 1}
		}
		pubsubChannels.Set(channel, clients)
		return fmt.Sprintf("%s %s", OUTPUT_SUBSCRIBED, channel)
	}
}

func unsubscribe(cli *Client, channel string) string {
	_, exists := cli.channels[channel]
	if exists {
		delete(cli.channels, channel)
		clients, exists := pubsubChannels.Get(channel)
		if exists {
			delete(clients, cli)
		}
		clients, exists = pubsubChannels.Get(channel)
		if len(clients) == 0 {
			pubsubChannels.Remove(channel)
		}
		return fmt.Sprintf("%s %s", OUTPUT_UNSUBSCRIBED, channel)
	} else {
		return fmt.Sprintf("%s %s", OUTPUT_NOT_SUBSCRIBED, channel)
	}
}

func unsubscribeAllChannels(cli *Client) {
	for channel, _ := range cli.channels {
		clients, _ := pubsubChannels.Get(channel)
		delete(clients, cli)
		if len(clients) == 0 {
			pubsubChannels.Remove(channel)
		}
	}
	cli.channels = nil
}

func publish(channel, msg string) string {
	clients, exists := pubsubChannels.Get(channel)
	if exists {
		msgByte := []byte(msg)
		log.Debug("channel %s subscribed by clients%s", channel, clients)
		for cli, _ := range clients {
			cli.msgQueue <- msgByte
		}
	}
	return OUTPUT_PUBLISHED
}
