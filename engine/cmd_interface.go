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
	"gopkg.in/mgo.v2"
	"github.com/nicholaskh/pushd/engine/offpush"
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
	CMD_FRAME_JOIN = "frame_join"
	CMD_FRAME_OUT = "frame_out"
	CMD_FRAME_ACCEPT = "frame_accept"
	CMD_FRAME_DISMISS = "frame_dismiss"
	CMD_FRAME_REFUSE = "frame_refuse"
	CMD_FRAME_INFO	= "frame_info"
	CMD_INVOKE_FRAME_ACTION = "frame_action"

	CMD_RETRACT_MESSAGE	= "retract_msg"
	CMD_UPDATE_OR_ADD_PUSH_ID = "up_ad_pushId"
	CMD_SET_OFF_NOTIFY = "set_notify"


	OUTPUT_FRAME_CHAT	= "FRAMECHAT"
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
	UNSTABLE_INFO_TYPE_FRAME_CHAT = 1
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
		params := strings.SplitN(this.Params, " ", 4)
		if len(params) < 4 {
			return "", errors.New("Lack msg\n")
		}

		isResend := params[0]
		channel := params[1]
		tempMsgId := params[2]
		msg := params[3]

		msgId, err := strconv.ParseInt(tempMsgId, 10, 64)
		if err != nil {
			return "", errors.New("msgid error")
		}

		// check if this message has been sent
		if isResend == "Y" {
			isHit := db.MgoSession().DB("pushd").C("msg_log").
					Find(bson.M{"channel": channel,
						"uuid": this.Client.uuid,
						"msgid": msgId}).One(nil)
			if isHit == nil {
				ret = fmt.Sprintf("%d %d", msgId, time.Now().UnixNano())
				return ret, nil
			}
		}

		_, exists := this.Client.Channels[channel]
		if !exists {
			Subscribe(this.Client, channel)

			// force other related online clients to join in this channel
			var result interface{}
			err := db.MgoSession().DB("pushd").C("channel_uuids").
				Find(bson.M{"_id": channel}).
				Select(bson.M{"uuids":1, "_id":0}).
				One(&result)

			if err == nil {
				uuids := result.(bson.M)["uuids"].([]interface{})
				for _, uuid := range uuids {
					tclient, exists := UuidToClient.GetClient(uuid.(string))
					if exists {
						Subscribe(tclient, channel)
					}
				}
			}

		}

		offpush.CheckAndPush(channel, msg, this.Client.uuid)
		ret = Publish(channel, msg, this.Client.uuid, msgId, false)

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

	case CMD_RETRACT_MESSAGE:
		if this.Params == "" {
			return "", errors.New(fmt.Sprintf("%d param is empty", CODE_PARAM_ERROR))
		}
		params := strings.SplitN(this.Params, " ", 3)
		if len(params) < 3 {
			return "", errors.New(fmt.Sprintf("%d Lack of param", CODE_PARAM_ERROR))
		}

		channel := params[0]
		msgId, err := strconv.ParseInt(params[1], 10, 64)
		if err != nil {
			return "", errors.New(fmt.Sprintf("%d msgId can not parse", CODE_PARAM_ERROR))
		}

		newMsgId, err := strconv.ParseInt(params[2], 10, 64)
		if err != nil {
			return "", errors.New(fmt.Sprintf("%d msgId can not parse", CODE_PARAM_ERROR))
		}

		col := db.MgoSession().DB("pushd").C("msg_log")
		err = col.Remove(bson.M{"channel": channel, "uuid": this.uuid, "msgid": msgId})

		if err != nil {
			if realError, ok := err.(interface{}).(*mgo.LastError); ok {
				return "", errors.New(fmt.Sprintf("%d %s",
					CODE_SERVER_ERROR, realError.Error()))
			}

			if err == mgo.ErrNotFound {
				return fmt.Sprintf("%d not found", CODE_FAILED), nil
			}
		}

		Publish(channel, fmt.Sprintf("[del] %s %d", this.uuid, msgId),
			this.Client.uuid, newMsgId, false)

		return fmt.Sprintf("%d success", CODE_SUCCESS), nil

	case CMD_UPDATE_OR_ADD_PUSH_ID:
		params := strings.Split(this.Params, " ")
		userId := params[0]
		pushId := params[1]

		if userId == "" || pushId == "" {
			return fmt.Sprintf("%d param error", CODE_PARAM_ERROR), nil
		}

		coll := db.MgoSession().DB("pushd").C("user_info")
		err := coll.FindId(userId).One(nil)
		if err != nil {
			if err == mgo.ErrNotFound {
				return fmt.Sprintf("%d userId not found", CODE_FAILED), nil
			}

			return fmt.Sprintf("%d server error", CODE_SERVER_ERROR), nil
		}

		err = coll.Update(bson.M{"_id": userId}, bson.M{"$set": bson.M{"pushId": pushId}})
		if err != nil {
			return fmt.Sprintf("%d server error", CODE_SERVER_ERROR), nil
		}

		offpush.UpdateUserPushId(userId, pushId)

		return fmt.Sprintf("%d success", CODE_SUCCESS), nil

	case CMD_SET_OFF_NOTIFY:
		params := strings.Split(this.Params, " ")
		userId := params[0]
		isAllowNotify := params[1]

		if userId == "" || isAllowNotify == "" {
			return fmt.Sprintf("%d param error", CODE_PARAM_ERROR), nil
		}

		coll := db.MgoSession().DB("pushd").C("user_info")
		err := coll.FindId(userId).One(nil)
		if err != nil {
			if err == mgo.ErrNotFound {
				return fmt.Sprintf("%d userId not found", CODE_FAILED), nil
			}

			return fmt.Sprintf("%d server error", CODE_SERVER_ERROR), nil
		}

		if isAllowNotify == "1" {
			err = coll.Update(bson.M{"_id": userId}, bson.M{"$set": bson.M{"isAllowNotify": true}})
			offpush.ValidUser(userId)
		} else {
			err = coll.Update(bson.M{"_id": userId}, bson.M{"$set": bson.M{"isAllowNotify": false}})
			offpush.InvalidUser(userId)
		}

		if err != nil {
			return fmt.Sprintf("%d server error", CODE_SERVER_ERROR), nil
		}

		return fmt.Sprintf("%d success", CODE_SUCCESS), nil

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

		oldChannelId := params[1]
		mainType, err0 :=  strconv.Atoi(params[0])
		if err0 != nil {
			return "", errors.New("errorparam wrong")
		}

		collection := db.MgoSession().DB("pushd").C("unstable_info")

		// check if channel has been applied
		var result interface{}
		err0 = collection.Find(bson.M{"channelId": oldChannelId}).Select(bson.M{"_id":0, "proposer":1}).One(&result)
		if err0 == nil {
			if result.(bson.M)["proposer"].(string) == this.Client.uuid {
				ret = "error10002"
			}else{
				ret = "error10001"
			}
			return
		}

		// create channel info in collection of unstable_info
		activeUser := []string{this.Client.uuid}
		objectId := bson.NewObjectId()

		err0 = collection.Insert(bson.M{"type": UNSTABLE_INFO_TYPE_FRAME_CHAT,
					"subtype": mainType,
					"proposer": this.Client.uuid,
					"channelId": oldChannelId,
					"time": time.Now().Unix(),
					"activeUser": activeUser,
					"_id": objectId})

		// this happens when another user apply on the channel at the same time
		if err0 != nil {
			ret = "error10001"
			return
		}

		// update userInfo
		uuids := storage.FetchUuidsAboutChannel(oldChannelId)
		var documents []interface{}
		for _, userId := range uuids {
			documents = append(documents, bson.M{"_id": userId})
			documents = append(documents, bson.M{"$push": bson.M{"frame_chat": objectId}})
		}

		bulk := db.MgoSession().DB("pushd").C("user_info").Bulk()
		bulk.Upsert(documents...)
		_, err0 = bulk.Run()
		if err0 != nil {
			collection.RemoveId(objectId)
			ret = "error500"
			return
		}

		Subscribe(this.Client, oldChannelId)
		for _, uuid := range uuids {
			tclient, exists := UuidToClient.GetClient(uuid)
			if exists {
				Subscribe(tclient, oldChannelId)
			}
		}

		newChannelId := objectId.Hex()
		// subscribe self to channel
		Subscribe(this.Client, newChannelId)

		// force notify all relevant online users
		notice := fmt.Sprintf("%s %s %d %s %s %s", OUTPUT_FRAME_CHAT, CMD_FRAME_APPLY, mainType, this.Client.uuid, newChannelId, oldChannelId)
		Publish2(oldChannelId, notice, this.Client.uuid, true)
		ret = newChannelId

	case CMD_FRAME_JOIN:
		if !bson.IsObjectIdHex(this.Params){
			ret = "error500"
			return
		}

		channelObjectId := bson.ObjectIdHex(this.Params)
		channelId := this.Params

		collection := db.MgoSession().DB("pushd").C("unstable_info")
		// why $push or not $addToSet
		// To prevent old conn from removing self from activeUser
		err0 := collection.UpdateId(channelObjectId, bson.M{"$push": bson.M{"activeUser": this.Client.uuid}})
		if err0 != nil {
			// cause is channel have been dismiss
			ret = "error10003"
			return
		}

		// join in this channel
		Subscribe(this.Client, channelId)

		// notify other users that I have join in
		notice := fmt.Sprintf("%s %s %d %s %s", OUTPUT_FRAME_CHAT, CMD_FRAME_JOIN, -1, this.Client.uuid, channelId)
		Publish2(channelId, notice, this.Client.uuid, false)
		ret = "success"

	case CMD_FRAME_ACCEPT:
		if !bson.IsObjectIdHex(this.Params){
			ret = "error500"
			return
		}

		channelObjectId := bson.ObjectIdHex(this.Params)
		channelId := this.Params

		collection := db.MgoSession().DB("pushd").C("unstable_info")
		err0 := collection.UpdateId(channelObjectId, bson.M{"$push": bson.M{"activeUser": this.Client.uuid}})
		if err0 != nil {
			// channel has been dismissed
			ret = "error10003"
			return
		}

		// join in this channel
		Subscribe(this.Client, channelId)

		// notify another user that I agree
		notice := fmt.Sprintf("%s %s %d %s %s", OUTPUT_FRAME_CHAT, CMD_FRAME_ACCEPT, -1, this.Client.uuid, channelId)
		Publish2(channelId, notice, this.Client.uuid, false)
		ret = "success"

	case CMD_FRAME_OUT:
		if !bson.IsObjectIdHex(this.Params){
			ret = "error500"
			return
		}

		channelObjectId := bson.ObjectIdHex(this.Params)
		channelId := this.Params

		collection := db.MgoSession().DB("pushd").C("unstable_info")
		change := mgo.Change{
			Update:bson.M{"$pull": bson.M{"activeUser": this.Client.uuid}},
			ReturnNew: true,
		}
		var result interface{}
		_, err0 := collection.Find(bson.M{"_id": channelObjectId}).Apply(change, &result)
		if err0 != nil {
			ret = "error500"
			return
		}

		if len(result.(bson.M)["activeUser"].([]interface{})) == 0 {
			// double check
			err0 = collection.Remove(bson.M{"_id": channelObjectId, "activeUser": []string{}})
			if err0 == nil {
				// clear relevant data about newChannelId in mongodb
				unstableInfo := result.(bson.M)
				realChannelId := unstableInfo["channelId"].(string)
				UUIDs := storage.FetchUuidsAboutChannel(realChannelId)
				var documents []interface{}
				for _, userId := range UUIDs {
					documents = append(documents, bson.M{"_id": userId})
					documents = append(documents, bson.M{"$pull": bson.M{"frame_chat": channelObjectId}})
				}

				bulk := db.MgoSession().DB("pushd").C("user_info").Bulk()
				bulk.Upsert(documents...)
				bulk.Run()

				notice := fmt.Sprintf("%s %s2 %d %s %s", OUTPUT_FRAME_CHAT, CMD_FRAME_DISMISS, -1, this.Client.uuid, channelId)
				Publish2(realChannelId, notice, this.Client.uuid, true)
				Unsubscribe(this.Client, channelId)
				ret = "success"
				return

			}
		}

		// quit out from this channel
		Unsubscribe(this.Client, channelId)

		// notify other users that I have quit
		notice := fmt.Sprintf("%s %s %d %s %s", OUTPUT_FRAME_CHAT, CMD_FRAME_OUT, -1, this.Client.uuid, channelId)
		Publish2(channelId, notice, this.Client.uuid, false)
		ret = "success"

	case CMD_FRAME_REFUSE:
		if !bson.IsObjectIdHex(this.Params){
			ret = "error500"
			return
		}
		notice := fmt.Sprintf("%s %s %d %s %s", OUTPUT_FRAME_CHAT, CMD_FRAME_REFUSE, -1, this.Client.uuid, this.Params)
		Publish2(this.Params, notice, this.Client.uuid, false)
		ret = "success"

	case CMD_FRAME_DISMISS:

		if !bson.IsObjectIdHex(this.Params){
			ret = "error500"
			return
		}

		channelObjectId := bson.ObjectIdHex(this.Params)
		channelId := this.Params

		collection := db.MgoSession().DB("pushd").C("unstable_info")


		change := mgo.Change{
			Update:bson.M{"$pull": bson.M{"activeUser": this.Client.uuid}},
			ReturnNew: true,
		}
		var result interface{}
		_, err0 := collection.Find(bson.M{"_id": channelObjectId}).Apply(change, &result)
		if err0 != nil {
			ret = "error500"
			return
		}

		if len(result.(bson.M)["activeUser"].([]interface{})) == 0 {
			// double check
			err0 = collection.Remove(bson.M{"_id": channelObjectId, "activeUser": []string{}})
			if err0 == nil {
				// clear relevant data about newChannelId in mongodb
				unstableInfo := result.(bson.M)
				realChannelId := unstableInfo["channelId"].(string)
				UUIDs := storage.FetchUuidsAboutChannel(realChannelId)
				var documents []interface{}
				for _, userId := range UUIDs {
					documents = append(documents, bson.M{"_id": userId})
					documents = append(documents, bson.M{"$pull": bson.M{"frame_chat": channelObjectId}})
				}

				bulk := db.MgoSession().DB("pushd").C("user_info").Bulk()
				bulk.Upsert(documents...)
				bulk.Run()

				notice := fmt.Sprintf("%s %s2 %d %s %s", OUTPUT_FRAME_CHAT, CMD_FRAME_DISMISS, -1, this.Client.uuid, channelId)
				Publish2(realChannelId, notice, this.Client.uuid, true)
				ret = "success"
				return
			}
		}

		// quit out from this channel
		Unsubscribe(this.Client, channelId)

		// notify other users that I have quit
		notice := fmt.Sprintf("%s %s %d %s %s", OUTPUT_FRAME_CHAT, CMD_FRAME_DISMISS, -1, this.Client.uuid, channelId)
		Publish2(channelId, notice, this.Client.uuid, false)
		ret = "success"

	case CMD_FRAME_INFO:

		if !bson.IsObjectIdHex(this.Params){
			ret = "error500"
			return
		}

		channelId := bson.ObjectIdHex(this.Params)
		recover()
		var result interface{}
		err0 := db.MgoSession().DB("pushd").C("unstable_info").FindId(channelId).One(&result)
		if err0 != nil {
			ret = "error10003"
			return
		}

		retBytes, _ := json.Marshal(result)
		ret = string(retBytes)

	case CMD_INVOKE_FRAME_ACTION:
		var result interface{}
		err0 := db.MgoSession().DB("pushd").C("user_info").FindId(this.Client.uuid).Select(bson.M{"frame_chat":1, "_id":0}).One(&result)
		if err0 != nil {
			return
		}

		frame_chat := result.(bson.M)["frame_chat"].([]interface{})
		if len(frame_chat) == 0 {
			return
		}

		for _, id := range frame_chat {
			channelId := id.(bson.ObjectId)
			err0 = db.MgoSession().DB("pushd").C("unstable_info").FindId(channelId).One(&result)
			if err0 != nil {
				continue
			}
			info := result.(bson.M)
			if info["type"].(int) == UNSTABLE_INFO_TYPE_FRAME_CHAT {
				mainType := info["subtype"].(int)
				proposerId := info["proposer"].(string)
				newChannelId := info["_id"].(bson.ObjectId).Hex()
				oldChannelId := info["channelId"].(string)
				notice := fmt.Sprintf("%s %s %d %s %s %s", OUTPUT_FRAME_CHAT, CMD_FRAME_APPLY, mainType, proposerId, newChannelId, oldChannelId)
				go this.Client.WriteFormatMsg(OUTPUT_RCIV, notice)
			}
		}


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
		err := coll.Insert(bson.M{"_id": this.Params, "channel_stat": bson.M{}, "frame_chat": []string{}})
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
		this.uuid = params[1]
		// 具体聊天环境初始化在另一个GO ROUTINE中来完成,目的是及早让客户端准备好
		go this.Client.initChatEnv(params[1])
		// TODO 新启动的GO routine 在本命令响应返回前完成，会引发一些问题（如：其他用户发来消息，概率非常小），该如何处理
		// TODO 考虑一种办法来close掉之前的client，目前做法是不做任何处理，超时后自动清理
		return "uuid saved", nil

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
