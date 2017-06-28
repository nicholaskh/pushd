package engine

import (
	"github.com/nicholaskh/golib/server"
	log "github.com/nicholaskh/log4go"
	"container/list"
	"github.com/nicholaskh/pushd/db"
	"gopkg.in/mgo.v2/bson"
	"sync"
	"fmt"
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

	this.Mutex.Lock()
	defer this.Mutex.Unlock()

	if !this.IsConnected(){
		return
	}
	UnsubscribeAllChannels(this)
	if this.uuid != "" {
		UuidToClient.Remove(this.uuid)
		uuidTokenMap.rmTokenInfo(this.uuid)
	}

	this.Client.Close()
}


func (this *Client) PushMsg(op, msg , channelId string, msgId, ts int64) {
	err := this.WriteFormatMsg(op, msg)

	if err == nil {
		ele := new(AckListElement)
		ele.msgId = msgId
		ele.channelId = channelId
		ele.ts = ts
		this.ackList.listLock.Lock()
		defer this.ackList.listLock.Unlock()
		this.ackList.PushBack(ele)
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

