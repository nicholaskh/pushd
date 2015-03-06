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
	*server.Server
	Stats *serverStats
}

func NewPushdServ() (this *PushdServ) {
	this = new(PushdServ)
	this.Server = server.NewServer("pushd")
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
				jsonChannels, err := engine.PubsubChannels.MarshalJSON()
				if err != nil {
					log.Error("Json marshal error: %s", err.Error())
				}

				log.Debug("pubsub channels: %s", jsonChannels)
				return
			} else if nerr, ok := err.(net.Error); !ok || !nerr.Temporary() {
				log.Error("Read from client[%s] error: %s", client.Conn.RemoteAddr(), err.Error())
				return
			}
		}

		client.LastTime = time.Now()

		//log.Debug("input: %x", input)

		cl := engine.NewCmdline(input, client)
		if cl.Cmd == "" {
			continue
		}
		ret, err := cl.ProcessCmd()
		if err != nil {
			log.Debug("Process cmd[%s %s] error: %s", cl.Cmd, cl.Params, err.Error())
			client.Output <- err.Error()
			continue
		}

		client.Output <- fmt.Sprintf("%s\n", ret)

		elapsed = time.Since(t1)
		log.Info(elapsed.Nanoseconds())
		this.Stats.CallLatencies.Update(elapsed.Nanoseconds() / 1e6)
		this.Stats.CallPerSecond.Mark(1)
	}

}

func (this *PushdServ) closeClient(client *client.Client) {
	client.Close()
	engine.UnsubscribeAllChannels(client)
}
