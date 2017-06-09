package engine

import (
	"fmt"
	"sort"
	"strings"
	"github.com/nicholaskh/pushd/engine/storage"
	"github.com/nicholaskh/pushd/db"
	"gopkg.in/mgo.v2/bson"
)

func createRoom(uuid, channelId, channelName string) string {
	uuids := []string{uuid}
	storage.EnqueueChanUuids(channelName, channelId, false, uuids)

	return fmt.Sprintf("%s %s", OUTPUT_CREATEROOM, channelId)
}

func joinRoom(channelId, uuid string) string {
	client, exists := UuidToClient.GetClient(uuid)
	if exists{
		Subscribe(client, channelId)
	}
	return fmt.Sprintf("%s %s", OUTPUT_JOINROOM, channelId)
}

func leaveRoom(channelId string, uuids ...string) string {
	for _, uuid := range uuids {
		client, exists := UuidToClient.GetClient(uuid)
		if exists {
			Unsubscribe(client, channelId)
		}
	}

	storage.EnqueueChanUuids("", channelId, true, uuids)

	return fmt.Sprintf("%s %s", OUTPUT_LEAVEROOM, channelId)
}

func disolveRoom(channelId string) {
	var result interface{}
	err := db.MgoSession().DB("pushd").C("channel_uuids").
		Find(bson.M{"_id": channelId}).
		Select(bson.M{"uuids":1, "_id":0}).
		One(&result)
	if err == nil {
		uuids := result.(bson.M)["uuids"].([]interface{})
		var uuids2 []string
		for _, uuid := range uuids {
			uuids2 = append(uuids2, uuid.(string))
		}
		leaveRoom(channelId, uuids2...)
	}
}

func roomid2Channelid(roomid string) string {
	return roomid
}

func generateRoomIdByUuidList(uuids ...string) string {
	sort.Strings(uuids)
	return fmt.Sprintf("pri_%s", strings.Join(uuids, "_"))
}
