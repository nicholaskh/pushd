package main

import (
	"fmt"
	"net"
	"time"

	"github.com/nicholaskh/golib/server"
	log "github.com/nicholaskh/log4go"
)

var (
	conns     []*net.TCPConn
	lostConns int = 0
)

func init() {
	parseFlags()
	conns = make([]*net.TCPConn, options.concurrency)

	server.SetupLogging(options.logFile, options.logLevel, options.crashLogFile)
}

func main() {
	buildConns()

	for i := 0; i < options.requests; i++ {
		log.Info("Start round %d", i)
		for j := 0; j < options.concurrency; j++ {
			if j%2 == 1 {
				go conns[j].Write([]byte(fmt.Sprintf("sub channel%d", i)))
			} else {
				go conns[j].Write([]byte(fmt.Sprintf("pub channel%d hello", i)))
			}
		}
		time.Sleep(time.Second)
	}

	shutdown()
}

func buildConns() {
	var err error
	tcpAddr, _ := net.ResolveTCPAddr("tcp", options.addr)
	for i := 0; i < options.concurrency; i++ {
		conns[i], err = net.DialTCP("tcp", nil, tcpAddr)
		conns[i].SetLinger(0)
		if err != nil {
			lostConns++
			log.Info("connection error: %s", err.Error())
		}

	}
}

func shutdown() {
	for _, conn := range conns {
		conn.Close()
	}
}
