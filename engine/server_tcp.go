package engine

import (
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/nicholaskh/golib/server"
	log "github.com/nicholaskh/log4go"
)

type PushdClientProcessor struct {
	enableAclCheck bool
	server         *server.TcpServer
	serverStats    *ServerStats
	proto          *server.Protocol
}

func NewPushdClientProcessor(server *server.TcpServer, serverStats *ServerStats) *PushdClientProcessor {
	this := new(PushdClientProcessor)
	this.server = server
	this.serverStats = serverStats
	this.enableAclCheck = true

	return this
}

func (this *PushdClientProcessor) OnAccept(c *server.Client) {
	client := NewClient()
	client.Client = c

	for {
		if this.server.SessTimeout.Nanoseconds() > int64(0) {
			client.SetReadDeadline(time.Now().Add(this.server.SessTimeout))
		}
		input, err := client.Proto.Read()

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

		this.OnRead(client, string(input))
	}
}

func (this *PushdClientProcessor) OnRead(client *Client, input string) {
	var (
		t1      time.Time
		elapsed time.Duration
	)

	t1 = time.Now()
	for _, inputUnit := range strings.Split(input, "\n") {
		cl := NewCmdline(inputUnit, client)
		if cl.Cmd == "" {
			continue
		}

		if this.enableAclCheck {
			err := AclCheck(client, cl.Cmd)
			if err != nil {
				go client.WriteMsg(fmt.Sprintf("%s", err.Error()))
				continue
			}

			err = TokenCheck(cl)
			if err != nil {
				go client.WriteMsg(fmt.Sprintf("%s", err.Error()))
				continue
			}
		}

		ret, err := cl.Process()
		if err != nil {
			log.Debug("Process cmd[%s %s] error: %s", cl.Cmd, cl.Params, err.Error())
			go client.WriteMsg(fmt.Sprintf("%s\n", err.Error()))
			continue
		}

		go client.WriteMsg(ret)
	}
	elapsed = time.Since(t1)
	this.serverStats.CallLatencies.Update(elapsed.Nanoseconds() / 1e6)
	this.serverStats.CallPerSecond.Mark(1)

}

func (this *PushdClientProcessor) EnableAclCheck() {
	this.enableAclCheck = true
}

func (this *PushdClientProcessor) DisableAclCheck() {
	this.enableAclCheck = false
}
