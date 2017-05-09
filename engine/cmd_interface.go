package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/engine/storage"
	"github.com/nicholaskh/pushd/db"
	"gopkg.in/mgo.v2/bson"
	"github.com/nicholaskh/pushd/config"
	"time"
)

type Cmdline struct {
	Cmd    string
	Params string
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


	OUTPUT_TOKEN 	           = "TOKEN"
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
	parts := strings.SplitN(trimCmdline(input), " ", 2)
	this.Cmd = parts[0]
	if len(parts) == 2 {
		this.Params = parts[1]
	}

	this.Client = cli
	return
}

func (this *Cmdline) Process() (ret string, err error) {
	switch this.Cmd {
	case CMD_SENDMSG:
		params := strings.SplitN(this.Params, " ", 2)
		if len(params) < 2 || params[1] == "" {
			return "", errors.New("Lack msg\n")
		}

		_, exists := this.Client.Channels[params[0]]
		if !exists {
			Subscribe(this.Client, params[0])

			// force other related online clients to join in this channel
			var result interface{}
			err := db.MgoSession().DB("pushd").C("channel_uuids").
				Find(bson.M{"_id": params[0]}).
				Select(bson.M{"uuids":1, "_id":0}).
				One(&result)

			if err == nil {
				uuids := result.(bson.M)["uuids"].([]interface{})
				for _, uuid := range uuids {
					tclient, exists := UuidToClient.GetClient(uuid.(string))
					if exists {
						Subscribe(tclient, params[0])
					}
				}
			}

		}

		ret = Publish(params[0], params[1], this.Client.uuid, false)

	case CMD_SUBSCRIBE:
		//		if !this.Client.IsClient() {
		//			return "", ErrNotPermit
		//		}
		if this.Params == "" {
			return "", errors.New("Lack sub channel")
		}
		ret = Subscribe(this.Client, this.Params)

	case CMD_PUBLISH:
		//		if !this.Client.IsClient() && !this.Client.IsServer() {
		//			return "", ErrNotPermit
		//		}
		params := strings.SplitN(this.Params, " ", 2)
		if len(params) < 2 || params[1] == "" {
			return "", errors.New("Publish without msg\n")
		} else {
			ret = Publish(params[0], params[1], this.Client.uuid, false)
		}

	case CMD_UNSUBSCRIBE:
		//		if !this.Client.IsClient() {
		//			return "", ErrNotPermit
		//		}
		params := strings.SplitN(this.Params, " ", 2)
		if len(params) < 1 || params[0] == "" {
			return "", errors.New("Lack unsub channel")
		}
		ret = Unsubscribe(this.Client, params[0])

	case CMD_CREATEROOM:
		params := strings.Split(this.Params, " ")
		if len(params) < 1 {
			return "", errors.New("Lack uuid")
		}
		if this.Client.uuid == "" {
			return "", errors.New("client has no uuid")
		}

		roomid := generateRoomIdByUuidList(append(params, this.Client.uuid)...)
		channelId := roomid2Channelid(roomid)
		ret = createRoom(this.Client.uuid, channelId, channelId)

		for _, uuid := range params {
			joinRoom(channelId, uuid)
		}

		storage.EnqueueChanUuids("", channelId, false, params)

	case CMD_JOINROOM:
		params := strings.Split(this.Params, " ")
		if len(params) < 1 || params[0] == "" {
			return "", errors.New("Lack roomid")
		}

		if this.Client.uuid == "" {
			return "", errors.New("client must setuuid first")
		}

		channelId := roomid2Channelid(params[0])
		ret = joinRoom(channelId, this.Client.uuid)

		uuids := []string{this.Client.uuid}
		storage.EnqueueChanUuids("", channelId, false, uuids)


	case CMD_LEAVEROOM:
		params := strings.Split(this.Params, " ")
		if len(params) < 1 || params[0] == "" {
			return "", errors.New("Lack roomid")
		}
		ret = leaveRoom(this.Client.uuid, roomid2Channelid(params[0]))

	case CMD_SUBS:
		params := strings.Split(this.Params, " ")
		if len(params) < 3 {
			return "", errors.New("param wrong")
		}

		createRoom(params[2], params[1], params[0])

		for _, uuid := range params[3:] {
			joinRoom(params[1], uuid)
		}
		if len(params[3:]) > 0{
			storage.EnqueueChanUuids("", params[1], false, params[3:])
		}

		ret = fmt.Sprintf("%s success", OUTPUT_SUBS)

	case CMD_HISTORY:
		//		if !this.Client.IsClient() {
		//			return "", ErrNotPermit
		//		}
		params := strings.Split(this.Params, " ")
		if len(params) < 2 {
			return "", errors.New("Invalid Params for history")
		}
		ts, err := strconv.ParseInt(params[1], 10, 64)
		if err != nil {
			return "", err
		}
		channel := params[0]
		hisRet, err := fullHistory(channel, ts)
		if err != nil {
			log.Error(err)
		}

		var retBytes []byte
		retBytes, err = json.Marshal(hisRet)

		ret = string(retBytes)

	//use one appId/secretKey pair
	case CMD_AUTH_SERVER:
		if this.Params == "" {
			return "", errors.New("Invalid Params for auth_server")
		}
		if this.Client.IsServer() {
			ret = "Already authed server"
			err = nil
		} else {
			ret, err = authServer(this.Params)
			if err == nil {
				this.Client.SetServer()
				ret = fmt.Sprintf("%s success", OUTPUT_AUTH_SERVER)
			}
		}

	case CMD_TOKEN:
		//if !this.Client.IsServer() {
		//	return "", ErrNotPermit
		//}
		token := getToken()
		if token == "" {
			return "", errors.New("gettoken error")
		}

		ret = fmt.Sprintf("%s %s", OUTPUT_TOKEN, token)

	case CMD_APPKEY:
		ret = fmt.Sprintf("%s %s", OUTPUT_APPKEY, getAppKey())

	case CMD_AUTH_CLIENT:
		params := strings.Split(this.Params, " ")
		if len(params) < 1 {
			return "", errors.New("Invalid Params for auth_client")
		}
		if this.Client.IsClient() {
			ret = "Already authed client"
			err = nil
		} else {
			ret, err = authClient(params[0])
			if err == nil {
				this.Client.SetClient()
				this.Client.token = params[0]
				this.Client.lastTimestamp = time.Now().Unix()
			}
		}

	case CMD_SETUUID:
		params := strings.Split(this.Params, " ")
		if len(params) < 1 || params[0] == "" {
			return "", errors.New("Lack uuid")
		}
		this.Client.uuid = params[0]

		_, exists := UuidToClient.GetClient(params[0])
		if !exists {
			UuidToClient.AddClient(params[0], this.Client)

			// check and force client to subscribe related and active channels
			var result interface{}
			err := db.MgoSession().DB("pushd").C("uuid_channels").
				Find(bson.M{"_id": params[0]}).
				Select(bson.M{"_id": 0, "channels": 1}).
				One(&result)

			if err == nil {
				channels := result.(bson.M)["channels"].([]interface{})
				for _, value := range channels {
					channel := value.(string)
					// check native
					_, exists = PubsubChannels.Get(channel)
					if !exists {
						// check other nodes
						if config.PushdConf.IsDistMode() {
							_, exists = Proxy.Router.LookupPeersByChannel(channel)
						}
					}

					if exists {
						Subscribe(this.Client, channel)
					}
				}
			}
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
