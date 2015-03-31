package engine

import (
	"strings"

	"github.com/nicholaskh/golib/server"
	"github.com/nicholaskh/golib/set"
	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/client"
)

type S2sClientHandler struct {
	client *client.Client
	serv   *server.TcpServer
}

func NewS2sClientHandler(serv *server.TcpServer) *S2sClientHandler {
	return &S2sClientHandler{serv: serv}
}

func (this *S2sClientHandler) OnAccept(cli *server.Client) {
	c := client.NewClient()
	c.Client = cli
	this.client = c
}

func (this *S2sClientHandler) OnRead(input string) {
	for _, inputUnit := range strings.Split(input, "\n") {
		cl := NewCmdline(inputUnit, this.client)
		if cl.Cmd == "" {
			continue
		}

		err := this.processCmd(cl)

		if err != nil {
			log.Debug("Process peer cmd[%s %s] error: %s", cl.Cmd, cl.Params, err.Error())
			this.client.WriteMsg(err.Error())
			continue
		}
	}
}

func (this *S2sClientHandler) OnClose() {
	this.client.Close()
}

func (this *S2sClientHandler) processCmd(cl *Cmdline) error {
	switch cl.Cmd {
	case CMD_PUBLISH:
		publish(cl.Params[0], cl.Params[1], true)

	case CMD_SUBSCRIBE:
		log.Debug("peer sub %s %s", cl.Cmd, cl.Params)
		peers, exists := Proxy.GetPeersByChannel(cl.Params[0])
		if !exists {
			peers = set.NewSet()
		}
		peers.Add(Proxy.peers[GetS2sAddr(cl.Params[1])])
		Proxy.ChannelPeers.Set(cl.Params[0], peers)
	}

	return nil
}
