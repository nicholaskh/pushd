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
	"gopkg.in/mgo.v2"
	"github.com/nicholaskh/pushd/engine/storage"
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
}

func NewClient() (this *Client) {
	this = new(Client)
	this.Channels = make(map[string]int)
	this.ackList = NewAckList()
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

	// clear revelant data in unstable_info
	var result interface{}
	err0 := db.MgoSession().DB("pushd").C("user_info").FindId(this.uuid).Select(bson.M{"frame_chat":1,"_id":0}).One(&result)
	if err0 == nil {
		frame_chat := result.(bson.M)["frame_chat"].([]interface{})
		if len(frame_chat) != 0 {
			collection := db.MgoSession().DB("pushd").C("unstable_info")
			for _, id := range frame_chat {
				objectId := id.(bson.ObjectId)
				channelId := objectId.Hex()
				_, exist := this.Channels[channelId]
				if exist {
					change := mgo.Change{
						Update:bson.M{"$pull": bson.M{"activeUser": this.uuid}},
						ReturnNew: true,
					}
					var result interface{}
					_, err0 := collection.Find(bson.M{"_id": objectId}).Apply(change, &result)
					if err0 != nil {
						continue
					}

					// check if anyone is in this channel chat
					err0 = collection.Remove(bson.M{"_id": objectId, "activeUser": []string{}})
					if err0 == nil {
						// clear relevant data about newChannelId in mongodb
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

					}

				}

			}
		}

	}

	UnsubscribeAllChannels(this)
	UuidToClient.Remove(this.uuid, this)
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
	var result interface{}

	UuidToClient.Remove(uuid, this)
	// store relationship between uuid and client to table of mapping
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
