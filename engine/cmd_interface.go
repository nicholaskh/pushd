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
	"time"
	"bytes"
	"encoding/binary"
)

type Cmdline struct {
	Cmd    string
	Params string
	Params2 []byte
	*Client
}

const (
	CMD_OFFLINE_MSG	= "ofli_msg"
	CMD_CREATE_USER	= "user_create"
	CMD_ACK_MSG	= "ack"
	CMD_VIDO_CHAT	= "vido"
	CMD_DISOLVE	= "disolve"
	CMD_UNSUBS 	= "unsubs"
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
	CMD_FRAME_APPLY = "frame_apply"


	OUTPUT_VIDO_CHAT	 = "bina"
	OUTPUT_TOKEN 	           = "TOKEN"
	OUTPUT_SUBS		    = "SUBS"
	OUTPUT_AUTH_SERVER        = "AUTHSERVER"
	OUTPUT_RCIV	            = "RCIV"
	OUTPUT_APPKEY             = "APPKEY"
	OUTPUT_SUBSCRIBED         = "SUBSCRIBED"
	OUTPUT_ALREADY_SUBSCRIBED = "ALREADY SUBSCRIBED"
	OUTPUT_NOT_SUBSCRIBED     = "NOT SUBSCRIBED"
	OUTPUT_UNSUBSCRIBED       = "UNSUBSCRIBED"
	OUTPUT_PONG               = "pong"
	OUTPUT_CREATEROOM = "CREATEROOM"
	OUTPUT_JOINROOM  = "JOINROOM"
	OUTPUT_LEAVEROOM = "LEAVEROOM"
)

const (
 TYPE_SINGLE_VOICE  = 1
 TYPE_MUL_VOICE  = 2
 TYPE_SINGLE_VIDEO  = 3
 TYPE_MUL_VIDEO  = 4
)

func NewCmdline(input []byte, cli *Client) (this *Cmdline, err error) {

	var headerLen int32
	if len(input) < 4 {
		this = nil
		err = errors.New("message has damaged")
		return
	}

	b_buf := bytes.NewBuffer(input[:4])
	binary.Read(b_buf, binary.BigEndian, &headerLen)
	headL := int(headerLen)

	if headL <= 0 {
		this = nil
		err = errors.New("skip")
		return
	}

	if len(input) < headL+4 {
		this = nil
		err = errors.New("message has damaged")
		return
	}

	this = new(Cmdline)
	this.Cmd = string(input[4: headL+4])

	if len(input) > headL + 4 + 4 {
		b_buf = bytes.NewBuffer(input[headL+4: headL+4+4])
		binary.Read(b_buf, binary.BigEndian, &headerLen)
		bodyL := int(headerLen)
		if len(input) != headL + 4 + 4 + bodyL {
			this = nil
			err = errors.New("message has damaged")
			return
		}
		if this.Cmd == CMD_VIDO_CHAT {
			this.Params2 = input[headL+4+4: headL+4+4+bodyL]
		} else {
			this.Params = string(input[headL+4+4: headL+4+4+bodyL])
		}
	}

	this.Client = cli
	return
}

