package engine

import (
	"errors"
	"fmt"
	"strings"

	"github.com/nicholaskh/golib/str"
	"github.com/nicholaskh/pushd/client"
)

type Cmdline struct {
	Cmd    string
	Params []string
	*client.Client
}

const (
	CMD_SUBSCRIBE   = "sub"
	CMD_PUBLISH     = "pub"
	CMD_UNSUBSCRIBE = "unsub"
	CMD_TOKEN       = "gettoken"
	CMD_AUTH_CLIENT = "auth_client"
	CMD_AUTH_SERVER = "auth_server"

	OUTPUT_SUBSCRIBED         = "SUBSCRIBED"
	OUTPUT_ALREADY_SUBSCRIBED = "ALREADY SUBSCRIBED"
	OUTPUT_PUBLISHED          = "PUBLISHED"
	OUTPUT_NOT_SUBSCRIBED     = "NOT SUBSCRIBED"
	OUTPUT_UNSUBSCRIBED       = "UNSUBSCRIBED"
)

func NewCmdline(input string, cli *client.Client) (this *Cmdline) {
	this = new(Cmdline)
	parts := strings.SplitN(trimCmdline(input), " ", 3)
	this.Cmd = parts[0]
	this.Params = parts[1:]
	this.Client = cli
	return
}

func (this *Cmdline) Process() (ret string, err error) {
	switch this.Cmd {
	case CMD_SUBSCRIBE:
		ret = subscribe(this.Client, this.Params[0])

	case CMD_PUBLISH:
		if len(this.Params) < 2 {
			return "", errors.New("Publish without msg\n")
		} else {
			ret = publish(this.Params[0], this.Params[1], false)
		}

	case CMD_UNSUBSCRIBE:
		ret = unsubscribe(this.Client, this.Params[0])

	//use one appId/secretKey pair
	case CMD_AUTH_SERVER:
		if this.Client.Authed {
			ret = "Already authed"
			err = nil
		} else {
			ret, err = authServer(this.Params[0], this.Params[1])
			if err == nil {
				this.Client.Authed = true
				this.Client.Type = client.TYPE_SERVER
			}
		}

	case CMD_TOKEN:
		// TODO more secure token generator
		ret = str.Rand(32)
		tokenPool.Set(ret, 1)

	case CMD_AUTH_CLIENT:
		if this.Client.Authed {
			ret = "Already authed"
			err = nil
		} else {
			ret, err = authClient(this.Params[0])
			if err == nil {
				this.Client.Authed = true
				this.Client.Type = client.TYPE_CLIENT
			}
		}

	default:
		return "", errors.New(fmt.Sprintf("Cmd not found: %s\n", this.Cmd))
	}

	return
}

func trimCmdline(str string) string {
	return strings.TrimRight(str, string([]rune{0, 13, 10}))
}
