package engine

import (
	"github.com/nicholaskh/golib/server"
	log "github.com/nicholaskh/log4go"
	"container/list"
	"github.com/nicholaskh/pushd/db"
	"gopkg.in/mgo.v2/bson"
	"sync"
	"fmt"
	"github.com/nicholaskh/pushd/config"
)

const (
	TYPE_CLIENT = 1
	TYPE_SERVER = 2
)

type Client struct {
	Channels map[string]int
	Type     uint8
	*server.Client
	uuid string
	ackList *AckList
	tokenInfo TokenInfo
	msgIdCache MsgIdCache
}

func NewClient() (this *Client) {
	this = new(Client)
	this.Channels = make(map[string]int)
	this.ackList = NewAckList()
	this.msgIdCache = NewMsgIdCashe()
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

	this.Mutex.Lock()
	cli, exist := UuidToClient.GetClient(this.uuid)
	if !exist || cli != this{
		return
	}
	UnsubscribeAllChannels(this)
	UuidToClient.Remove(this.uuid)
	this.Mutex.Unlock()

	this.Client.Close()
}


func (this *Client) PushMsg(op, msg , channelId string, msgId, ts int64) {
	err := this.WriteFormatMsg(op, msg)

	if err == nil {
		this.ackList.push(channelId, ts, msgId)
	}
}

func (this *Client) AckMsg(msgId int64, channelId string) {
	this.ackList.listLock.Lock()
	defer this.ackList.listLock.Unlock()

	log.Info(fmt.Sprintf("log ack:channel:%s msgid:%d", channelId, msgId))
	for e := this.ackList.Front(); e != nil; e = e.Next() {
		element := e.Value.(*AckListElement)
		if element.msgId != msgId || element.channelId != channelId{
			continue
		}

		this.ackList.List.Remove(e)
		channelKey := fmt.Sprintf("channel_stat.%s", channelId)
		db.MgoSession().DB("pushd").
			C("user_info").
			Update(
			bson.M{"_id": this.uuid, channelKey: bson.M{"$lt": element.ts}},
			bson.M{"$set": bson.M{channelKey: element.ts}})
	}
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

	this.uuid = uuid

	// clear old client connection
	oldClient, exi := UuidToClient.GetClient(uuid)
	if exi {
		if this != oldClient{
			oldClient.Close()
		}
	}

	// cache last msgid and push all offline messsage that are relevant to the user
	var result interface{}
	coll := db.MgoSession().DB("pushd").C("user_info")
	coll.FindId(uuid).Select(bson.M{"_id":0, "channel_stat":1}).One(&result)
	if result != nil {
		// cache last msgid
		userInfo := result.(bson.M)
		channelStat := userInfo["channel_stat"].(bson.M)
		maxTs := int64(0)
		hitChannle := ""
		for channel, va := range channelStat {
			ts := va.(int64)
			if ts > maxTs {
				maxTs = ts
				hitChannle = channel
			}
		}
		if maxTs > 0 {
			db.MgoSession().DB("pushd").C("msg_log").Find(bson.M{"channel": hitChannle, "ts": maxTs}).
				Select(bson.M{"_id":0,"msgid":1}).One(&result)
			if result != nil {
				if result.(bson.M)["msgid"] != nil {
					this.msgIdCache.CheckAndSet(result.(bson.M)["msgid"].(int64))
				}
			}
		}
	}

	// store relate of uuid and client to table of mapping
	UuidToClient.AddClient(uuid, this)

	// check and force client to subscribe related and active channels
	err := db.MgoSession().DB("pushd").C("uuid_channels").
		Find(bson.M{"_id": uuid}).
		Select(bson.M{"_id": 0, "channels": 1}).
		One(&result)

	if err == nil {
		channels := result.(bson.M)["channels"].([]interface{})
		for _, value := range channels {
			channel := value.(string)
			// check native
			_, exists := PubsubChannels.Get(channel)
			if !exists {
				// check other nodes
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

type AckListElement struct {
	msgId int64
	channelId string
	ts int64
}

type AckList struct {
	*list.List
	listLock sync.Mutex
}

func NewAckList() (acklist *AckList) {
	acklist = new(AckList)
	acklist.List = list.New()
	return
}

func (this *AckList)push(channelId string, ts, msgId int64) {
	ele := new(AckListElement)
	ele.msgId = msgId
	ele.channelId = channelId
	ele.ts = ts
	this.listLock.Lock()
	defer this.listLock.Unlock()
	this.PushBack(ele)
}

type TokenInfo struct {
	token string
	expire int64
}

type MsgIdCache struct {
	*list.List
}

func NewMsgIdCashe() MsgIdCache {
	caches := MsgIdCache{list.New()}
	for i:=int64(0);i<10;i++ {
		caches.PushFront(i)
	}
	return caches
}

func (this MsgIdCache) CheckAndSet(msgId int64) bool {
	for e := this.Front(); e != nil; e = e.Next() {
		if e.Value == nil {
			continue
		} else if e.Value.(int64) == msgId {
			return true
		}
	}

	this.PushFront(msgId)
	this.Remove(this.Back())
	return false

}
