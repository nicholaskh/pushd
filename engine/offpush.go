package engine

import (
	"sync/atomic"
	"errors"
	"jpushclient"
	"fmt"
	"github.com/nicholaskh/pushd/db"
	"gopkg.in/mgo.v2/bson"
	"github.com/nicholaskh/pushd/config"
	log "github.com/nicholaskh/log4go"
	"sync"
	"github.com/nicholaskh/golib/set"

	"github.com/nicholaskh/golib/cache"
)


var (
	senderPool *OffSenderPool
	channelToUserIds *ChannelToUserIds
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

	mType, ok := temp.(int)
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

// 存储某个群聊中所有的用户的Id
type UserIdCollection struct {
	rwMutex sync.RWMutex
	userIds set.Set
}

func newUserIdCollection() *UserIdCollection {
	temp := new(UserIdCollection)
	temp.rwMutex = sync.RWMutex{}
	temp.userIds = set.NewSet()
	return temp
}

// 存储群聊Id到群聊中用户集合的映射
type ChannelToUserIds struct {
	*cache.LruCache
}

func newChannelToUserIds(maxChannelCache int) *ChannelToUserIds{
	temp := new(ChannelToUserIds)
	temp.LruCache = cache.NewLruCache(maxChannelCache)
	return temp
}

// 线程安全添加一个key和空的value
func (this *ChannelToUserIds) addNewKeyEmptyValue(channel string) (*UserIdCollection) {
	value := this.LruCache.GetOrAdd(channel, newUserIdCollection())
	return value.(*UserIdCollection)
}

func (this *ChannelToUserIds) getValue(channel string) (*UserIdCollection, bool) {
	t, exists := this.LruCache.Get(channel)
	if exists {
		return t.(*UserIdCollection), true
	}
	return nil, exists
}

func (this *ChannelToUserIds)AddUserId(channel, userId string){
	collection, exists := this.getValue(channel)
	if !exists {
		collection = this.addNewKeyEmptyValue(channel)
	}
	collection.rwMutex.Lock()
	collection.userIds.Add(userId)
	collection.rwMutex.Unlock()
}

func (this *ChannelToUserIds)AddAllUserIds(channel string, freshUserIds ...string) {
	collection, exists := this.getValue(channel)
	if !exists {
		collection = this.addNewKeyEmptyValue(channel)
	}

	collection.rwMutex.Lock()
	for _, userId := range freshUserIds {
		collection.userIds.Add(userId)
	}
	collection.rwMutex.Unlock()
	return
}

func (this *ChannelToUserIds) RemoveUserId(channel, userId string) {
	collection, exists := this.getValue(channel)
	if !exists {
		return
	}
	collection.rwMutex.Lock()
	collection.userIds.Remove(userId)
	collection.rwMutex.Unlock()
}

func (this *ChannelToUserIds) GetUserIdsByChannel(channel string) ([]string, bool){
	collection, exists := this.getValue(channel)
	if !exists {
		return nil, false
	}

	userIds := collection.userIds
	items := make([]string, len(userIds))

	// TODO 每次都要复制是不是性能太差
	// TODO 补充：不一定是复制，引用地址？
	collection.rwMutex.RLock()
	for key := range userIds {
		items = append(items, key.(string))
	}
	collection.rwMutex.RUnlock()

	return items, true
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


func InitOffPushService() error {
	channelToUserIds = newChannelToUserIds(config.PushdConf.ChannelInfoCacheNum)
	err := initOffSenderPool(config.PushdConf.OffSenderPoolNum, config.PushdConf.MsgQueueAddrs)
	if err != nil {
		return err
	}
	return nil
}

func CheckAndPush(channel, message, ownerId string) {

	if senderPool == nil {
		panic("offpush模块未执行初始化")
	}

	userIds, exists := channelToUserIds.GetUserIdsByChannel(channel)
	if !exists {
		log.Info(fmt.Sprintf("load %s.........", channel))
		var result interface{}
		err := db.MgoSession().DB("pushd").C("channel_uuids").
			Find(bson.M{"_id":channel}).
			Select(bson.M{"uuids":1,"_id":0}).
			One(&result)

		if err != nil {
			return
		}

		t, exists := result.(bson.M)["uuids"]
		if !exists {
			return
		}

		tempUserIds := t.([]interface{})

		userIds = make([]string, 0, len(tempUserIds))
		// 将用户信息缓存在内存中
		for _, value := range tempUserIds {
			uId := value.(string)
			userIds = append(userIds, uId)
		}
		// 缓存群中所有用户id
		channelToUserIds.AddAllUserIds(channel, userIds...)
	}

	if  len(userIds) < 1 {
		return
	}

	// 筛选符合离线推送条件的用户，提取出相应的pushId
	pushIds := make([]string, 0, len(userIds)-1)
	for _, uId := range userIds {
		if uId == ownerId {
			continue
		}
		client, exists := UuidToClient.GetClient(uId)
		if exists && client.IsAllowForceNotify{
			pushIds = append(pushIds, client.PushId)
		}
	}
	if len(pushIds) == 0 {
		return
	}

	go func() {
		err := senderPool.ObtainOneOffSender().send(pushIds, message, ownerId, channel)
		if err != nil {
			log.Info(fmt.Sprint("offline push error: %s", err.Error()))
		}
	}()

}