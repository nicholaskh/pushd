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
	"github.com/nicholaskh/pushd/config"
	"github.com/nicholaskh/pushd/engine/mail"
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
	CMD_VIDO_CHAT	= "vido"
	CMD_ACK_MSG	= "ack"
	CMD_DISOLVE	= "disolve"
	CMD_UNSUBS 	= "unsubs"
	CMD_SUBS	= "subs"
	CMD_ADD_USER_INTO_ROOM = "add_into_room"
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
	CMD_PUBLISH_NOTIFICATION_IN_CHAT_ROOM = "pnicr"
	CMD_PUSH_NOTIFY_TO_USERS = "pntus"
	CMD_ACK_SERVER_NOTIFY = "asn"
	CMD_PEEK_SERVER_NOTIFY = "psn"


	OUTPUT_FRAME_CHAT	= "FRAMECHAT"
	OUTPUT_TOKEN 	           = "TOKEN"
	OUTPUT_AUTH_SERVER        = "AUTHSERVER"
	OUTPUT_RCIV	            = "RCIV"
	OUTPUT_APPKEY             = "APPKEY"
	OUTPUT_SUBSCRIBED         = "SUBSCRIBED"
	OUTPUT_ALREADY_SUBSCRIBED = "ALREADY SUBSCRIBED"
	OUTPUT_NOT_SUBSCRIBED     = "NOT SUBSCRIBED"
	OUTPUT_UNSUBSCRIBED       = "UNSUBSCRIBED"
	OUTPUT_PONG               = "pong"
)

const (
	UNSTABLE_INFO_TYPE_FRAME_CHAT = 1
)

// TODO 思考定义了这些常量，为什么没有用上
const (
 TYPE_SINGLE_VOICE  = 1
 TYPE_MUL_VOICE  = 2
 TYPE_SINGLE_VIDEO  = 3
 TYPE_MUL_VIDEO  = 4
)

