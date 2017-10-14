package offpush

import (
	"sync"
	"github.com/nicholaskh/golib/set"
	"github.com/nicholaskh/golib/concurrent/map"
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
	cmap.ConcurrentMap
}

func newUserInfoCollection() (t *UserInfoCollection) {
	t = new(UserInfoCollection)
	t.ConcurrentMap = cmap.New()
	return
}

func (this *UserInfoCollection) getUserInfo(userId string) (*UserInfoEntity, bool) {

	entity, exists := this.Get(userId)
	if exists {
		return entity.(*UserInfoEntity), exists
	}
	return nil, exists
}

// pushId, 是否离线且允许推送， 用户信息是否存在
func (this *UserInfoCollection)checkAndFetchPushId(userId string) (string, bool, bool){
	entity, exists := this.getUserInfo(userId)
	if !exists {
		return "", true, false
	}
	if entity.isOnline && entity.isAllowNotify{
		// 如果pushId为空，默认他的userId为pushId
		if entity.pushId == ""{
			return userId, true, true
		}
		return entity.pushId, true, true
	}

	return "", false, true

}

func (this *UserInfoCollection) addUserInfo(userId, pushId string, isOnline, isAllowNotify bool) *UserInfoEntity {
	entity, exists := this.getUserInfo(userId)
	if exists {
		return entity
	}
	entity = new(UserInfoEntity)
	entity.pushId = pushId
	entity.isAllowNotify = isAllowNotify
	entity.isOnline = true
	this.Set(userId, entity)

	return entity
}

func (this *UserInfoCollection)updateOrAddUserInfo(userId, pushId string, isOnline, isAllowNotify bool){
	entity, exists := this.getUserInfo(userId)
	if !exists {
		entity = this.addUserInfo(userId, pushId, isOnline, isAllowNotify)
	}

	entity.isOnline = isOnline
	entity.isAllowNotify = isAllowNotify
	entity.pushId = pushId
}

func initBasicData(){
	channelToUserIds = newChannelToUserIds()
	userInfoCollection = newUserInfoCollection()
}
