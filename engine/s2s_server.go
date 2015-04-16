package engine

import (
	"io"
	"net"
	"strings"
	"time"

	"github.com/nicholaskh/golib/server"
	"github.com/nicholaskh/golib/set"
	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/config"
)

type S2sClientProcessor struct {
	server *server.TcpServer
}

func NewS2sClientProcessor(server *server.TcpServer) *S2sClientProcessor {
	return &S2sClientProcessor{server: server}
}

func (this *S2sClientProcessor) Run(client *server.Client) {
	if this.server.SessTimeout.Nanoseconds() > int64(0) {
		go client.CheckTimeout()
	}

	for {
		input := make([]byte, 1460)
		n, err := client.Conn.Read(input)

		input = input[:n]

		if err != nil {
			if err == io.EOF {
				client.Close()
				return
			} else if nerr, ok := err.(net.Error); !ok || !nerr.Temporary() {
				client.Close()
				return
			}
		}

		client.LastTime = time.Now()

		strInput := string(input)
		log.Debug("input: %s", strInput)

		this.OnRead(client, strInput)
	}

	client.Done <- 0
}

func (this *S2sClientProcessor) OnRead(client *server.Client, input string) {
	for _, inputUnit := range strings.Split(input, "\n") {
		cl := NewCmdline(inputUnit, nil)
		if cl.Cmd == "" {
			continue
		}

		err := this.processCmd(cl, client)

		if err != nil {
			log.Debug("Process peer cmd[%s %s] error: %s", cl.Cmd, cl.Params, err.Error())
			go client.WriteMsg(err.Error())
			continue
		}
	}
}

func (this *S2sClientProcessor) processCmd(cl *Cmdline, client *server.Client) error {
	switch cl.Cmd {
	case S2S_PUB_CMD:
		publish(cl.Params[0], cl.Params[1], true)

	case S2S_SUB_CMD:
		log.Debug("peer %s %s", cl.Cmd, cl.Params)
		peers, exists := Proxy.GetPeersByChannel(cl.Params[0])
		if !exists {
			peers = set.NewSet()
		}
		log.Debug("Remote addr: %s", client.RemoteAddr())
		peers.Add(Proxy.peers[config.GetS2sAddr(client.RemoteAddr().String())])
		Proxy.ChannelPeers.Set(cl.Params[0], peers)

	case S2S_UNSUB_CMD:
		log.Debug("peer unsub %s", cl.Params)
		peers, exists := Proxy.GetPeersByChannel(cl.Params[0])
		if !exists {
			log.Error("Peer[%s] unsubscribe unexists channel[%s]", cl.Params[1], cl.Params[0])
		} else {
			peers.Remove(Proxy.peers[config.GetS2sAddr(client.RemoteAddr().String())])
			Proxy.ChannelPeers.Set(cl.Params[0], peers)
		}
	}

	return nil
}
