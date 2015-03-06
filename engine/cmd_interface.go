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
	CMD_AUTH        = "auth"

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

func (this *Cmdline) ProcessCmd() (ret string, err error) {
	//if this.Cmd != CMD_TOKEN && this.Cmd != CMD_AUTH {
	//	if err = checkLogin(this.Client.Uname); err != nil {
	//		return "", err
	//	}
	//}

	switch this.Cmd {
	case CMD_SUBSCRIBE:
		ret = subscribe(this.Client, this.Params[0])

	case CMD_PUBLISH:
		if len(this.Params) < 2 {
			return "", errors.New("Publish without msg\n")
		} else {
			ret = publish(this.Params[0], this.Params[1])
		}

	case CMD_UNSUBSCRIBE:
		ret = unsubscribe(this.Client, this.Params[0])

	// TODO use one appId/secretKey pair
	case CMD_TOKEN:
		// TODO more secure token generator
		ret = str.Rand(32)
		userToken.Set(this.Params[0], ret)

	case CMD_AUTH:
		ret, err = authStep(this.Params[0], this.Params[1])
		if err == nil {
			this.Client.Uname = this.Params[0]
		}

	default:
		return "", errors.New(fmt.Sprintf("Cmd not found: %s\n", this.Cmd))
	}

	return
}

func trimCmdline(str string) string {
	return strings.TrimRight(str, string([]rune{0, 13, 10}))
}
