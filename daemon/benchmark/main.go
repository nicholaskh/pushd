package main

import (
	"fmt"
	"math"
	"net"
	"sync"
	"time"

	"github.com/nicholaskh/golib/server"
	log "github.com/nicholaskh/log4go"
)

var (
	conns     []net.Conn
	lostConns int = 0
	wg        sync.WaitGroup
)

func init() {
	parseFlags()
	conns = make([]net.Conn, options.concurrency)

	server.SetupLogging(options.logFile, options.logLevel, options.crashLogFile)
}

func main() {
	buildConns()

	for i := 0; i < options.requests; i++ {
		log.Info("Start round %d", i)
		batchStart := 0
		for connLeft := options.concurrency; connLeft > 0; connLeft -= options.batchSize {
			thisBatch := int(math.Min(float64(connLeft), float64(options.batchSize)))
			go batchWrite(thisBatch, batchStart, i)
			batchStart += thisBatch
		}
		time.Sleep(time.Second)
	}

	shutdown()

	time.Sleep(time.Second)
}

func buildConns() {
	batchStart := 0
	for connLeft := options.concurrency; connLeft > 0; connLeft -= options.batchSize {
		wg.Add(1)
		thisBatch := int(math.Min(float64(connLeft), float64(options.batchSize)))
		go batchConn(thisBatch, batchStart)
		batchStart += thisBatch
	}

	wg.Wait()
}

func batchConn(batchSize, firstNum int) {
	var err error
	for i := 0; i < batchSize; i++ {
		log.Info("%d", i)
		conns[firstNum+i], err = net.DialTimeout("tcp", options.addr, options.connTimeout)
		if err != nil {
			lostConns++
			log.Info("connection error: %s", err.Error())
		}
	}
	wg.Done()
	log.Info("Established %d connections", batchSize)
}

func batchWrite(batchSize, firstNum, round int) {
	for i := 0; i < batchSize; i++ {
		if conns[firstNum+i] != nil {
			if i%2 == 1 {
				conns[firstNum+i].Write([]byte(fmt.Sprintf("sub channel%d\n", round)))
			} else {
				conns[firstNum+i].Write([]byte(fmt.Sprintf("pub channel%d hello\n", round)))
			}
		}
	}
}

func shutdown() {
	for i, conn := range conns {
		log.Debug("close connection %d", i)
		err := conn.Close()
		if err != nil {
			log.Info("close error: %s", err.Error())
		}
	}
}
