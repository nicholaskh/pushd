package offpush

import (
	"github.com/nicholaskh/pushd/config"
	"github.com/nicholaskh/pushd/db"
	"gopkg.in/mgo.v2/bson"
	log "github.com/nicholaskh/log4go"
	"fmt"
)

func InitOffPushService() error {
	err := initOffSenderPool(config.PushdConf.OffSenderPoolNum, config.PushdConf.MsgQueueAddrs)
	if err != nil {
		return err
	}
	initBasicData()
	return nil
}

func CheckAndPush(channel, message, ownerId string) {

	if senderPool == nil {
		panic("offpush模块未执行初始化")
	}

	userIds, exists := channelToUserIds.getUserIdsByChannel(channel)
	if !exists {
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
		channelToUserIds.addAllUserIds(channel, userIds...)

		userIds, _ = channelToUserIds.getUserIdsByChannel(channel)
	}
	// 筛选符合离线推送条件的用户，提取出相应的pushId
	pushIds := make([]string, 0, len(userIds))
	for _, uId := range userIds {
		if uId == ownerId {
			continue
		}

		pushId, isValid, exists := userInfoCollection.checkAndFetchPushId(uId)
		if !exists || !isValid {
			continue
		}
		pushIds = append(pushIds, pushId)
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

func UpdateOrAddUserInfo(userId string, pushId string, isOnline, isAllowNotify bool){
	userInfoCollection.updateOrAddUserInfo(userId, pushId, isOnline, isAllowNotify)
}

func UpdateUserPushId(userId, pushId string) {
	userInfo, exists := userInfoCollection.getUserInfo(userId)
	if exists {
		userInfo.pushId = pushId
		return
	}
}

func ChangeUserStatus(userId string, isOnline bool) {
	userInfo, exists := userInfoCollection.getUserInfo(userId)
	if exists {
		userInfo.isOnline = isOnline
	}
}

func ValidUser(userId string) {
	userInfo, exists := userInfoCollection.getUserInfo(userId)
	if exists {
		userInfo.isAllowNotify = true
	}
}

func InvalidUser(userId string) {
	userInfo, exists := userInfoCollection.getUserInfo(userId)
	if exists {
		userInfo.isAllowNotify = false
	}
}
