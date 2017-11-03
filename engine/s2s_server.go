package engine

import (
	"io"
	"net"
	"strings"
	"time"

	"gopkg.in/mgo.v2/bson"
	"github.com/nicholaskh/golib/server"
	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/config"
	"github.com/nicholaskh/pushd/db"
	"errors"
	"bytes"
	"encoding/binary"
)

type S2sClientProcessor struct {
	server *server.TcpServer
}

func NewS2sClientProcessor(server *server.TcpServer) *S2sClientProcessor {
	return &S2sClientProcessor{server: server}
}

func (this *S2sClientProcessor) OnAccept(client *server.Client) {
	for {
		if this.server.SessTimeout.Nanoseconds() > int64(0) {
			client.SetReadDeadline(time.Now().Add(this.server.SessTimeout))
		}

		input, err := client.Proto.Read()

		if err != nil {
			if err == io.EOF {
				client.Close()
				return
			} else if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				log.Info("client[%s] read timeout", client.RemoteAddr())
				client.Close()
				return
			} else if nerr, ok := err.(net.Error); !ok || !nerr.Temporary() {
				client.Close()
				return
			} else {
				log.Info("Unexpected error: %s", err.Error())
				client.Close()
				return
			}
		}

		if this.OnRead(client, input) != nil {
			client.Close()
			break
		}
	}
}

func (this *S2sClientProcessor) OnRead(client *server.Client, input []byte) error {
	cl, err := NewServerCmdline(input, nil)
	if err != nil {
		return err
	}

	err = this.processCmd(cl, client)

	if err != nil {
		log.Debug("Process peer cmd[%s %s] error: %s", cl.Cmd, cl.Params, err.Error())
	}

	return err
}

type ServerCmdline struct {
	Cmd    string
	Params []byte
	*server.Client
}

func NewServerCmdline(input []byte, cli *server.Client) (this *ServerCmdline, err error) {

	var headerLength int32
	if len(input) < 4 {
		this = nil
		err = errors.New("message has been damaged")
		return
	}

	bodyLenBuffer := bytes.NewBuffer(input[:4])
	binary.Read(bodyLenBuffer, binary.BigEndian, &headerLength)
	headLen := int(headerLength)

	if headLen <= 0 {
		this = nil
		err = errors.New("message has been damaged")
		return
	}

	if len(input) < headLen +4 {
		this = nil
		err = errors.New("message has damaged")
		return
	}

	this = new(ServerCmdline)
	this.Cmd = string(input[4: headLen +4])

	if len(input) <= 4 + headLen + 4 {
		this = nil
		err = errors.New("message has damaged")
		return
	}

	var bodyLength int32
	bodyLenBuffer = bytes.NewBuffer(input[headLen +4: headLen +4+4])
	binary.Read(bodyLenBuffer, binary.BigEndian, &bodyLength)
	bodyLen := int(bodyLength)

	if len(input) != 4 + headLen + 4 + bodyLen {
		this = nil
		err = errors.New("message has damaged")
		return
	}

	this.Params = input[4 + headLen + 4: 4 + headLen + 4 + bodyLen]

	this.Client = cli
	return
}

func (this *S2sClientProcessor) processCmd(cl *ServerCmdline, client *server.Client) error {

	if len(cl.Params) == 0 {
		return errors.New("param wrong")
	}

	switch cl.Cmd {

	case S2S_SUB_CMD:
		channelId := string(cl.Params)
		checkAndSetLocalChannel(channelId, client)

	case S2S_UNSUB_CMD:
		channelId := string(cl.Params)
		Proxy.Router.RemovePeerFromChannel(config.GetS2sAddr(client.RemoteAddr().String()), channelId)

	case S2S_PUSH_BINARY_MESSAGE:
		// 提取channelId
		index1 := 0
		for i, value := range cl.Params {
			if value == ' ' {
				index1 = i
				break
			}
		}
		if index1 == 0 {
			return errors.New("param wrong")
		}

		channelId := string(cl.Params[: index1])
		binMsg := cl.Params[index1+1:]

		checkAndSetLocalChannel(channelId, client)

		PublishBinMsg(channelId, "", binMsg, false)

	case S2S_PUSH_STRING_MESSAGE:
		params := strings.SplitN(string(cl.Params), " ", 2)
		if len(params) != 2 {
			return errors.New("param error")
		}
		channelId := params[0]
		message := params[1]

		checkAndSetLocalChannel(channelId, client)

		PublishStrMsg(channelId, message, "", false)
	}

	return nil
}

// 检查本地是否已存在此channel的订阅，如果没有则设置订阅此channel，并将所有关心此channel的client，强制注册到此channel
func checkAndSetLocalChannel(channelId string, client *server.Client) {
	isLoadFromDatabase := false

	_, exists := Proxy.Router.LookupPeersByChannel(channelId)
	if !exists {
		isLoadFromDatabase = true
		Proxy.Router.AddPeerToChannel(config.GetS2sAddr(client.RemoteAddr().String()), channelId)
	}

	_, exists = PubsubChannels.Get(channelId)
	if exists {
		return
	}

	if !isLoadFromDatabase {
		return
	}

	// 将属于此channel的所有的在线client，注册到这个channel中
	var result interface{}
	err := db.MgoSession().DB("pushd").C("channel_uuids").
		Find(bson.M{"_id": channelId}).
		Select(bson.M{"uuids":1, "_id":0}).
		One(&result)

	if err != nil {
		return
	}

	userIds := result.(bson.M)["uuids"].([]interface{})
	for _, userId := range userIds {
		tempClient, exists := UuidToClient.GetClient(userId.(string))
		if exists {
			Subscribe(tempClient, channelId)
		}
	}
}


