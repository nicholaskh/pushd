package engine

import (
	"io"
	"net"
	"strings"
	"time"

	"github.com/nicholaskh/golib/server"
	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/config"
	"bytes"
	"fmt"
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

		strInput := string(input)
		log.Debug("input: %s", strInput)

		this.OnRead(client, strInput)
	}
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
		publish(cl.Params[0], cl.Params[3], cl.Params[1], true)

	case S2S_SUB_CMD:
		log.Debug("Remote addr %s sub: %s", client.RemoteAddr(), cl.Params[0])
		if len(cl.Params) == 3{
			me, exists := UuidToClient.GetClient(cl.Params[0])
			if exists {
				var channel string
				if bytes.Compare([]byte(cl.Params[0]), []byte(cl.Params[1])) > 0 {
					channel = fmt.Sprintf("pri_%s_%s", cl.Params[0], cl.Params[1])
				} else {
					channel = fmt.Sprintf("pri_%s_%s", cl.Params[1], cl.Params[0])
				}

				_, exists := me.Channels[channel]
				if exists {
					_, exists := Proxy.Router.LookupPeersByChannel(channel)
					if !exists {
						peerAddr := config.GetS2sAddr(client.RemoteAddr().String())
						Proxy.Router.AddPeerToChannel(peerAddr, channel)

						// notify friend of the news that I am in here
						peer, _ := Proxy.Router.Peers[peerAddr]
						go peer.writeMsg(fmt.Sprintf("sub %s", channel))

						msg := fmt.Sprintf("%s %s", cl.Params[1], cl.Params[2])
						go me.WriteMsg(msg)
					}

				} else {
					subscribe(me, channel, -1)

					peerAddr := config.GetS2sAddr(client.RemoteAddr().String())
					Proxy.Router.AddPeerToChannel(peerAddr, channel)

					peer, _ := Proxy.Router.Peers[peerAddr]
					go peer.writeMsg(fmt.Sprintf("sub %s", channel))

					msg := fmt.Sprintf("%s %s", cl.Params[1], cl.Params[2])
					go me.WriteMsg(msg)
				}

			}

		} else {
			Proxy.Router.AddPeerToChannel(config.GetS2sAddr(client.RemoteAddr().String()), cl.Params[0])
		}


	case S2S_UNSUB_CMD:
		log.Debug("Remote addr %s unsub: %s", client.RemoteAddr(), cl.Params[0])
		Proxy.Router.RemovePeerFromChannel(config.GetS2sAddr(client.RemoteAddr().String()), cl.Params[0])
	}

	return nil
}
