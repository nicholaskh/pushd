package engine

import (
	"io"
	"net"
	"strings"
	"time"

	"github.com/nicholaskh/golib/server"
	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/config"
	"strconv"
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
		params := strings.SplitN(cl.Params, " ", 5)
		msgId, err := strconv.ParseInt(params[3], 10, 64)
		if err != nil {
			return err
		}
		Publish(params[0], params[4], params[1], msgId, true)

	case S2S_SUB_CMD:
		log.Debug("Remote addr %s sub: %s", client.RemoteAddr(), cl.Params)
		Proxy.Router.AddPeerToChannel(config.GetS2sAddr(client.RemoteAddr().String()), cl.Params)

	case S2S_UNSUB_CMD:
		log.Debug("Remote addr %s unsub: %s", client.RemoteAddr(), cl.Params)
		Proxy.Router.RemovePeerFromChannel(config.GetS2sAddr(client.RemoteAddr().String()), cl.Params)
	}

	return nil
}
