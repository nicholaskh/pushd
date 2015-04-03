package engine

import (
	"fmt"
	"strings"
	"time"

	"github.com/nicholaskh/golib/server"
	log "github.com/nicholaskh/log4go"
)

type ClientHandler struct {
	serv           *server.TcpServer
	serverStats    *ServerStats
	client         *Client
	enableAclCheck bool
}

func NewClientHandler(serv *server.TcpServer, serverStats *ServerStats) *ClientHandler {
	return &ClientHandler{serv: serv, serverStats: serverStats}
}

func (this *ClientHandler) OnAccept(cli *server.Client) {
	c := NewClient()
	c.Client = cli
	this.client = c
}

func (this *ClientHandler) OnRead(input string) {
	var (
		t1      time.Time
		elapsed time.Duration
	)

	t1 = time.Now()

	for _, inputUnit := range strings.Split(input, "\n") {
		cl := NewCmdline(inputUnit, this.client)
		if cl.Cmd == "" {
			continue
		}

		if this.enableAclCheck {
			err := AclCheck(this.client, cl.Cmd)
			if err != nil {
				this.client.WriteMsg(err.Error())
				continue
			}
		}

		ret, err := cl.Process()
		if err != nil {
			log.Debug("Process cmd[%s %s] error: %s", cl.Cmd, cl.Params, err.Error())
			this.client.WriteMsg(fmt.Sprintf("%s\n", err.Error()))
			continue
		}

		this.client.WriteMsg(fmt.Sprintf("%s\n", ret))

		elapsed = time.Since(t1)
		this.serverStats.CallLatencies.Update(elapsed.Nanoseconds() / 1e6)
		this.serverStats.CallPerSecond.Mark(1)
	}

}

func (this *ClientHandler) OnClose() {
	log.Debug("client channels: %s", this.client.Channels)
	log.Debug("pubsub channels: %s", PubsubChannels)

	UnsubscribeAllChannels(this.client)
	this.client.Close()
}

func (this *ClientHandler) EnableAclCheck() {
	this.enableAclCheck = true
}

func (this *ClientHandler) DisableAclCheck() {
	this.enableAclCheck = false
}
