package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/engine/storage"
)

type Cmdline struct {
	Cmd    string
	Params []string
	*Client
}

const (
	CMD_SUBS	= "subs"
	CMD_APPKEY    = "getappkey"
	CMD_SENDMSG   = "sendmsg"
	CMD_SETUUID	= "setuuid"
	CMD_SUBSCRIBE   = "sub"
	CMD_PUBLISH     = "pub"
	CMD_UNSUBSCRIBE = "unsub"
	CMD_HISTORY     = "his"
	CMD_TOKEN       = "gettoken"
	CMD_AUTH_CLIENT = "auth_client"
	CMD_AUTH_SERVER = "auth_server"
	CMD_PING        = "ping"
	CMD_CREATEROOM  = "create_room"
	CMD_JOINROOM = "join_room"
	CMD_LEAVEROOM = "leave_room"


	OUTPUT_SUBS		    = "SUBS"
	OUTPUT_AUTH_SERVER        = "AUTHSERVER"
	OUTPUT_RCIV	            = "RCIV"
	OUTPUT_APPKEY             = "APPKEY"
	OUTPUT_SUBSCRIBED         = "SUBSCRIBED"
	OUTPUT_ALREADY_SUBSCRIBED = "ALREADY SUBSCRIBED"
	OUTPUT_PUBLISHED          = "PUBLISHED"
	OUTPUT_NOT_SUBSCRIBED     = "NOT SUBSCRIBED"
	OUTPUT_UNSUBSCRIBED       = "UNSUBSCRIBED"
	OUTPUT_PONG               = "pong"
	OUTPUT_CREATEROOM = "CREATEROOM"
	OUTPUT_JOINROOM  = "JOINROOM"
	OUTPUT_LEAVEROOM = "LEAVEROOM"
)

func NewCmdline(input string, cli *Client) (this *Cmdline) {
	this = new(Cmdline)
	parts := strings.Split(trimCmdline(input), " ")
	this.Cmd = parts[0]
	this.Params = parts[1:]

	this.Client = cli
	return
}

func (this *Cmdline) Process() (ret string, err error) {
	switch this.Cmd {
	case CMD_SENDMSG:
		if len(this.Params) < 2 || this.Params[1] == "" {
			return "", errors.New("Lack msg\n")
		}
		ret = Publish(this.Params[0], this.Params[1], this.Client.uuid, false)

	case CMD_SUBSCRIBE:
		//		if !this.Client.IsClient() {
		//			return "", ErrNotPermit
		//		}
		if len(this.Params) < 1 || this.Params[0] == "" {
			return "", errors.New("Lack sub channel")
		}
		ret = Subscribe(this.Client, this.Params[0])

	case CMD_PUBLISH:
		//		if !this.Client.IsClient() && !this.Client.IsServer() {
		//			return "", ErrNotPermit
		//		}
		if len(this.Params) < 2 || this.Params[1] == "" {
			return "", errors.New("Publish without msg\n")
		} else {
			ret = Publish(this.Params[0], this.Params[1], this.Client.uuid, false)
		}

	case CMD_UNSUBSCRIBE:
		//		if !this.Client.IsClient() {
		//			return "", ErrNotPermit
		//		}
		if len(this.Params) < 1 || this.Params[0] == "" {
			return "", errors.New("Lack unsub channel")
		}
		ret = Unsubscribe(this.Client, this.Params[0])

	case CMD_CREATEROOM:
		if len(this.Params) < 1 {
			return "", errors.New("Lack uuid")
		}
		if this.Client.uuid == "" {
			return "", errors.New("client has no uuid")
		}

		roomid := generateRoomIdByUuidList(append(this.Params, this.Client.uuid)...)
		channelId := roomid2Channelid(roomid)
		ret = createRoom(this.Client.uuid, channelId, channelId)

		for _, uuid := range this.Params {
			joinRoom(channelId, uuid)
		}

		storage.EnqueueChanUuids("", channelId, false, this.Params)

	case CMD_JOINROOM:
		if len(this.Params) < 1 || this.Params[0] == "" {
			return "", errors.New("Lack roomid")
		}

		if this.Client.uuid == "" {
			return "", errors.New("client must setuuid first")
		}

		channelId := roomid2Channelid(this.Params[0])
		ret = joinRoom(channelId, this.Client.uuid)

		uuids := []string{this.Client.uuid}
		storage.EnqueueChanUuids("", channelId, false, uuids)


	case CMD_LEAVEROOM:
		if len(this.Params) < 1 || this.Params[0] == "" {
			return "", errors.New("Lack roomid")
		}
		ret = leaveRoom(this.Client.uuid, roomid2Channelid(this.Params[0]))

	case CMD_SUBS:
		if len(this.Params) < 3 {
			return "", errors.New("param wrong")
		}

		createRoom(this.Params[2], this.Params[1], this.Params[0])

		for _, uuid := range this.Params[3:] {
			joinRoom(this.Params[1], uuid)
		}
		if len(this.Params[3:]) > 0{
			storage.EnqueueChanUuids("", this.Params[1], false, this.Params[3:])
		}

		ret = fmt.Sprintf("%s success", OUTPUT_SUBS)

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
		if len(this.Params) < 1 {
			return "", errors.New("Invalid Params for auth_server")
		}
		if this.Client.IsServer() {
			ret = "Already authed server"
			err = nil
		} else {
			ret, err = authServer(this.Params[0])
			if err == nil {
				this.Client.SetServer()
				ret = fmt.Sprintf("%s success", OUTPUT_AUTH_SERVER)
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

	case CMD_APPKEY:
		ret = fmt.Sprintf("%s %s", OUTPUT_APPKEY, getAppKey())

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
