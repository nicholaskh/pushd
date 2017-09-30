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
	"strconv"
	"github.com/nicholaskh/pushd/db"
	"github.com/nicholaskh/pushd/engine/offpush"
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
		input := make([]byte, 1460)
		n, err := client.Conn.Read(input)

		input = input[:n]

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

		this.OnRead(client, input)
	}
}

func (this *S2sClientProcessor) OnRead(client *server.Client, input []byte) {
	cl, err := NewCmdline(input, nil)
	if err != nil {
		return
	}

	err = this.processCmd(cl, client)

	if err != nil {
		log.Debug("Process peer cmd[%s %s] error: %s", cl.Cmd, cl.Params, err.Error())
		go client.WriteMsg(err.Error())
	}
}

func (this *S2sClientProcessor) processCmd(cl *Cmdline, client *server.Client) error {
	switch cl.Cmd {
	case S2S_PUB_CMD:
		params := strings.SplitN(cl.Params, " ", 2)
		this.processChildCmdOfPub(params[0], params[1])

	case S2S_SUB_CMD:
		log.Debug("Remote addr %s sub: %s", client.RemoteAddr(), cl.Params)

		_, exists := Proxy.Router.LookupPeersByChannel(cl.Params)
		if exists {
			return nil
		}
		Proxy.Router.AddPeerToChannel(config.GetS2sAddr(client.RemoteAddr().String()), cl.Params)

		_, exists = PubsubChannels.Get(cl.Params)
		if exists {
			return nil
		}

		// force local clients to join in this channel
		var result interface{}
		err := db.MgoSession().DB("pushd").C("channel_uuids").
			Find(bson.M{"_id": cl.Params}).
			Select(bson.M{"uuids":1, "_id":0}).
			One(&result)

		if err == nil {
			uuids := result.(bson.M)["uuids"].([]interface{})
			for _, uuid := range uuids {
				tclient, exists := UuidToClient.GetClient(uuid.(string))
				if exists {
					Subscribe(tclient, cl.Params)
				}
			}
		}

	case S2S_UNSUB_CMD:
		log.Debug("Remote addr %s unsub: %s", client.RemoteAddr(), cl.Params)
		Proxy.Router.RemovePeerFromChannel(config.GetS2sAddr(client.RemoteAddr().String()), cl.Params)
	}

	return nil
}

func (this *S2sClientProcessor)processChildCmdOfPub(cmd, params string) {
	switch cmd {
	case S2S_PUSH_CMD:
		/**
			params:	   "channel message"
		 */
		params2 := strings.SplitN(params, " ", 2)
		Publish2(params2[0], params2[1], "", false)

	case S2S_ADD_USER_INFO:
		/**
			params:	   "userId pushId isAllowNotify"
			example:   "7097d5d45754859550d4 91W490747097d5d45754859550d44776d68400046027f 1"

			1: true 0: false
		 */
		t := strings.SplitN(params, " ", 3)
		userId := t[0]
		pushId := t[1]
		isAllowNotify := true
		if t[2] != "1" {
			isAllowNotify = false
		}

		offpush.UpdateOrAddUserInfo(userId, pushId, true, isAllowNotify)

	case S2S_ENABLE_NOTIFY:
		/**
			params:	   "userId"
			example:   "7097d5d45754859550d4"
		 */

		userId := params
		offpush.InvalidUser(userId)


	case S2S_DISABLE_NOTIFY:
		/**
			params:	   "userId"
			example:   "7097d5d45754859550d4"
 		*/

		userId := params
		offpush.ValidUser(userId)

	case S2S_USER_OFFLINE:
		/**
			params:	   "userId"
			example:   "7097d5d45754859550d4"
	 	*/

		userId := params
		offpush.ChangeUserStatus(userId, false)

	default:
		// TODO 为其指定一个命令标志
		params := strings.SplitN(params, " ", 5)
		msgId, err := strconv.ParseInt(params[3], 10, 64)
		if err != nil {
			return
		}
		Publish(params[0], params[4], params[1], msgId, true)
	}
}
