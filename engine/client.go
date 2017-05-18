package engine

import (
	"github.com/nicholaskh/golib/server"
	log "github.com/nicholaskh/log4go"
)

const (
	TYPE_CLIENT = 1
	TYPE_SERVER = 2
)

type Client struct {
	Channels map[string]int
	Type     uint8
	*server.Client
	uuid string
}

func NewClient() (this *Client) {
	this = new(Client)
	this.Channels = make(map[string]int)
	return
}

func (this *Client) SetClient() {
	this.Type |= TYPE_CLIENT
}

func (this *Client) SetServer() {
	this.Type |= TYPE_SERVER
}

func (this *Client) IsClient() bool {
	return (this.Type & TYPE_CLIENT) != 0
}

func (this *Client) IsServer() bool {
	return (this.Type & TYPE_SERVER) != 0
}

func (this *Client) ClearIdentity() {
	this.Type &= 0
}

func (this *Client) Close() {
	log.Debug("client channels: %s", this.Channels)
	log.Debug("pubsub channels: %s", PubsubChannels)

	UnsubscribeAllChannels(this)
	if this.uuid != "" {
		UuidToClient.Remove(this.uuid)
	}
	uuidTokenMap.rmTokenInfo(this.uuid)
	this.Client.Close()
}
