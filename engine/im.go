package engine

import (
	"fmt"
	"sort"
	"strings"
	"github.com/nicholaskh/pushd/db"
	"gopkg.in/mgo.v2/bson"
	"time"
)

func createRoom(userId, channelId string) (err error) {
	// 更新群聊与用户id映射表channel_uuids
	userIds := []string{userId}
	err = db.MgoSession().DB("pushd").C("channel_uuids").
		Insert(bson.M{"_id": channelId, "uuids": userIds})
	if err != nil {
		return err
	}

	// 更新用户信息表user_info
	ts := time.Now().UnixNano()
	channelKey := fmt.Sprintf("channel_stat.%s", channelId)
	_, err = db.MgoSession().DB("pushd").
			C("user_info").
			Upsert(bson.M{"_id": userId}, bson.M{"$set": bson.M{channelKey: ts}})
	if err != nil {
		return err
	}

	// 更新用户id与群聊信息映射表uuid_channels
	_, err = db.MgoSession().DB("pushd").
		C("uuid_channels").
		UpsertId(userId, bson.M{"$addToSet": bson.M{"channels": channelId}})
	return
}

func joinRoom(channelId string, userIds ...string) (err error) {
	// 更新群聊与用户id映射表channel_uuids
	err = db.MgoSession().DB("pushd").C("channel_uuids").
		Update(bson.M{"_id": channelId}, bson.M{"$addToSet": bson.M{"uuids": bson.M{"$each": userIds}}})
	if err != nil {
		return
	}

	// 更新用户信息表user_info
	ts := time.Now().UnixNano()
	var documents []interface{}
	channelKey := fmt.Sprintf("channel_stat.%s", channelId)
	for _, v := range userIds {
		documents = append(documents, bson.M{"_id": v})
		documents = append(documents, bson.M{"$set": bson.M{channelKey: ts}})
	}
	bulk := db.MgoSession().DB("pushd").C("user_info").Bulk()
	bulk.Update(documents...)
	_, err = bulk.Run()
	if err != nil {
		return err
	}

	// 更新用户id与群聊信息映射表uuid_channels
	var documents2 []interface{}
	for _, v := range userIds {
		documents2 = append(documents2, bson.M{"_id": v})
		documents2 = append(documents2, bson.M{"$addToSet": bson.M{"channels": channelId}})
	}
	bulk2 := db.MgoSession().DB("pushd").C("uuid_channels").Bulk()
	bulk2.Upsert(documents2...)
	_, err = bulk2.Run()
	if err != nil {
		return err
	}

	for _, userId := range userIds {
		client, exists := UuidToClient.GetClient(userId)
		if exists{
			Subscribe(client, channelId)
		}
	}
	return nil
}

func leaveRoom(channelId string, userIds ...string) (err error) {
	// 更新群聊与用户id映射表channel_uuids
	err = db.MgoSession().DB("pushd").C("channel_uuids").
		Update(bson.M{"_id": channelId}, bson.M{"$pullAll": bson.M{"uuids": userIds}})
	if err != nil {
		return
	}

	// 如果群聊中一个人也没有了，就将聊天室也删除
	var emptyUserIds []string
	db.MgoSession().DB("pushd").C("channel_uuids").Remove(bson.M{"_id":channelId, "uuids": emptyUserIds})

	// 更新用户信息表user_info
	bulk2 := db.MgoSession().DB("pushd").C("user_info").Bulk()
	var documents2 []interface{}
	channelKey := fmt.Sprintf("channel_stat.%s", channelId)
	for _, v := range userIds {
		documents2 = append(documents2, bson.M{"_id": v})
		documents2 = append(documents2, bson.M{"$unset": bson.M{channelKey: 1}})
	}
	bulk2.Update(documents2...)
	_, err = bulk2.Run()

	// 更新用户id与群聊信息映射表uuid_channels
	bulk := db.MgoSession().DB("pushd").C("uuid_channels").Bulk()
	var documents []interface{}
	for _, v := range userIds {
		documents = append(documents, bson.M{"_id": v})
		documents = append(documents, bson.M{"$pull": bson.M{"channels": channelId}})
	}
	bulk.Update(documents...)
	_, err = bulk.Run()

	// 更新内存中用户的订阅信息
	for _, uuid := range userIds {
		client, exists := UuidToClient.GetClient(uuid)
		if exists {
			Unsubscribe(client, channelId)
		}
	}

	return
}

func disolveRoom(channelId string) (err error) {
	var result interface{}
	err = db.MgoSession().DB("pushd").C("channel_uuids").
		Find(bson.M{"_id": channelId}).
		Select(bson.M{"uuids":1, "_id":0}).
		One(&result)
	if err != nil {
		return err
	}

	userIds := result.(bson.M)["uuids"].([]interface{})
	var userIds2 []string
	for _, uuid := range userIds {
		userIds2 = append(userIds2, uuid.(string))
	}
	err = leaveRoom(channelId, userIds2...)
	return
}

func roomid2Channelid(roomid string) string {
	return roomid
}

func generateRoomIdByUuidList(uuids ...string) string {
	sort.Strings(uuids)
	return fmt.Sprintf("pri_%s", strings.Join(uuids, "_"))
}
