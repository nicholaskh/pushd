package engine

import (
	"fmt"
	"time"

	"github.com/nicholaskh/golib/cache"
	cmap "github.com/nicholaskh/golib/concurrent/map"
	"github.com/nicholaskh/golib/set"
	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/config"
	//"github.com/nicholaskh/pushd/engine/storage"
	"bytes"
	"github.com/nicholaskh/pushd/engine/storage"
)

var (
	PubsubChannels *PubsubChans
	UuidToClient *UuidClientMap
)

type PubsubChans struct {
	*cache.LruCache
}

func NewPubsubChannels(maxChannelItems int) (this *PubsubChans) {
	this = new(PubsubChans)
	this.LruCache = cache.NewLruCache(maxChannelItems)
	return
}

func (this *PubsubChans) Get(channel string) (clients cmap.ConcurrentMap, exists bool) {
	clientsInterface, exists := PubsubChannels.LruCache.Get(channel)
	clients, _ = clientsInterface.(cmap.ConcurrentMap)
	return
}

type UuidClientMap struct {
	uuidToClient cmap.ConcurrentMap
}

func NewUuidClientMap() (this *UuidClientMap) {
	this = new(UuidClientMap)
	this.uuidToClient = cmap.New()
	return
}

func (this *UuidClientMap) AddClient(uuid string, client *Client) {
	this.uuidToClient.Set(uuid, client)
}

func (this *UuidClientMap) GetClient(uuid string) (client *Client, exists bool) {
	temp, exists := this.uuidToClient.Get(uuid)
	if exists {
		client = temp.(*Client)
	}
	return
}

func (this *UuidClientMap) Remove(uuid string, client *Client) {
	cli, exists := this.GetClient(uuid)
	if !exists {
		return
	}
	cli.Mutex.Lock()
	defer cli.Mutex.Unlock()

	cli, _ = this.GetClient(uuid)
	if cli != client {
		return
	}
	this.uuidToClient.Remove(uuid)
}

func Subscribe(cli *Client, channel string) string {
	//log.Debug("%x", channel)
	//_, exists := cli.Channels[channel]
	//if exists {
	//	return fmt.Sprintf("%s %s", OUTPUT_ALREADY_SUBSCRIBED, channel)
	//} else {
	//	cli.Channels[channel] = 1
	//	clients, exists := PubsubChannels.Get(channel)
	//	if exists {
	//		clients.Set(cli.RemoteAddr().String(), cli)
	//	} else {
	//		temp := PubsubChannels.GetOrAdd(channel, cmap.New())
	//		clients = temp.(cmap.ConcurrentMap)
	//		clients.Set(cli.RemoteAddr().String(), cli)
	//
	//		// subscribe只在本节点进行
	//		//// TODO 两个GOROUTINE同时subscribe，可能导致两次广播
	//		//forwardToAllOtherServer(S2S_SUB_CMD, []byte(channel))
	//	}
	//
	//	return fmt.Sprintf("%s %s", OUTPUT_SUBSCRIBED, channel)
	//}
	return ""

}

func Unsubscribe(cli *Client, channel string) string {
	//_, exists := cli.Channels[channel]
	//if exists {
	//	delete(cli.Channels, channel)
	//	clients, exists := PubsubChannels.Get(channel)
	//	if exists {
	//		clients.Remove(cli.RemoteAddr().String())
	//	}
	//	clients, exists = PubsubChannels.Get(channel)
	//
	//	if clients.Count() == 0 {
	//		PubsubChannels.Del(channel)
	//		forwardToAllOtherServer(S2S_UNSUB_CMD, []byte(channel))
	//	}
	//
	//	return fmt.Sprintf("%s %s", OUTPUT_UNSUBSCRIBED, channel)
	//} else {
	//	return fmt.Sprintf("%s %s", OUTPUT_NOT_SUBSCRIBED, channel)
	//}
	return ""
}

