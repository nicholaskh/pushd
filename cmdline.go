package main

import (
	"errors"
	"strings"

	log "github.com/nicholaskh/log4go"
)

type Cmdline struct {
	cmd    string
	params []string
	*Client
}

const (
	CMD_SUBSCRIB    = "subscribe"
	CMD_PUBLISH     = "publish"
	CMD_UNSUBSCRIBE = "unsubscribe"

	OUTPUT_SUBSCRIBED         = "SUBSCRIBED"
	OUTPUT_ALREADY_SUBSCRIBED = "ALREADY SUBSCRIBED"
	OUTPUT_PUBLISHED          = "PUBLISHED"
	OUTPUT_NOT_SUBSCRIBED     = "NOT SUBSCRIBED"
	OUTPUT_UNSUBSCRIBED       = "UNSUBSCRIBED"
)

func NewCmdline(input []byte, cli *Client) (this *Cmdline) {
	this = new(Cmdline)
	parts := strings.Split(trim(string(input)), " ")
	this.cmd = parts[0]
	this.params = parts[1:]
	this.Client = cli
	return
}

func (this *Cmdline) processCmd() (string, error) {
	log.Debug("Input cmd: %s %s", this.cmd, this.params)
	var ret string
	switch this.cmd {
	case CMD_SUBSCRIB:
		ret = subscribe(this.Client, this.params[0])

	case CMD_PUBLISH:
		if len(this.params) < 2 {
			return "", errors.New("Publish without msg")
		} else {
			ret = publish(this.params[0], this.params[1])
		}

	case CMD_UNSUBSCRIBE:
		ret = unsubscribe(this.Client, this.params[0])

	default:
		return "", errors.New("Cmd not found: " + this.cmd)
	}

	return ret, nil
}

func trim(str string) string {
	return strings.TrimRight(str, string([]rune{0, 13, 10}))
}
