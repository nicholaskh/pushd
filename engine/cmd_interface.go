package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	log "github.com/nicholaskh/log4go"
)

type Cmdline struct {
	Cmd    string
	Params []string
	*Client
}

const (
	CMD_SET		= "set"
	CMD_SUBSCRIBE   = "sub"
	CMD_PUBLISH     = "pub"
	CMD_UNSUBSCRIBE = "unsub"
	CMD_HISTORY     = "his"
	CMD_TOKEN       = "gettoken"
	CMD_AUTH_CLIENT = "auth_client"
	CMD_AUTH_SERVER = "auth_server"
	CMD_PING        = "ping"

	OUTPUT_SUBSCRIBED         = "SUBSCRIBED"
	OUTPUT_ALREADY_SUBSCRIBED = "ALREADY SUBSCRIBED"
	OUTPUT_PUBLISHED          = "PUBLISHED"
	OUTPUT_NOT_SUBSCRIBED     = "NOT SUBSCRIBED"
	OUTPUT_UNSUBSCRIBED       = "UNSUBSCRIBED"
	OUTPUT_PONG               = "pong"
)

func NewCmdline(input string, cli *Client) (this *Cmdline) {
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
		//		if !this.Client.IsClient() {
		//			return "", ErrNotPermit
		//		}
		if len(this.Params) < 1 || this.Params[0] == "" {
			return "", errors.New("Lack sub channel")
		}
		ret = subscribe(this.Client, this.Params[0])

	case CMD_PUBLISH:
		//		if !this.Client.IsClient() && !this.Client.IsServer() {
		//			return "", ErrNotPermit
		//		}
		if len(this.Params) < 2 || this.Params[1] == "" {
			return "", errors.New("Publish without msg\n")
		} else {
			ret = publish(this.Params[0], this.Params[1], false)
		}

	case CMD_UNSUBSCRIBE:
		//		if !this.Client.IsClient() {
		//			return "", ErrNotPermit
		//		}
		if len(this.Params) < 1 || this.Params[0] == "" {
			return "", errors.New("Lack unsub channel")
		}
		ret = unsubscribe(this.Client, this.Params[0])

	case CMD_HISTORY:
		//		if !this.Client.IsClient() {
		//			return "", ErrNotPermit
		//		}
		if len(this.Params) < 2 {
			return "", errors.New("Invalid Params for history")
		}
		ts, err := strconv.ParseInt(this.Params[1], 10, 64)
		if err != nil {
			return "", err
		}
		channel := this.Params[0]
		hisRet, err := fullHistory(channel, ts)
		if err != nil {
			log.Error(err)
		}

		var retBytes []byte
		retBytes, err = json.Marshal(hisRet)

		ret = string(retBytes)

	//use one appId/secretKey pair
	case CMD_AUTH_SERVER:
		if len(this.Params) < 2 {
			return "", errors.New("Invalid Params for auth_server")
		}
		if this.Client.IsServer() {
			ret = "Already authed server"
			err = nil
		} else {
			ret, err = authServer(this.Params[0], this.Params[1])
			if err == nil {
				this.Client.SetServer()
			}
		}

	case CMD_TOKEN:
		//if !this.Client.IsServer() {
		//	return "", ErrNotPermit
		//}
		ret = getToken()
		if ret == "" {
			return "", errors.New("gettoken error")
		}

	case CMD_AUTH_CLIENT:
		if len(this.Params) < 1 {
			return "", errors.New("Invalid Params for auth_client")
		}
		if this.Client.IsClient() {
			ret = "Already authed client"
			err = nil
		} else {
			ret, err = authClient(this.Params[0])
			if err == nil {
				this.Client.SetClient()
			}
		}
	case CMD_SET:
		if len(this.Params) < 2 || this.Params[0] == "" {
			return "", errors.New("miss parameter\n")
		}
		if this.Params[0] == ""{
			return "", errors.New("value is null\n")
		}
		switch this.Params[0]{
		case "uuid":
			this.Client.SetUUID(this.Params[1])
			ret = "uuid saved"
			err = nil
		default:
			return "", errors.New("Invalid attribute\n")
		}

	case CMD_PING:
		return OUTPUT_PONG, nil

	default:
		return "", errors.New(fmt.Sprintf("Cmd not found: %s\n", this.Cmd))
	}

	return
}

func trimCmdline(str string) string {
	return strings.TrimRight(str, string([]rune{0, 13, 10}))
}