func UnsubscribeAllChannels(cli *Client) {
	//for channel, _ := range cli.Channels {
	//	clients, _ := PubsubChannels.Get(channel)
	//	clients.Remove(cli.RemoteAddr().String())
	//	if clients.Count() == 0 {
	//		PubsubChannels.Del(channel)
	//		forwardToAllOtherServer(S2S_UNSUB_CMD, []byte(channel))
	//	}
	//}
	//cli.Channels = nil
}

func Publish(channel, msg , userId string, msgId int64, fromS2s bool) string {
	// 执行推送到客户端和转发到其他服务器
	ts := time.Now().UnixNano()
	clientMsg := fmt.Sprintf("%d %s %s %d %d %s",MESSAGE_TYPE_NORMAL, channel, userId, ts, msgId, msg)
	PublishStrMsg(channel, clientMsg, userId, !fromS2s)

	// 保存到数据库
	storage.MsgCache.Store(&storage.MsgTuple{Channel: channel, Msg: msg, Ts: ts, Uuid: userId})
	if !fromS2s && config.PushdConf.EnableStorage() {
		storage.EnqueueMsg(channel, msg, userId, ts, msgId)
	}

	return ""
}

func PushToClients(msg string, isToOtherServer bool, userIds ...string){
	for _, ele := range userIds {
		cli, exists := UuidToClient.GetClient(ele)
		if !exists {
			continue
		}
		go cli.WriteFormatBinMsg(OUTPUT_RCIV, []byte(msg))
	}
}

// 推送和转发字符类型消息
func PublishStrMsg(channel, msg, ownerId string, isToOtherServer bool) {
	pushToNativeClient(OUTPUT_RCIV, channel, ownerId, []byte(msg))
	if isToOtherServer {
		msg2 := fmt.Sprintf("%s %s", channel, msg)
		forwardToAllOtherServer(S2S_PUSH_STRING_MESSAGE, []byte(msg2))
	}
}

// 推送和转发二进制类型消息
func PublishBinMsg(channel, ownerId string, msg []byte, isToOtherServer bool) {
	pushToNativeClient(CMD_VIDO_CHAT, channel, ownerId, msg)
	if isToOtherServer {
		channelByte := []byte(channel)
		buff := bytes.NewBuffer(make([]byte, 0, len(channelByte) + 1 + len(msg)))
		buff.Write(channelByte)
		buff.WriteByte(' ')
		buff.Write(msg)
		forwardToAllOtherServer(CMD_VIDO_CHAT, buff.Bytes())
	}
}

/**
	获取所有在群聊channelId中并且在线的userId集合
 */
func fetchAllMatchUserIds(channelId string) ([]string, error){
	return LoadUserIdsByChannel(channelId), nil
}

// 推送给本地连接的客户端
func pushToNativeClient(op, channelId, ownerId string, msg []byte){
	userIds, err := fetchAllMatchUserIds(channelId)
	if err != nil {
		log.Error(fmt.Sprintf("chatRoom(id: %s) has no userIds", channelId))
		return
	}

	for _, ele := range userIds {
		cli, exists := UuidToClient.GetClient(ele)
		if !exists || cli.uuid == ownerId{
			continue
		}
		go cli.WriteFormatBinMsg(op, msg)
	}

}

// 转发给所有其他节点
func forwardToAllOtherServer(cmd string, message []byte){
	if !config.PushdConf.IsDistMode() || cmd == "" || len(message) == 0{
		return
	}

	peers := set.NewSet()
	for _, v := range Proxy.Router.Peers {
		peers.Add(v)
	}
	Proxy.ForwardTuple <- NewForwardTuple(peers, message, cmd)
}

// 根据channel转发给相应的节点
func forwardToOtherServerInChannel(cmd , channelId string, message []byte){
	if !config.PushdConf.IsDistMode() || cmd == "" || channelId == "" || len(message) == 0{
		return
	}

	peers := set.NewSet()
	for _, v := range Proxy.Router.Peers {
		peers.Add(v)
	}
	Proxy.ForwardTuple <- NewForwardTuple(peers, message, cmd)
}
