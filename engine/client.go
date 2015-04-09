package engine

import (
	"fmt"

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
	Authed   bool
	*server.Client
}

func NewClient() (this *Client) {
	this = new(Client)
	this.Channels = make(map[string]int)
	this.Authed = false
	return
}

func (this *Client) Close() {
	log.Debug("client channels: %s", this.Channels)
	log.Debug("pubsub channels: %s", PubsubChannels)

	UnsubscribeAllChannels(this)
}

func (this *Client) FormatMessageOutput(msg string) string {
	return fmt.Sprintf("%c%s%c", OUTPUT_MESSAGE_PREFIX, msg, OUTPUT_DELIMITER)
}

func (this *Client) FormatCommandOutput(msg string) string {
	return fmt.Sprintf("%c%s%c", OUTPUT_COMMAND_PREFIX, msg, OUTPUT_DELIMITER)
}
