package engine

import (
	"fmt"
	"github.com/nicholaskh/golib/set"
)

func createRoom(client *Client, roomid string) string {
	Subscribe(client, roomid2Channelid(roomid))
	return fmt.Sprintf("%s %s", OUT_CREATEROOM, roomid)
}

func joinRoom(uuid, roomid string) string {
	// check whether the client with this uuid is on this node
	client, exists := UuidToClient.GetClient(uuid)
	if exists{
		Subscribe(client, roomid2Channelid(roomid))
	} else {
		peers := set.NewSet()
		for _, peer := range Proxy.Router.Peers {
			peers.Add(peer)
		}
		Broadcast(peers, fmt.Sprintf("%d %s %s", BOARDCAST_JOIN, uuid, roomid))
	}
	return fmt.Sprintf("%s %s", OUT_JOINROOM, roomid)
}

func leaveRoom(client *Client, roomid string) string {
	Unsubscribe(client, roomid)
	return fmt.Sprintf("%s %s", OUT_LEAVEROOM, roomid2Channelid(roomid))
}

func roomid2Channelid(roomid string) string {
	return roomid
}
