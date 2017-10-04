package offpush

import (
	"sync"
	"github.com/nicholaskh/golib/set"
	log "github.com/nicholaskh/log4go"
	"fmt"
)

var (
	channelToUserIds *ChannelToUserIds
	userInfoCollection *UserInfoCollection
)

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
	rwMutex sync.RWMutex
	m map[string]*UserIdCollection
}

func newChannelToUserIds() *ChannelToUserIds{
	temp := new(ChannelToUserIds)
	temp.rwMutex = sync.RWMutex{}
	temp.m = make(map[string]*UserIdCollection)
	return temp
}

// 线程安全添加一个key和空的value
func (this *ChannelToUserIds) addNewKeyEmptyValue(channel string) (collection *UserIdCollection) {
	this.rwMutex.Lock()
	defer this.rwMutex.Unlock()
	collection, exists := this.m[channel]
	if exists {
		return
	}
	collection = newUserIdCollection()
	this.m[channel] = collection
	return
}

func (this *ChannelToUserIds) getValue(channel string) (*UserIdCollection, bool) {
	this.rwMutex.RLock()
	defer this.rwMutex.RUnlock()
	t, exists := this.m[channel]
	return t, exists
}

func (this *ChannelToUserIds)addUserId(channel, userId string){
	collection, exists := this.getValue(channel)
	if !exists {
		collection = this.addNewKeyEmptyValue(channel)
	}
	collection.rwMutex.Lock()
	collection.userIds.Add(userId)
	collection.rwMutex.Unlock()
}

func (this *ChannelToUserIds)addAllUserIds(channel string, freshUserIds ...string) {
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

func (this *ChannelToUserIds) removeUserId(channel, userId string) {
	collection, exists := this.getValue(channel)
	if !exists {
		return
	}
	collection.rwMutex.Lock()
	collection.userIds.Remove(userId)
	collection.rwMutex.Unlock()
}

func (this *ChannelToUserIds) getUserIdsByChannel(channel string) ([]string, bool){
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

// 存储用户信息
type UserInfoEntity struct {
	pushId string // 设备id
	isOnline bool // 用户是否在线
	isAllowNotify bool // 用户是否允许接收离线通知
}

// 存储用户Id到用户信息的映射
type UserInfoCollection struct{
	m map[string]*UserInfoEntity
	// TODO 锁机制优化
	mutex sync.RWMutex
}

func newUserInfoCollection() (t *UserInfoCollection) {
	t = new(UserInfoCollection)
	t.m = make(map[string]*UserInfoEntity)
	t.mutex = sync.RWMutex{}
	return
}

func (this *UserInfoCollection) getUserInfo(userId string) (*UserInfoEntity, bool) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	entity, exists := this.m[userId]
	return entity, exists
}

// pushId, 是否离线且允许推送， 用户信息是否存在
func (this *UserInfoCollection)checkAndFetchPushId(userId string) (string, bool, bool){

	entity, exists := this.getUserInfo(userId)
	if !exists {
		return "", true, false
	}
	if !entity.isOnline && entity.isAllowNotify && entity.pushId != ""{
		return entity.pushId, true, true
	}

	return "", false, true

}

func (this *UserInfoCollection) addUserInfo(userId, pushId string, isOnline, isAllowNotify bool) *UserInfoEntity {

	// 先加锁后再判断是否存在，防止后面的添加覆盖前面的
	this.mutex.Lock()
	defer this.mutex.Unlock()

	entity, exists := this.m[userId]
	if exists {
		return entity
	}
	entity = new(UserInfoEntity)
	entity.pushId = pushId
	entity.isAllowNotify = isAllowNotify
	this.m[userId] = entity

	return entity
}

func (this *UserInfoCollection)updateOrAddUserInfo(userId, pushId string, isOnline, isAllowNotify bool){

	entity, exists := this.m[userId]
	if !exists {
		entity = this.addUserInfo(userId, pushId, isOnline, isAllowNotify)
	}

	log.Info(fmt.Sprint(userId, " ", pushId, " ", isOnline, " ", isAllowNotify))
	entity.isOnline = isOnline
	entity.isAllowNotify = isAllowNotify

}

func initBasicData(){
	channelToUserIds = newChannelToUserIds()
	userInfoCollection = newUserInfoCollection()
}
