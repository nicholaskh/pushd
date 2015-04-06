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
}

func NewPushdClientProcessor(server *server.TcpServer, serverStats *ServerStats) *PushdClientProcessor {
	this := new(PushdClientProcessor)
	this.server = server
	this.serverStats = serverStats

	return this
}

func (this *PushdClientProcessor) Run() {
	log.Debug("start server go routine")
	this.server.AcceptLock.Acquire()
	conn, err := this.server.Fd.(*net.TCPListener).AcceptTCP()
	this.server.AcceptLock.Release()
	if err != nil {
		log.Error("Accept error: %s", err.Error())
	}

	go this.Run()

	client := NewClient()
	client.Client = server.NewClient(conn, time.Now(), this.server.SessTimeout)

	if this.server.SessTimeout.Nanoseconds() > int64(0) {
		go client.Client.CheckTimeout(client.Close)
	}

	for {
		input := make([]byte, 1460)
		n, err := client.Conn.Read(input)

		input = input[:n]

		if err != nil {
			if err == io.EOF {
				log.Info("Client shutdown: %s", client.Conn.RemoteAddr())
				client.Close()
				return
			} else if nerr, ok := err.(net.Error); !ok || !nerr.Temporary() {
				log.Error("Read from client[%s] error: %s", client.Conn.RemoteAddr(), err.Error())
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
				go client.WriteMsg(err.Error())
				continue
			}
		}

		ret, err := cl.Process()
		if err != nil {
			log.Debug("Process cmd[%s %s] error: %s", cl.Cmd, cl.Params, err.Error())
			go client.WriteMsg(fmt.Sprintf("%s\n", err.Error()))
			continue
		}

		go client.WriteMsg(fmt.Sprintf("%s\n", ret))

		elapsed = time.Since(t1)
		this.serverStats.CallLatencies.Update(elapsed.Nanoseconds() / 1e6)
		this.serverStats.CallPerSecond.Mark(1)
	}

}

func (this *PushdClientProcessor) EnableAclCheck() {
	this.enableAclCheck = true
}

func (this *PushdClientProcessor) DisableAclCheck() {
	this.enableAclCheck = false
}

func (this *PushdClientProcessor) Close() {

}
