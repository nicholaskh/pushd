package engine

import (
	"io"
	"net"
	"strings"
	"time"

	"github.com/nicholaskh/golib/server"
	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/config"
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
		log.Debug("Remote addr %s sub: %s", client.RemoteAddr(), cl.Params[1])
		if cl.Params[0] != ""{
			me, exists := UuidToClient.GetClient(cl.Params[0])
			if exists {
				_, exists := me.Channels[cl.Params[2]]
				if exists {
					_, exists := Proxy.Router.LookupPeersByChannel(cl.Params[2])
					if !exists {
						peerAddr := config.GetS2sAddr(client.RemoteAddr().String())
						Proxy.Router.AddPeerToChannel(peerAddr, cl.Params[2])
						peer := Proxy.Router.Peers[peerAddr]

						// say I am here
						go peer.writeMsg(fmt.Sprintf("%s   %s\n", S2S_SUB_CMD, cl.Params[2]))

						go me.WriteMsg(fmt.Sprintf("%s %s", cl.Params[1], cl.Params[2]))
					} else {
						remoteAddr := config.GetS2sAddr(client.RemoteAddr().String())
						isNeedClean := true
						peers,_ := Proxy.Router.LookupPeersByChannel(remoteAddr)
						for p := range peers.Iter() {
							peer := p.(*Peer)
							if peer.addr == remoteAddr {
								isNeedClean = false
								break
							}
							go peer.writeMsg(fmt.Sprintf("%s %s\n", S2S_UNSUB_CMD, cl.Params[2]))
						}

						// this case happen when one client crashed then connect to another server
						if isNeedClean {
							peers.Clear()
							Proxy.Router.AddPeerToChannel(remoteAddr, cl.Params[2])

							go me.WriteMsg(fmt.Sprintf("%s %s", cl.Params[1], cl.Params[2]))
						}
					}
				} else {
					subscribe(me, cl.Params[2])

					peerAddr := config.GetS2sAddr(client.RemoteAddr().String())
					Proxy.Router.AddPeerToChannel(peerAddr, cl.Params[2])

					go me.WriteMsg(fmt.Sprintf("%s %s", cl.Params[1], cl.Params[2]))
				}
			}
		} else {
			Proxy.Router.AddPeerToChannel(config.GetS2sAddr(client.RemoteAddr().String()), cl.Params[2])
		}


	case S2S_UNSUB_CMD:
		log.Debug("Remote addr %s unsub: %s", client.RemoteAddr(), cl.Params[0])
		Proxy.Router.RemovePeerFromChannel(config.GetS2sAddr(client.RemoteAddr().String()), cl.Params[0])
	}

	return nil
}
