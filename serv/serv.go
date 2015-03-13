package serv

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/nicholaskh/golib/server"
	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/client"
	"github.com/nicholaskh/pushd/engine"
)

type PushdServ struct {
	*server.TcpServer
	Stats *serverStats
}

func NewPushdServ() (this *PushdServ) {
	this = new(PushdServ)
	this.TcpServer = server.NewTcpServer("pushd")
	this.Stats = newServerStats()

	return
}

func (this *PushdServ) Run(cli *server.Client) {
	client := client.NewClient()
	client.Client = cli
	go client.WaitMsg()

	var (
		t1      time.Time
		elapsed time.Duration
	)
	for {
		input := make([]byte, 1460)
		_, err := client.Conn.Read(input)

		t1 = time.Now()

		if err != nil {
			if err == io.EOF {
				log.Info("Client shutdown: %s", client.Conn.RemoteAddr())
				this.closeClient(client)

				log.Debug("client channels: %s", client.Channels)

				log.Debug("pubsub channels: %s", engine.PubsubChannels)
				return
			} else if nerr, ok := err.(net.Error); !ok || !nerr.Temporary() {
				log.Error("Read from client[%s] error: %s", client.Conn.RemoteAddr(), err.Error())
				this.closeClient(client)
				return
			}
		}

		if err != nil {
			log.Debug("Client auth fail: %s", err.Error())
			this.sendClient(client, err.Error())
		}

		client.LastTime = time.Now()

		//log.Debug("input: %x", input)

		cl := engine.NewCmdline(input, client)
		if cl.Cmd == "" {
			continue
		}

		//err = engine.AclCheck(client, cl.Cmd)
		//if err != nil {
		//	this.sendClient(client, err.Error())
		//	continue
		//}

		ret, err := cl.Process()
		if err != nil {
			log.Debug("Process cmd[%s %s] error: %s", cl.Cmd, cl.Params, err.Error())
			this.sendClient(client, err.Error())
			continue
		}

		this.sendClient(client, ret)

		elapsed = time.Since(t1)
		this.Stats.CallLatencies.Update(elapsed.Nanoseconds() / 1e6)
		this.Stats.CallPerSecond.Mark(1)
	}

}

func (this *PushdServ) sendClient(cli *client.Client, msg string) {
	cli.Output <- fmt.Sprintf("%s\n", msg)
}

func (this *PushdServ) closeClient(cli *client.Client) {
	engine.UnsubscribeAllChannels(cli)
	cli.Close()
}
