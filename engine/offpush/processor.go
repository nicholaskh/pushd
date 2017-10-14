package offpush

import (
	"sync/atomic"
	"errors"
	"jpushclient"
	"fmt"
	"github.com/nicholaskh/pushd/db"
	"gopkg.in/mgo.v2/bson"
	"github.com/nicholaskh/pushd/config"
)


var (
	senderPool *OffSenderPool
)

type IOffSender interface {
	send(pushIds []string, message, ownerId, channelId string) error
}

type OffSenderPool struct {
	pool []IOffSender
	baton int32
	size int32
}

func (this *OffSenderPool)ObtainOneOffSender() IOffSender {
	var now int32
	var hitLoc int32
	for ;; {
		now = atomic.LoadInt32(&this.baton)
		now = senderPool.baton

		hitLoc = (now+1) % this.size
		if atomic.CompareAndSwapInt32(&this.baton, now, hitLoc) {
			break
		}
	}
	return this.pool[hitLoc]
}

type Sender struct {
	pushClient *jpushclient.PushClient
}

func newSender(address string) (*Sender, error) {

	sender := new(Sender)
	// TODO 检测是否是http长连接
	sender.pushClient = jpushclient.NewPushClient(config.PushdConf.JpushSecretKey, config.PushdConf.JpushAppKey)
	return sender, nil

}

func (this *Sender) send(pushIds []string, message, ownerId , channelId string) error {

	if len(pushIds) == 0 {
		return nil
	}
	pf := jpushclient.Platform{}
	pf.Add(jpushclient.ANDROID)

	ad := jpushclient.Audience{}
	ad.SetAlias(pushIds)

	if !bson.IsObjectIdHex(channelId) {
		return errors.New("channelId is invalid ObjectIdHex")
	}
	var result interface{}

	err := db.ChatRoomCollection().
		FindId(bson.ObjectIdHex(channelId)).
		Select(bson.M{"type":1, "_id":0, "name":1}).
		One(&result)

	if err != nil {
		return err
	}
	chatRoomInfo := result.(bson.M)
	temp, exists := chatRoomInfo["type"]
	if !exists {
		return errors.New("collection chatroom has no field of type")
	}

	// TODO 为何是float64而不能用int32
	mType, ok := temp.(float64)
	if !ok {
		return errors.New("field type in collection chatroom is not a int")
	}
	if !bson.IsObjectIdHex(ownerId){
		return errors.New("ownerId is invalid ObjectIdHex")
	}
	err = db.UserCollection().
		FindId(bson.ObjectIdHex(ownerId)).
		Select(bson.M{"userName":1,"_id":0}).
		One(&result)

	if err != nil {
		return err
	}
	userName, exists := result.(bson.M)["userName"]
	if !exists {
		return errors.New("collection user has no field of userName")
	}

	var notice jpushclient.Notice
	androidNotice := jpushclient.AndroidNotice{}
	notice.SetAndroidNotice(&androidNotice)
	if mType == 1 {
		androidNotice.Title = userName.(string)
		androidNotice.Alert = message
	}else{
		temp, exists = chatRoomInfo["name"]
		if !exists {
			return errors.New("collection chatroom has no field of name")
		}
		roomName, ok := temp.(string)
		if !ok {
			return errors.New("field name in collection chatroom is not a string")
		}

		androidNotice.Title = roomName
		androidNotice.Alert = fmt.Sprintf("%s:%s", userName, message)
	}
	payload := jpushclient.NewPushPayLoad()
	payload.SetPlatform(&pf)
	payload.SetAudience(&ad)
	payload.SetNotice(&notice)

	bytes, _ := payload.ToBytes()
	_, err = this.pushClient.Send(bytes)

	// TODO 处理pushId不合法的情况
	return err
}

func (this *Sender) close(){
	// TODO 实现
}

func initOffSenderPool(capacity int, addrs []string) error {
	if len(addrs) < 1 {
		return errors.New("地址为空")
	}

	senderPool = new(OffSenderPool)
	senderPool.size = int32(capacity)
	senderPool.baton = 0
	senderPool.pool = make([]IOffSender, capacity)

	length := len(addrs)
	for i:= 0; i<capacity; i++ {
		address := addrs[i % length]
		t, err := newSender(address)
		if err != nil {
			// 关掉所有建立好的连接
			for _, temp := range senderPool.pool {
				if temp != nil {
					sender, ok := interface{}(temp).(Sender)
					if ok {
						sender.close()
					}
				}
			}
			return err
		}
		senderPool.pool[i] = t
	}

	return nil

}
