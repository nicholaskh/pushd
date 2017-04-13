package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	log "github.com/nicholaskh/log4go"
	"bytes"
)

type Cmdline struct {
	Cmd    string
	Params []string
	*Client
}

const (
	CMD_SENDMSG	= "sendmsg"
	CMD_SETUUID	= "setuuid"
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
	// when msg send by a client
	if cli != nil{
		parts := strings.SplitN(trimCmdline(input), " ", 3)
		this.Cmd = parts[0]
		this.Params = parts[1:]
	}else {
		parts := strings.SplitN(trimCmdline(input), " ", 5)
		this.Cmd = parts[0]
		this.Params = parts[1:]
	}

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
		ret = Subscribe(this.Client, this.Params[0], 1)

	case CMD_PUBLISH:
		//		if !this.Client.IsClient() && !this.Client.IsServer() {
		//			return "", ErrNotPermit
		//		}
		if len(this.Params) < 2 || this.Params[1] == "" {
			return "", errors.New("Publish without msg\n")
		} else {
			ret = Publish(this.Params[0], this.Params[1], this.Client.uuid, false)
		}


	case CMD_SENDMSG:

		if len(this.Params) < 2 || this.Params[1] == "" {
			return "", errors.New("send no msg\n")
		} else if this.Client.uuid == "" {
			return "", errors.New("client is not set uuid\n")
		} else if bytes.Compare([]byte(this.Params[0]), []byte(this.Client.uuid)) == 0 {

			return "", errors.New("target is yourself")
		}

		var channel string
		if bytes.Compare([]byte(this.Client.uuid), []byte(this.Params[0])) > 0 {
			channel = fmt.Sprintf("pri_%s_%s", this.Client.uuid, this.Params[0])
		} else {
			channel = fmt.Sprintf("pri_%s_%s", this.Params[0], this.Client.uuid)
		}

		_, exists := this.Client.Channels[channel]
		if !exists {
			friend, exists := UuidToClient.GetClient(this.Params[0])
			if !exists {
				Subscribe(this.Client, channel, 2)
			}else{
				Subscribe(this.Client, channel, -1)
				Subscribe(friend, channel, -1)
			}
		}

		ret = Publish(channel, this.Params[1], this.Client.uuid, false)

	case CMD_UNSUBSCRIBE:
		//		if !this.Client.IsClient() {
		//			return "", ErrNotPermit
		//		}
		if len(this.Params) < 1 || this.Params[0] == "" {
			return "", errors.New("Lack unsub channel")
		}
		ret = Unsubscribe(this.Client, this.Params[0])

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
	case CMD_SETUUID:
		if len(this.Params) < 1 || this.Params[0] == "" {
			return "", errors.New("Lack uuid")
		}
		this.Client.uuid = this.Params[0]

		_, exists := UuidToClient.GetClient(this.Params[0])
		if !exists {
			UuidToClient.AddClient(this.Params[0], this.Client)
		}

		ret = "uuid saved"
		err = nil

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
