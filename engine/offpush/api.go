package offpush

import (
	"github.com/nicholaskh/pushd/config"
	"github.com/nicholaskh/pushd/db"
	"gopkg.in/mgo.v2/bson"
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
		coll := db.MgoSession().DB("pushd").C("user_info")

		userIds = make([]string, len(tempUserIds))
		// 将用户信息缓存在内存中
		for _, value := range tempUserIds {
			uId := value.(string)
			userIds = append(userIds, uId)
			_, exists = userInfoCollection.getUserInfo(uId)
			if !exists {
				// 从MONGODB中加载用户的信息
				err = coll.FindId(uId).
					Select(bson.M{"isAllowNotify":1, "pushId":1,"_id":0}).
					One(&result)

				if err != nil {
					continue
				}
				info, _ := result.(bson.M)

				tPushId, ok := info["pushId"]
				if !ok {
					// 如果没有设置pushId，默认用户Id为pushId
					tPushId = ""
				}
				pushId := tPushId.(string)
				tisAllowNotify, ok := info["isAllowNotify"]
				var isAllowNotify bool
				if !ok {
					isAllowNotify = true
				}else{
					isAllowNotify = tisAllowNotify.(bool)
				}
				userInfoCollection.addUserInfo(uId, pushId, false, isAllowNotify)

			}
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

	go senderPool.ObtainOneOffSender().send(pushIds, message, ownerId, channel)

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
