package main

import (
	"fmt"
	"net"
	"sync"
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
				go conns[j].Write([]byte(fmt.Sprintf("sub channel%d\n", i)))
			} else {
				go conns[j].Write([]byte(fmt.Sprintf("pub channel%d hello\n", i)))
			}
		}
		time.Sleep(time.Second)
	}

	shutdown()

	time.Sleep(time.Second)
}

func buildConns() {
	var wg sync.WaitGroup
	var err error
	tcpAddr, _ := net.ResolveTCPAddr("tcp", options.addr)
	for i := 0; i < options.concurrency; i++ {
		wg.Add(1)
		go func(i int) {
			conns[i], err = net.DialTCP("tcp", nil, tcpAddr)
			if err != nil {
				lostConns++
				log.Info("connection error: %s", err.Error())
			} else {
				//conns[i].SetLinger(0)
			}
			wg.Done()
		}(i)

	}
	wg.Wait()
}

func shutdown() {
	for i, conn := range conns {
		log.Info("close connection %d", i)
		err := conn.Close()
		if err != nil {
			log.Info("close error: %s", err.Error())
		}
	}
}
