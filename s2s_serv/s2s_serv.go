package s2s_serv

import (
	"io"
	"net"
	"time"

	"github.com/nicholaskh/golib/server"
	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/client"
	"github.com/nicholaskh/pushd/engine"
	"github.com/nicholaskh/pushd/s2s_proxy"
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

		cl := engine.NewCmdline(input, client)
		if cl.cmd == "" {
			continue
		}

		err = this.processCmd(client.cl)

		if err != nil {
			log.Debug("Process peer cmd[%s %s] error: %s", cl.cmd, cl.params, err.Error())
			client.Conn.Write([]byte(err.Error()))
			continue
		}
	}

}

func (this *S2sServ) processCmd(cl *Cmdline) error {
	switch cl.cmd {
	case CMD_PUBLISH {

	}

	if cl.cmd == CMD_SUBSCRIBE {
		peers, exists := s2sProxy.channelPeers.getPeersByChannel(channel)
		if exists {
			peers.Add()
		}
		s2sProxy.channelPeers.Set(channel)
	}
}

func (this *S2sServ) LaunchProxyServ() {
	s := NewS2sServ()
	//FIXME
	//s.LaunchTcpServ(confPushd.s2sAddr, s, confPushd.s2sPingInterval)
	s.LaunchTcpServ(GetS2sAddr(config.PushdConf.TcpListenAddr), s, config.PushdConf.S2sConnTimeout)
}

// TODO port should fixed
func GetS2sAddr(servAddr string) (s2sAddr string) {
	parts := strings.Split(servAddr, ":")
	ip := parts[0]
	port := parts[1]
	intPort, _ := strconv.Atoi(port)
	s2sPort := strconv.Itoa((intPort + 1))
	s2sAddr = fmt.Sprintf("%s:%s", ip, s2sPort)
	return
}
