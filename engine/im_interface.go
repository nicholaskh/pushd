package engine

import (
	"errors"
	"fmt"
	"github.com/nicholaskh/golib/set"
)

const (
	CMD_CREATEROOM  = "create_room"
	CMD_JOINROOM = "join_room"
	CMD_LEAVEROOM = "leave_room"

	OUT_CREATEROOM = "CREATEROOM"
	OUT_JOINROOM  = "JOINROOM"
	OUT_LEAVEROOM = "LEAVEROOM"
)

func ProcessImCmd(cmd *Cmdline)(ret string, err error){
	switch cmd.Cmd {
	case CMD_CREATEROOM:
		if len(cmd.Params) < 1 || cmd.Params[0] == "" {
			return "", errors.New("Lack roomid")
		}
		ret = createRoom(cmd.Client, cmd.Params[0])

	case CMD_JOINROOM:
		if len(cmd.Params) < 2{
			return "", errors.New("Lack params")
		}
		if cmd.Params[0] == ""{
			return "", errors.New("uuid is empty")
		}
		if cmd.Params[1] == ""{
			return "", errors.New("roomid is empty")
		}

		ret = joinRoom(cmd.Params[0], cmd.Params[1])

	case CMD_LEAVEROOM:
		if len(cmd.Params) < 1 || cmd.Params[0] == "" {
			return "", errors.New("Lack roomid")
		}
		ret = leaveRoom(cmd.Client, cmd.Params[0])
	}
	return
}

func createRoom(client *Client, roomid string) string {
	Subscribe(client, roomid)
	return fmt.Sprintf("%s %s", OUT_CREATEROOM, roomid)
}

func joinRoom(uuid, roomid string) string {
	// check whether the client with this uuid is on this node
	client, exists := UuidToClient.GetClient(uuid)
	if exists{
		Subscribe(client, roomid)
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
	return fmt.Sprintf("%s %s", OUT_LEAVEROOM, roomid)
}
