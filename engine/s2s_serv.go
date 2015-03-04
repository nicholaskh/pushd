package engine

import (
	"io"
	"net"
	"time"

	"github.com/nicholaskh/golib/server"
	"github.com/nicholaskh/golib/set"
	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/client"
	"github.com/nicholaskh/pushd/config"
)

type S2sServ struct {
	*server.Server
}

func NewS2sServ() (this *S2sServ) {
	this = new(S2sServ)
	this.Server = server.NewServer("pushd_s2s")
	return
}

func (this *S2sServ) Run(cli *server.Client) {
	client := client.NewClient()
	client.Client = cli
	for {
		input := make([]byte, 1460)
		_, err := client.Conn.Read(input)

		if err != nil {
			if err == io.EOF {
				log.Info("Peer shutdown: %s", client.Conn.RemoteAddr())
				client.Conn.Close()
				return
			} else if nerr, ok := err.(net.Error); !ok || !nerr.Temporary() {
				log.Error("Read from peer[%s] error: %s", client.Conn.RemoteAddr(), err.Error())
				return
			}
		}

		client.LastTime = time.Now()

		log.Debug("peer input: %x", input)

		cl := NewCmdline(input, client)
		if cl.Cmd == "" {
			continue
		}

		err = this.processCmd(cl)

		if err != nil {
			log.Debug("Process peer cmd[%s %s] error: %s", cl.Cmd, cl.Params, err.Error())
			client.Conn.Write([]byte(err.Error()))
			continue
		}
	}

}

func (this *S2sServ) processCmd(cl *Cmdline) error {
	switch cl.Cmd {
	case CMD_PUBLISH:
		publish(cl.Params[0], cl.Params[1])

	case CMD_SUBSCRIBE:
		log.Debug("peer sub %s %s", cl.Cmd, cl.Params)
		peers, exists := Proxy.GetPeersByChannel(cl.Params[0])
		if !exists {
			peers = set.NewSet()
		}
		peers.Add(Proxy.peers[GetS2sAddr(cl.Params[1])])
		Proxy.channelPeers.Set(cl.Params[0], peers)
	}

	return nil
}

func (this *S2sServ) LaunchProxyServ() {
	s := NewS2sServ()
	//FIXME
	//s.LaunchTcpServ(confPushd.s2sAddr, s, confPushd.s2sPingInterval)
	s.LaunchTcpServ(GetS2sAddr(config.PushdConf.TcpListenAddr), s, config.PushdConf.S2sConnTimeout)
}