// TODO 所有响应消息，修改为使用code码来区别类型
const (
	CODE_SUCCESS = 200
	CODE_PARAM_ERROR = 202
	CODE_SERVER_ERROR = 500
	CODE_FAILED = 400
	CODE_TOKEN_OR_TERM_ERROR = 508
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
			return fmt.Sprintf("%d param number is lacked", CODE_PARAM_ERROR), nil
		}

		isResend := params[0]
		channel := params[1]
		tempMsgId := params[2]
		msg := params[3]

		msgId, err := strconv.ParseInt(tempMsgId, 10, 64)
		if err != nil {
			return fmt.Sprintf("%d param error, msgId cannot be parse", CODE_PARAM_ERROR), nil
		}

		// check if this message has been sent
		if isResend == "Y" {
			isHit := db.MgoSession().DB("pushd").C("msg_log").
					Find(bson.M{"channel": channel,
						"uuid": this.Client.uuid,
						"msgid": msgId}).One(nil)
			if isHit == nil {
				return fmt.Sprintf("%d %d %d", CODE_SUCCESS, msgId, time.Now().UnixNano()), nil
			}
		}

		//_, exists := this.Client.Channels[channel]
		//if !exists {
		//	Subscribe(this.Client, channel)
		//
		//	// force other related online clients to join in this channel
		//	var result interface{}
		//	err := db.MgoSession().DB("pushd").C("channel_uuids").
		//		Find(bson.M{"_id": channel}).
		//		Select(bson.M{"uuids":1, "_id":0}).
		//		One(&result)
		//
		//	if err == nil {
		//		uuids := result.(bson.M)["uuids"].([]interface{})
		//		for _, uuid := range uuids {
		//			tclient, exists := UuidToClient.GetClient(uuid.(string))
		//			if exists {
		//				Subscribe(tclient, channel)
		//			}
		//		}
		//	}
		//
		//}

		// CheckAndPush(channel, msg, this.Client.uuid)

		Publish(channel, msg, this.Client.uuid, msgId, false)

		return fmt.Sprintf("%d %d %d", CODE_SUCCESS, msgId, time.Now().UnixNano()), nil

	case CMD_VIDO_CHAT:
		index := 0
		for i, value := range this.Params2 {
			if value == ' ' {
				index = i
				break
			}
		}
		if index == 0 {
			return fmt.Sprintf("%d param number is lacked", CODE_PARAM_ERROR), nil
		}
		channelId := string(this.Params2[: index])

		_, exists := this.Client.Channels[channelId]
		if !exists {
			Subscribe(this.Client, channelId)
		}

		// 构造用于最终推送给其他client的消息体
		ownerIdByte := []byte(this.Client.uuid)
		buff := bytes.NewBuffer(make([]byte, 0, len(ownerIdByte) + 1 + len(this.Params2)))
		buff.Write(ownerIdByte)
		buff.WriteByte(' ')
		buff.Write(this.Params2)
		msgPush := buff.Bytes()

		PublishBinMsg(channelId, this.Client.uuid, msgPush, true)

		return fmt.Sprintf("%d success", CODE_SUCCESS), nil

	case CMD_ACK_MSG:
		params := strings.SplitN(this.Params, " ", 4)
		if len(params) != 4 {
			return fmt.Sprintf("%d param number is lacked", CODE_PARAM_ERROR), nil
		}

		channelId := params[0]
		msgId, err := strconv.ParseInt(params[1], 10, 64)
		if err != nil {
			return fmt.Sprintf("%d param error, msgId cannot be parse", CODE_PARAM_ERROR), nil
		}
		ts, err := strconv.ParseInt(params[2], 10, 64)
		if err != nil {
			return fmt.Sprintf("%d param error, ts cannot be parse", CODE_PARAM_ERROR), nil
		}
		ownerId := params[3]

		this.Client.AckMsg(msgId, ts, ownerId, channelId)
		return fmt.Sprintf("%d success", CODE_SUCCESS), nil

	case CMD_SUBSCRIBE:
		//		if !this.Client.IsClient() {
		//			return "", ErrNotPermit
		//		}
		if this.Params == "" {
			return fmt.Sprintf("%d param number is lacked", CODE_PARAM_ERROR), nil
		}
		Subscribe(this.Client, this.Params)
		return fmt.Sprintf("%d success", CODE_SUCCESS), nil

	case CMD_PUBLISH:
		//		if !this.Client.IsClient() && !this.Client.IsServer() {
		//			return "", ErrNotPermit
		//		}
		params := strings.SplitN(this.Params, " ", 3)
		if len(params) < 3 || params[2] == "" {
			return fmt.Sprintf("%d param number is lacked", CODE_PARAM_ERROR), nil
		} else {
			msgId, err := strconv.ParseInt(params[1], 10, 64)
			if err != nil {
				return fmt.Sprintf("%d param error, msgId cannot be parse", CODE_PARAM_ERROR), nil
			}

			Publish(params[0], params[2], this.Client.uuid, msgId, false)
			return fmt.Sprintf("%d success", CODE_SUCCESS), nil
		}

	case CMD_UNSUBSCRIBE:
		//		if !this.Client.IsClient() {
		//			return "", ErrNotPermit
		//		}
		params := strings.SplitN(this.Params, " ", 2)
		if len(params) < 1 || params[0] == "" {
			return fmt.Sprintf("%d param number is lacked", CODE_PARAM_ERROR), nil
		}
		Unsubscribe(this.Client, params[0])
		return fmt.Sprintf("%d success", CODE_SUCCESS), nil

	case CMD_RETRACT_MESSAGE:
		if this.Params == "" {
			return fmt.Sprintf("%d param number is lacked", CODE_PARAM_ERROR), nil
		}
		params := strings.SplitN(this.Params, " ", 3)
		if len(params) < 3 {
			return fmt.Sprintf("%d param number is lacked", CODE_PARAM_ERROR), nil
		}

		channel := params[0]
		msgId, err := strconv.ParseInt(params[1], 10, 64)
		if err != nil {
			return fmt.Sprintf("%d msgId can not parse", CODE_PARAM_ERROR), nil
		}

		newMsgId, err := strconv.ParseInt(params[2], 10, 64)
		if err != nil {
			return fmt.Sprintf("%d param error", CODE_PARAM_ERROR), nil
		}

		col := db.MgoSession().DB("pushd").C("msg_log")
		err = col.Remove(bson.M{"channel": channel, "uuid": this.uuid, "msgid": msgId})

		if err != nil {
			if realError, ok := err.(interface{}).(*mgo.LastError); ok {
				return fmt.Sprintf("%d %s",CODE_SERVER_ERROR, realError.Error()), nil
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
			return fmt.Sprintf("%d param number is lacked", CODE_PARAM_ERROR), nil
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

		client, exists := UuidToClient.GetClient(userId)
		if exists {
			client.PushId = pushId
		}

		return fmt.Sprintf("%d success", CODE_SUCCESS), nil

	case CMD_SET_OFF_NOTIFY:
		params := strings.Split(this.Params, " ")
		userId := params[0]
		isAllowNotify := params[1]

		if userId == "" || isAllowNotify == "" {
			return fmt.Sprintf("%d param number is lacked", CODE_PARAM_ERROR), nil
		}

		client, exists := UuidToClient.GetClient(userId)
		if exists {
			if isAllowNotify == "1" {
				client.IsAllowForceNotify = true
			} else {
				client.IsAllowForceNotify = false
			}
		}

		return fmt.Sprintf("%d success", CODE_SUCCESS), nil

	case CMD_CREATEROOM:
		params := strings.Split(this.Params, " ")
		if len(params) < 1 {
			return fmt.Sprintf("%d param error", CODE_PARAM_ERROR), nil
		}

		roomId := generateRoomIdByUuidList(append(params, this.Client.uuid)...)
		channelId := roomid2Channelid(roomId)
		err = createRoom(this.Client.uuid, channelId)
		if err != nil {
			return fmt.Sprintf("%d %s", CODE_FAILED, err.Error()), nil
		}

		for _, uuid := range params {
			err = joinRoom(channelId, uuid)
			return fmt.Sprintf("%d %s", CODE_FAILED, err.Error()), nil
		}

		return fmt.Sprintf("%d success", CODE_SUCCESS), nil

	case CMD_JOINROOM:
		params := strings.Split(this.Params, " ")
		if len(params) < 1 || params[0] == "" {
			return fmt.Sprintf("%d param error", CODE_PARAM_ERROR), nil
		}

		channelId := roomid2Channelid(params[0])
		err = joinRoom(channelId, this.Client.uuid)
		if err != nil {
			return fmt.Sprintf("%d %s", CODE_FAILED, err.Error()), nil
		}

		return fmt.Sprintf("%d success", CODE_SUCCESS), nil

	case CMD_LEAVEROOM:
		params := strings.Split(this.Params, " ")
		if len(params) < 1 || params[0] == "" {
			return fmt.Sprintf("%d param error", CODE_PARAM_ERROR), nil
		}

		err = leaveRoom(this.Client.uuid, roomid2Channelid(params[0]))
		if err != nil {
			return fmt.Sprintf("%d %s", CODE_FAILED, err.Error()), nil
		}

		return fmt.Sprintf("%d success", CODE_SUCCESS), nil

	case CMD_PEEK_SERVER_NOTIFY:
		letters, err := mail.GetLettersByType(this.uuid, MESSAGE_FOR_PERSON)
		if err != nil {
			return "", err
		}

		data, err := json.Marshal(letters)
		if err != nil {
			return "", err
		}

		return string(data), nil

	case CMD_ACK_SERVER_NOTIFY:
		notifyIds := strings.Split(this.Params, " ")
		err := mail.RemoveLetter(this.uuid, notifyIds)
		if err != nil {
			return fmt.Sprintf("%d %s", CODE_PARAM_ERROR, err.Error()), nil
		}

		return fmt.Sprintf("%d success", CODE_SUCCESS), nil

	case CMD_PUSH_NOTIFY_TO_USERS:
		var temp1 interface{}
		err := json.Unmarshal([]byte(this.Params), &temp1)
		if err!= nil {
			return fmt.Sprintf("%d param error1", CODE_PARAM_ERROR), nil
		}

		temp2, ok := temp1.(map[string]interface{})
		if !ok {
			return fmt.Sprintf("%d param error2", CODE_PARAM_ERROR), nil
		}

		temp4, exists := temp2["type"]
		if !exists {
			return fmt.Sprintf("%d param error3", CODE_PARAM_ERROR), nil
		}

		temp41, err := strconv.ParseInt(temp4.(string), 10, 32)
		if err != nil {
			return fmt.Sprintf("%d param error4", CODE_PARAM_ERROR), nil
		}

		mtype := int(temp41)

		// 提取userIds
		temp5, exists := temp2["userIds"]
		if !exists {
			return fmt.Sprintf("%d param error5", CODE_PARAM_ERROR), nil
		}
		temp6 := temp5.([]interface{})
		userIds := make([]string, 0, len(temp6))
		for _, ele := range temp6 {
			userIds = append(userIds, ele.(string))
		}

		message := temp2["message"].(string)

		ts := time.Now().UnixNano()
		msgId := bson.NewObjectId().Hex()
		clientMsg := fmt.Sprintf("%d %d %s %d %s",MESSAGE_FOR_PERSON, mtype, msgId, ts, message)

		// store message to mail of users
		letter := mail.Letter{}
		letter.Id = msgId
		letter.Ts = ts
		letter.Type = mtype
		letter.Data = message

		err = mail.SendLetterToMail(userIds, &letter)
		if err != nil {
			log.Error(fmt.Sprintf("sendLetterToMailError: %s, %s", clientMsg, err.Error()))
		}
		PushToClients(clientMsg, true, userIds...)

		log.Info(fmt.Sprintf("push to users: %s", clientMsg));

		return fmt.Sprintf("%d success", CODE_SUCCESS), nil

	case CMD_PUBLISH_NOTIFICATION_IN_CHAT_ROOM:
		//params格式 [type chatRoomId message]
		params := strings.SplitN(this.Params, " ", 3)
		if len(params) < 3 {
			return fmt.Sprintf("%d param number is lacked", CODE_PARAM_ERROR), nil
		}

		mtype , err := strconv.Atoi(params[0])
		if err != nil {
			return fmt.Sprintf("%d type can not parse to int32", CODE_PARAM_ERROR), nil
		}

		chatRoomId := params[1]
		message := params[2]

		notifyInChatRoom := func(chatRoomId, message string, mtype int) (string, error){
			//err = db.MgoSession().DB("pushd").C("channel_uuids").
			//	Find(bson.M{"_id": chatRoomId}).
			//	Select(bson.M{"uuids":1, "_id":0}).
			//	One(nil)
			//
			//if err != nil {
			//	return fmt.Sprintf("%d chatRoomId is not exists", CODE_PARAM_ERROR), err
			//}
			//
			//var result interface{}
			//err := db.MgoSession().DB("pushd").C("channel_uuids").
			//	Find(bson.M{"_id": chatRoomId}).
			//	Select(bson.M{"uuids":1, "_id":0}).
			//	One(&result)
			//
			//if err == nil {
			//	userIds := result.(bson.M)["uuids"].([]interface{})
			//	for _, userId := range userIds {
			//		tclient, exists := UuidToClient.GetClient(userId.(string))
			//		if exists {
			//			Subscribe(tclient, chatRoomId)
			//		}
			//	}
			//}

			ts := time.Now().UnixNano()

			if config.PushdConf.EnableStorage() {
				db.MgoSession().
					DB("pushd").
					C("msg_log").
					Insert(
					bson.M{"channel": chatRoomId,
						"msg": message,
						"ts": ts,
						"type": mtype})
			}

			clientMsg := fmt.Sprintf("%d %s %d %s",mtype, chatRoomId, ts, message)
			log.Info(fmt.Sprintf("chatRoom:%s notify:%s", chatRoomId, clientMsg));
			PublishStrMsg(chatRoomId, clientMsg, "", true)

			return "", nil
		}

		/**
		 record
		 群聊msg_log不能删除，因为我们的离线消息是基于msg_log做的
		  */
		switch mtype {

		// 某人被踢出群聊
		case SOME_ONE_EXPELLED_FROM_CHAT_ROOM:
			res, err := notifyInChatRoom(chatRoomId, message, mtype)
			if err != nil {
				return res, nil
			}
			// TODO 通知被踢出的人
		default:
			res, err := notifyInChatRoom(chatRoomId, message, mtype)
			if err != nil {
				return res, nil
			}
		}

		return fmt.Sprintf("%d success", CODE_SUCCESS), nil

	case CMD_FRAME_APPLY:
		params := strings.Split(this.Params, " ")
		if len(params) != 2 {
			return fmt.Sprintf("%d param number is lacked", CODE_PARAM_ERROR), nil
		}

		oldChannelId := params[1]
		mainType, err0 :=  strconv.Atoi(params[0])
		if err0 != nil {
			return fmt.Sprintf("%d param number is lacked", CODE_PARAM_ERROR), nil
		}

		collection := db.MgoSession().DB("pushd").C("unstable_info")

		// check if channel has been applied
		var result interface{}
		err0 = collection.Find(bson.M{"channelId": oldChannelId}).Select(bson.M{"_id":0, "proposer":1}).One(&result)
		if err0 == nil {
			if result.(bson.M)["proposer"].(string) == this.Client.uuid {
				return fmt.Sprintf("%d 10002", CODE_FAILED), nil
			}else{
				return fmt.Sprintf("%d 10001", CODE_FAILED), nil
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
			return fmt.Sprintf("%d 10001", CODE_FAILED), nil
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
			return fmt.Sprintf("%d 500", CODE_FAILED), nil
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
		PublishStrMsg(oldChannelId, notice, this.Client.uuid, true)
		ret = newChannelId
		return fmt.Sprintf("%d %s", CODE_SUCCESS, newChannelId), nil


	case CMD_FRAME_JOIN:
		if !bson.IsObjectIdHex(this.Params){
			return fmt.Sprintf("%d 500", CODE_FAILED), nil
		}

		channelObjectId := bson.ObjectIdHex(this.Params)
		channelId := this.Params

		collection := db.MgoSession().DB("pushd").C("unstable_info")
		// why $push or not $addToSet
		// To prevent old conn from removing self from activeUser
		err0 := collection.UpdateId(channelObjectId, bson.M{"$push": bson.M{"activeUser": this.Client.uuid}})
		if err0 != nil {
			// cause is channel have been dismiss
			return fmt.Sprintf("%d 10003", CODE_FAILED), nil
		}

		// join in this channel
		Subscribe(this.Client, channelId)

		// notify other users that I have join in
		notice := fmt.Sprintf("%s %s %d %s %s", OUTPUT_FRAME_CHAT, CMD_FRAME_JOIN, -1, this.Client.uuid, channelId)
		PublishStrMsg(channelId, notice, this.Client.uuid, true)
		return fmt.Sprintf("%d success", CODE_SUCCESS), nil

	case CMD_FRAME_ACCEPT:
		if !bson.IsObjectIdHex(this.Params){
			return fmt.Sprintf("%d 500", CODE_FAILED), nil
		}

		channelObjectId := bson.ObjectIdHex(this.Params)
		channelId := this.Params

		collection := db.MgoSession().DB("pushd").C("unstable_info")
		err0 := collection.UpdateId(channelObjectId, bson.M{"$push": bson.M{"activeUser": this.Client.uuid}})
		if err0 != nil {
			// channel has been dismissed
			return fmt.Sprintf("%d 10003", CODE_FAILED), nil
		}

		// join in this channel
		Subscribe(this.Client, channelId)

		// notify another user that I agree
		notice := fmt.Sprintf("%s %s %d %s %s", OUTPUT_FRAME_CHAT, CMD_FRAME_ACCEPT, -1, this.Client.uuid, channelId)
		PublishStrMsg(channelId, notice, this.Client.uuid, true)
		return fmt.Sprintf("%d success", CODE_SUCCESS), nil

	case CMD_FRAME_OUT:
		if !bson.IsObjectIdHex(this.Params){
			return fmt.Sprintf("%d 500", CODE_FAILED), nil
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
			return fmt.Sprintf("%d 500", CODE_FAILED), nil
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
				PublishStrMsg(realChannelId, notice, this.Client.uuid, true)
				Unsubscribe(this.Client, channelId)
				return fmt.Sprintf("%d success", CODE_SUCCESS), nil

			}
		}

		// quit out from this channel
		Unsubscribe(this.Client, channelId)

		// notify other users that I have quit
		notice := fmt.Sprintf("%s %s %d %s %s", OUTPUT_FRAME_CHAT, CMD_FRAME_OUT, -1, this.Client.uuid, channelId)
		PublishStrMsg(channelId, notice, this.Client.uuid, true)
		return fmt.Sprintf("%d success", CODE_SUCCESS), nil

	case CMD_FRAME_REFUSE:
		if !bson.IsObjectIdHex(this.Params){
			return fmt.Sprintf("%d 500", CODE_FAILED), nil
		}
		notice := fmt.Sprintf("%s %s %d %s %s", OUTPUT_FRAME_CHAT, CMD_FRAME_REFUSE, -1, this.Client.uuid, this.Params)
		PublishStrMsg(this.Params, notice, this.Client.uuid, true)
		return fmt.Sprintf("%d success", CODE_SUCCESS), nil

	case CMD_FRAME_DISMISS:

		if !bson.IsObjectIdHex(this.Params){
			return fmt.Sprintf("%d 500", CODE_FAILED), nil
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
			return fmt.Sprintf("%d 500", CODE_FAILED), nil
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
				PublishStrMsg(realChannelId, notice, this.Client.uuid, true)
				return fmt.Sprintf("%d success", CODE_SUCCESS), nil
			}
		}

		// quit out from this channel
		Unsubscribe(this.Client, channelId)

		// notify other users that I have quit
		notice := fmt.Sprintf("%s %s %d %s %s", OUTPUT_FRAME_CHAT, CMD_FRAME_DISMISS, -1, this.Client.uuid, channelId)
		PublishStrMsg(channelId, notice, this.Client.uuid, true)
		return fmt.Sprintf("%d success", CODE_SUCCESS), nil

	case CMD_FRAME_INFO:

		if !bson.IsObjectIdHex(this.Params){
			return fmt.Sprintf("%d 500", CODE_FAILED), nil
		}

		channelId := bson.ObjectIdHex(this.Params)
		recover()
		var result interface{}
		err0 := db.MgoSession().DB("pushd").C("unstable_info").FindId(channelId).One(&result)
		if err0 != nil {
			return fmt.Sprintf("%d 10003", CODE_FAILED), nil
		}

		retBytes, _ := json.Marshal(result)
		return fmt.Sprintf("%d %s", CODE_SUCCESS, string(retBytes)), nil

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

	case CMD_ADD_USER_INTO_ROOM:
		params := strings.Split(this.Params, " ")
		if len(params) < 2 {
			return fmt.Sprintf("%d param wrong", CODE_PARAM_ERROR), nil
		}

		channelId := params[0]
		err = db.MgoSession().DB("pushd").C("channel_uuids").FindId(channelId).One(nil)
		if err != nil {
			if err == mgo.ErrNotFound {
				return fmt.Sprintf("%d channelId is not exists", CODE_FAILED), nil
			}
			return fmt.Sprintf("%d %s", CODE_SERVER_ERROR, err.Error()), nil
		}

		for _, tempUserId := range params[1:]{
			err := db.MgoSession().DB("pushd").C("user_info").FindId(tempUserId).One(nil)
			if err != nil {
				if err == mgo.ErrNotFound {
					return fmt.Sprintf("%d userId:%s is not exists", CODE_FAILED, tempUserId), nil
				}

				return fmt.Sprintf("%d %s", CODE_SERVER_ERROR, err.Error()), nil
			}
		}

		for _, tempUserId := range params[1:] {
			err = joinRoom(channelId, tempUserId)
			if err != nil {
				return fmt.Sprintf("%d %s", CODE_FAILED, err.Error()), nil
			}
		}


		if channelToUserIds.Exists(channelId) {
			channelToUserIds.AddAllUserIds(channelId, params[1:]...)
		} else {
			// 在执行Exists过程中，可能别的Goroutine后来执行了AddAllUserIds（此时放进内存中的可能是旧数据）
			// 所以为了保险起见，强制刷新
			LoadUserIdsFromDatabase(channelId)
		}

		return fmt.Sprintf("%d success", CODE_SUCCESS), nil

    //subs: subscribe from server
	case CMD_SUBS:
		params := strings.Split(this.Params, " ")
		if len(params) < 3 {
			return fmt.Sprintf("%d param wrong", CODE_PARAM_ERROR), nil
		}

		ownerId := params[2]
		channelId := params[1]
		// channelName := params[0]

		for _, tempUserId := range params[2:]{
			err := db.MgoSession().DB("pushd").C("user_info").FindId(tempUserId).One(nil)
			if err != nil {
				if err == mgo.ErrNotFound {
					return fmt.Sprintf("%d userId:%s is not exists", CODE_FAILED, tempUserId), nil
				}

				return fmt.Sprintf("%d server error", CODE_SERVER_ERROR), nil
			}
		}

		err = db.MgoSession().DB("pushd").C("channel_uuids").FindId(channelId).One(nil)
		if err == nil {
			return fmt.Sprintf("%d channelId has been exists", CODE_FAILED), nil
		}

		err = createRoom(ownerId, channelId)
		if err != nil {
			return fmt.Sprintf("%d %s", CODE_FAILED, err.Error()), nil
		}

		for _, uuid := range params[3:] {
			err = joinRoom(channelId, uuid)
			if err != nil {
				return fmt.Sprintf("%d %s", CODE_FAILED, err.Error()), nil
			}
		}

		return fmt.Sprintf("%d success", CODE_SUCCESS), nil

	case CMD_UNSUBS:
		params := strings.Split(this.Params, " ")
		if len(params) < 2 {
			return fmt.Sprintf("%d param wrong", CODE_PARAM_ERROR), nil
		}
		err = leaveRoom(params[0], params[1:]...)
		if err != nil {
			return fmt.Sprintf("%d %s", CODE_FAILED, err.Error()), nil
		}

		if channelToUserIds.Exists(params[0]) {
			for _, userId := range params[1:] {
				channelToUserIds.RemoveUserId(params[0], userId)
			}
		} else {
			// TODO 如果此时已经有其他Goroutine拿到了旧数据，正要加载到channelToUserIds，如何处理？
			// 概率非常小
		}

		return fmt.Sprintf("%d success", CODE_SUCCESS), nil

	case CMD_DISOLVE:
		if this.Params == "" {
			return fmt.Sprintf("%d param wrong", CODE_PARAM_ERROR), nil
		}
		channelId := this.Params
		err = disolveRoom(channelId)
		if err != nil {
			return fmt.Sprintf("%d %s", CODE_FAILED, err.Error()), nil
		}

		channelToUserIds.Del(channelId)

		return fmt.Sprintf("%d success", CODE_SUCCESS), nil

	case CMD_CREATE_USER:
		if this.Params == "" {
			return fmt.Sprintf("%d param number is lacked", CODE_PARAM_ERROR), nil
		}
		coll := db.MgoSession().DB("pushd").C("user_info")

		userId := this.Params
		err := coll.FindId(userId).One(nil)
		if err == nil {
			return fmt.Sprintf("%d userId has been exists", CODE_FAILED), nil
		}

		err = coll.Insert(bson.M{"_id": userId, "channel_stat": bson.M{}, "frame_chat": []string{}})
		if err != nil {
			return fmt.Sprintf("%d %s", CODE_FAILED, err.Error()), nil
		}

		return fmt.Sprintf("%d success", CODE_SUCCESS), nil

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
			var hisRet []interface{}
			err = db.MgoSession().DB("pushd").C("msg_log").Find(
				bson.M{"ts": bson.M{"$gt": ts},"channel": channel,"uuid":bson.M{"$ne":this.uuid}}).
				Select(bson.M{"_id": 0}).All(&hisRet)

			if len(hisRet) > 0 {
				data[channel] = hisRet
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
			return fmt.Sprintf("%d param number is lacked", CODE_PARAM_ERROR), nil
		}
		ts, err := strconv.ParseInt(params[1], 10, 64)
		if err != nil {
			return fmt.Sprintf("%d time cannot be parse", CODE_PARAM_ERROR), nil
		}
		channel := params[0]
		hisRet, err := fullHistory(channel, ts)
		if err != nil {
			log.Error(err)
		}

		var retBytes []byte
		retBytes, err = json.Marshal(hisRet)

		return fmt.Sprintf("%d %s", CODE_SUCCESS, string(retBytes)), nil

		//TODO auth_server修改为code码形式
	case CMD_AUTH_SERVER:
		if this.Params == "" {
			return "", errors.New("Invalid Params for auth_server")
		}
		ret, err = authServer(this.Params)
		if err == nil {
			this.Client.SetServer()
			ret = fmt.Sprintf("%s %s", OUTPUT_AUTH_SERVER, ret)
		}
		return ret, err

	//TODO CMD_TOKEN修改为code码形式
	case CMD_TOKEN:
		//if !this.Client.IsServer() {
		//	return "", ErrNotPermit
		//}
		token := getClientToken()
		if token == "" {
			return "", errors.New("gettoken error")
		}

		ret = fmt.Sprintf("%s %s", OUTPUT_TOKEN, token)

	//TODO CMD_APPKEY修改为code码形式
	case CMD_APPKEY:
		ret = fmt.Sprintf("%s %s", OUTPUT_APPKEY, getAppKey())

	case CMD_AUTH_CLIENT:
		params := strings.Split(this.Params, " ")
		if len(params) < 1 {
			return fmt.Sprintf("%d param number is lacked", CODE_PARAM_ERROR), nil
		}

		ret, err = authClient(params[0])
		if err == nil {
			this.Client.SetClient()
			// set token
			this.Client.initToken(params[0], time.Now().UnixNano())
			return fmt.Sprintf("%d %s", CODE_SUCCESS, ret), nil
		}
		return fmt.Sprintf("%d %s", CODE_FAILED, err.Error()), nil

	case CMD_SETUUID:
		params := strings.Split(this.Params, " ")
		if len(params) < 2 || params[1] == "" {
			return fmt.Sprintf("%d param number is lacked", CODE_PARAM_ERROR), nil
		}
		this.uuid = params[1]
		// 具体聊天环境初始化在另一个GO ROUTINE中来完成,目的是及早让客户端准备好
		go this.Client.initChatEnv(params[1])
		// TODO 新启动的GO routine 在本命令响应返回前完成，会引发一些问题（如：其他用户发来消息，概率非常小），该如何处理
		// TODO 考虑一种办法来close掉之前的client，目前做法是不做任何处理，超时后自动清理
		return fmt.Sprintf("%d success", CODE_SUCCESS), nil

	case CMD_PING:
		return OUTPUT_PONG, nil

	default:
		return fmt.Sprintf("%d Cmd not found: %s\n", CODE_FAILED, this.Cmd), nil
	}

	return
}

func trimCmdline(str string) string {
	return strings.TrimRight(str, string([]rune{0, 13, 10}))
}
