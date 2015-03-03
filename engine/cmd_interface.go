package engine

import (
	"errors"
	"fmt"
	"strings"

	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/client"
)

type Cmdline struct {
	Cmd    string
	Params []string
	*client.Client
}

const (
	CMD_SUBSCRIB    = "sub"
	CMD_PUBLISH     = "pub"
	CMD_UNSUBSCRIBE = "unsub"

	OUTPUT_SUBSCRIBED         = "SUBSCRIBED"
	OUTPUT_ALREADY_SUBSCRIBED = "ALREADY SUBSCRIBED"
	OUTPUT_PUBLISHED          = "PUBLISHED"
	OUTPUT_NOT_SUBSCRIBED     = "NOT SUBSCRIBED"
	OUTPUT_UNSUBSCRIBED       = "UNSUBSCRIBED"
)

func NewCmdline(input []byte, cli *client.Client) (this *Cmdline) {
	this = new(Cmdline)
	parts := strings.Split(trimCmdline(string(input)), " ")
	this.Cmd = parts[0]
	this.Params = parts[1:]
	this.Client = cli
	return
}

func (this *Cmdline) ProcessCmd() (string, error) {
	log.Debug("Input cmd: %s %s", this.Cmd, this.Params)
	var ret string
	switch this.Cmd {
	case CMD_SUBSCRIB:
		ret = subscribe(this.Client, this.Params[0])

	case CMD_PUBLISH:
		if len(this.Params) < 2 {
			return "", errors.New("Publish without msg\n")
		} else {
			ret = publish(this.Params[0], this.Params[1])
		}

	case CMD_UNSUBSCRIBE:
		ret = unsubscribe(this.Client, this.Params[0])

	default:
		return "", errors.New(fmt.Sprintf("Cmd not found: %s\n", this.Cmd))
	}

	return ret, nil
}

func trimCmdline(str string) string {
	return strings.TrimRight(str, string([]rune{0, 13, 10}))
}
