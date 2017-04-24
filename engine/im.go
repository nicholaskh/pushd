package engine

import (
	"fmt"
	"sort"
	"strings"
	"github.com/nicholaskh/pushd/engine/storage"
)

func createRoom(client *Client, roomid string) string {
	channel := roomid2Channelid(roomid)
	Subscribe(client, roomid2Channelid(roomid))

	uuids := []string{client.uuid}
	storage.EnqueueChanUuids(channel, false, uuids)

	return fmt.Sprintf("%s %s", OUTPUT_CREATEROOM, roomid)
}

func joinRoom(roomid, uuid string) string {
	client, exists := UuidToClient.GetClient(uuid)
	if exists{
		Subscribe(client, roomid2Channelid(roomid))
	}
	return fmt.Sprintf("%s %s", OUTPUT_JOINROOM, roomid)
}

func leaveRoom(client *Client, roomid string) string {
	Unsubscribe(client, roomid)

	channel := roomid2Channelid(roomid)
	uuids := []string{client.uuid}
	storage.EnqueueChanUuids(channel, true, uuids)

	return fmt.Sprintf("%s %s", OUTPUT_LEAVEROOM, roomid2Channelid(roomid))
}

func roomid2Channelid(roomid string) string {
	return roomid
}

func generateRoomIdByUuidList(uuids ...string) string {
	sort.Strings(uuids)
	return fmt.Sprintf("pri_%s", strings.Join(uuids, "_"))
}