func (this *Cmdline) Process() (ret string, err error) {
	switch this.Cmd {
	case CMD_SENDMSG:
		params := strings.SplitN(this.Params, " ", 3)
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

		msgId, err := strconv.ParseInt(params[1], 10, 64)
		if err != nil {
			return "", errors.New("msgid error")
		}

		if this.Client.msgIdCache.CheckAndSet(msgId) {
			ret = fmt.Sprintf("%d %d", msgId, time.Now().UnixNano());
			return ret, nil
		}

		ret = Publish(params[0], params[2], this.Client.uuid, msgId, false)

	case CMD_VIDO_CHAT:
		len := 0
		for i, value := range this.Params2 {
			if value == ' ' {
				len = i
				break
			}
		}
		if len == 0 {
			return
		}
		channelId := string(this.Params2[: len])

		_, exists := this.Client.Channels[channelId]
		if !exists {
			Subscribe(this.Client, channelId)

			// force other related online clients to join in this channel
			var result interface{}
			err := db.MgoSession().DB("pushd").C("channel_uuids").
				Find(bson.M{"_id": channelId}).
				Select(bson.M{"uuids":1, "_id":0}).
				One(&result)

			if err == nil {
				uuids := result.(bson.M)["uuids"].([]interface{})
				for _, uuid := range uuids {
					tclient, exists := UuidToClient.GetClient(uuid.(string))
					if exists {
						Subscribe(tclient, channelId)
					}
				}
			}

		}

		Forward(channelId, this.Client.uuid, this.Params2[len+1:], false)

	case CMD_ACK_MSG:
		params := strings.SplitN(this.Params, " ", 2)
		if len(params) != 2 {
			return "", errors.New("params number error")
		}
		msgId, err := strconv.ParseInt(params[1], 10, 64)
		if err != nil {
			return "", errors.New("msgid error")
		}
		this.Client.AckMsg(msgId, params[0])

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
		params := strings.SplitN(this.Params, " ", 3)
		if len(params) < 3 || params[2] == "" {
			return "", errors.New("Publish without msg\n")
		} else {
			msgId, err := strconv.ParseInt(params[1], 10, 64)
			if err != nil {
				return "", errors.New("msgid error")
			}

			if this.Client.msgIdCache.CheckAndSet(msgId) {
				ret = fmt.Sprintf("%d %d", msgId, time.Now().UnixNano());
				return ret, nil
			}

			ret = Publish(params[0], params[2], this.Client.uuid, msgId, false)
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

	case CMD_FRAME_APPLY:
		params := strings.Split(this.Params, " ")
		if len(params) != 2 {
			return "", errors.New("errorparam wrong")
		}

		mainType, _err0 :=  strconv.Atoi(params[0])
		if _err0 != nil {
			return "", errors.New("errorparam wrong")
		}

		newChannelId := fmt.Sprintf("%s_", params[1])
		activeUser := []string{this.Client.uuid}
		collection := db.MgoSession().DB("pushd").C("unstable_info")
		// create temp channle
		_, err0 := collection.Upsert(
			bson.M{"_id": newChannelId},
			bson.M{"type": 1,
				"subtype": mainType,
				"proposer": this.Client.uuid,
				"channelId": params[1],
				"time": time.Now().Unix(),
				"activeUser": activeUser})

		if err0 != nil {
			ret = fmt.Sprintf("error%s", err0.Error())
			return
		}

		// update userInfo
		uuids := storage.FetchUuidsAboutChannel(params[1])
		var documents []interface{}
		for _, userId := range uuids {
			documents = append(documents, bson.M{"_id": userId})
			documents = append(documents, bson.M{"$push": bson.M{"frame_chat": newChannelId}})
		}

		bulk := db.MgoSession().DB("pushd").C("user_info").Bulk()
		bulk.Upsert(documents...)
		_, err0 = bulk.Run()
		if err0 != nil {
			collection.RemoveId(newChannelId)
		}

		// push notify according to type
		notice := fmt.Sprintf("%s %d %s %s", CMD_FRAME_APPLY, mainType, this.Client.uuid, params[0])
		Publish2(params[1], notice, true)
		ret = "success"

    //subs: subscribe from server
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

	case CMD_UNSUBS:
		params := strings.Split(this.Params, " ")
		if len(params) < 2 {
			return "", errors.New("param wrong")
		}
		leaveRoom(params[0], params[1:]...)
		ret = "success"

	case CMD_DISOLVE:
		if this.Params == "" {
			return "", errors.New("param wrong")
		}
		disolveRoom(this.Params)
		ret = "success"

	case CMD_CREATE_USER:
		if this.Params == "" {
			return "", errors.New("param is empty")
		}
		coll := db.MgoSession().DB("pushd").C("user_info")

		var user interface{}
		coll.FindId(this.Params).One(&user)
		if user != nil {
			return "user exists", nil
		}
		err := coll.Insert(bson.M{"_id": this.Params, "channel_stat": bson.M{}})
		if err != nil {
			return "create error", nil
		}

		return "success", nil

	case CMD_OFFLINE_MSG:
		coll := db.MgoSession().DB("pushd").C("user_info")
		var result interface{}
		coll.FindId(this.Client.uuid).Select(bson.M{"_id":0, "channel_stat":1}).One(&result)
		if result == nil {
			return "{}", nil
		}
		userInfo := result.(bson.M)
		channelStat := userInfo["channel_stat"].(bson.M)

		data := make(map[string][]interface{})
		for channel, va := range channelStat {
			ts := va.(int64)
			hisRet, err := fullHistory(channel, ts)
			if err != nil {
				continue
			}
			if len(hisRet) > 0 {
				data[channel] = hisRet
				for _, msg := range hisRet {
					msgInfo := msg.(bson.M)
					msgid := msgInfo["msgid"].(int64)
					ts := msgInfo["ts"].(int64)
					this.Client.ackList.push(channel, ts, msgid)
				}
			}
		}

		var retBytes []byte
		retBytes, err = json.Marshal(data)
		ret = string(retBytes)


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
				ret = fmt.Sprintf("%s %s", OUTPUT_AUTH_SERVER, ret)
			}
			return ret, err
		}

	case CMD_TOKEN:
		//if !this.Client.IsServer() {
		//	return "", ErrNotPermit
		//}
		token := getClientToken()
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
				// set token
				this.Client.initToken(params[0], time.Now().UnixNano())
			}
		}

	case CMD_SETUUID:
		params := strings.Split(this.Params, " ")
		if len(params) < 2 || params[1] == "" {
			return "", errors.New("Lack uuid")
		}
		this.Client.initChatEnv(params[1])
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
