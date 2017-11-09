package engine

import (
	"github.com/nicholaskh/golib/server"
	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/db"
	"gopkg.in/mgo.v2/bson"
	"fmt"
	"github.com/nicholaskh/pushd/config"
	"gopkg.in/mgo.v2"
	"github.com/nicholaskh/pushd/engine/storage"
	"sync"
)

const (
	TYPE_CLIENT = 1
	TYPE_SERVER = 2

	CLIENT_STATE_CREATE = 0
	CLIENT_STATE_INIT_DOEN = 1
	CLIENT_STATE_DEAD = 2

)

type Client struct {
	Channels           map[string]int
	Type               uint8
	*server.Client
	uuid               string
	tokenInfo          TokenInfo
	PushId             string
	IsAllowForceNotify bool
	clientLock *sync.Mutex
	state int
}

func NewClient() (this *Client) {
	this = new(Client)
	this.Channels = make(map[string]int)
	this.clientLock = &sync.Mutex{}
	this.state = CLIENT_STATE_CREATE
	return
}

func (this *Client) SetClient() {
	this.Type |= TYPE_CLIENT
}

func (this *Client) SetServer() {
	this.Type |= TYPE_SERVER
}

func (this *Client) IsClient() bool {
	return (this.Type & TYPE_CLIENT) != 0
}

func (this *Client) IsServer() bool {
	return (this.Type & TYPE_SERVER) != 0
}

func (this *Client) ClearIdentity() {
	this.Type &= 0
}

func (this *Client) Close() {
	log.Debug("client channels: %s", this.Channels)
	log.Debug("pubsub channels: %s", PubsubChannels)

	this.clientLock.Lock()
	defer this.clientLock.Unlock()

	this.state = CLIENT_STATE_DEAD
	if this.IsClient() {
		this.clearFrameChat()
		UnsubscribeAllChannels(this)
		UuidToClient.Remove(this.uuid, this)
	}
	this.Client.Close()

}


// 清理和语音电话、视频聊天有关的信息
func (this *Client) clearFrameChat(){
	var result interface{}
	err0 := db.MgoSession().DB("pushd").C("user_info").FindId(this.uuid).Select(bson.M{"frame_chat":1,"_id":0}).One(&result)
	if err0 != nil {
		return
	}
	frame_chat := result.(bson.M)["frame_chat"].([]interface{})
	if len(frame_chat) == 0 {
		return
	}
	collection := db.MgoSession().DB("pushd").C("unstable_info")
	for _, id := range frame_chat {
		objectId := id.(bson.ObjectId)
		channelId := objectId.Hex()
		change := mgo.Change{
			Update:bson.M{"$pull": bson.M{"activeUser": this.uuid}},
			ReturnNew: true,
		}
		var result interface{}
		_, err0 := collection.Find(bson.M{"_id": objectId}).Apply(change, &result)
		if err0 != nil {
			continue
		}

		// 通过删除的方式来检测是否临时聊天已经结束
		err0 = collection.Remove(bson.M{"_id": objectId, "activeUser": []string{}})
		if err0 != nil {
			continue
		}

		// 如果结束清楚相关用户与此临时群聊相关的数据信息
		unstableInfo := result.(bson.M)
		realChannelId := unstableInfo["channelId"].(string)
		UUIDs := storage.FetchUuidsAboutChannel(realChannelId)
		var documents []interface{}
		for _, userId := range UUIDs {
			documents = append(documents, bson.M{"_id": userId})
			documents = append(documents, bson.M{"$pull": bson.M{"frame_chat": objectId}})
		}

		bulk := db.MgoSession().DB("pushd").C("user_info").Bulk()
		bulk.Upsert(documents...)
		bulk.Run()

		notice := fmt.Sprintf("%s %s2 %d %s %s", OUTPUT_FRAME_CHAT, CMD_FRAME_DISMISS, -1, this.uuid, channelId)
		PublishStrMsg(realChannelId, notice, this.uuid, true)
	}

}

func (this *Client) AckMsg(msgId, ts int64, ownerId, channelId string) {
	log.Info(fmt.Sprintf("log ack:channel:%s msgid:%d ownerId: %s", channelId, msgId, ownerId))
	channelKey := fmt.Sprintf("channel_stat.%s", channelId)
	db.MgoSession().DB("pushd").
		C("user_info").
		Update(
		bson.M{"_id": this.uuid, channelKey: bson.M{"$lt": ts}},
		bson.M{"$set": bson.M{channelKey: ts}})
}

func (this *Client) initToken(token string, expire int64) {
	this.tokenInfo.token = token
	this.tokenInfo.expire = expire

	//TODO is it neccesary to synchronize to database
	db.MgoSession().DB("pushd").C("client_token").
		Update(bson.M{"tk": token}, bson.M{"$set": bson.M{"uuid": this.uuid, "expire": expire}})
}

func (this *Client) updateTokenExpire(expire int64) {
	this.tokenInfo.expire = expire
	//TODO is it neccesary to synchronize to database
	db.MgoSession().DB("pushd").C("client_token").UpdateId(this.uuid, bson.M{"expire": expire})

}

func (this *Client) initChatEnv(uuid string) {
	this.clientLock.Lock()
	this.clientLock.Unlock()

	if this.state != CLIENT_STATE_CREATE{
		return
	}

	this.loadUserInfo()
	this.updateUserIdToClientMappingTable()
	this.subAllAssociateChannels()

	this.state = CLIENT_STATE_INIT_DOEN
}

// 更新用户全局状态信息缓存表
func (this *Client) loadUserInfo() error {
	// 获取用户基本信息
	var result interface{}
	coll := db.MgoSession().DB("pushd").C("user_info")
	err := coll.FindId(this.uuid).Select(bson.M{"pushId":1,"_id":0}).One(&result)
	if err != nil {
		return err
	}

	info, _ := result.(bson.M)
	tPushId, ok := info["pushId"]
	if !ok {
		this.PushId = this.uuid
	}else {
		this.PushId = tPushId.(string)
	}
	this.IsAllowForceNotify = false
	return nil
}

// 更新用户Id到Client映射表
func (this *Client) updateUserIdToClientMappingTable(){
	// 清除之前还未来得及关闭的client
	UuidToClient.Remove(this.uuid, this)
	// 注册自己到映射表
	UuidToClient.AddClient(this.uuid, this)
}

// 订阅所有和自己相关的处于活跃状态的群聊
func (this *Client) subAllAssociateChannels(){
	// 获取自己所在的所有群聊Id
	var result interface{}
	err := db.MgoSession().DB("pushd").C("uuid_channels").
		Find(bson.M{"_id": this.uuid}).
		Select(bson.M{"_id": 0, "channels": 1}).
		One(&result)

	if err == nil {
		// 订阅所有和自己相关的处于活跃状态的群聊
		channels := result.(bson.M)["channels"].([]interface{})
		for _, value := range channels {
			channel := value.(string)
			_, exists := PubsubChannels.Get(channel)
			if !exists {
				if config.PushdConf.IsDistMode() {
					_, exists = Proxy.Router.LookupPeersByChannel(channel)
				}
			}

			if exists {
				Subscribe(this, channel)
			}
		}
	}
}

type TokenInfo struct {
	token string
	expire int64
}
