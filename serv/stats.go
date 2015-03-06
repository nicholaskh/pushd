package serv

import (
	"log"
	"os"
	"time"

	"github.com/nicholaskh/metrics"
)

type serverStats struct {
	CallLatencies metrics.Histogram
	CallPerSecond metrics.Meter
}

func newServerStats() (this *serverStats) {
	this = new(serverStats)
	this.CallLatencies = metrics.NewHistogram(metrics.NewExpDecaySample(1028, 0.015))
	metrics.Register("latency.call", this.CallLatencies)
	this.CallPerSecond = metrics.NewMeter()
	metrics.Register("qps.call", this.CallPerSecond)

	return
}

func (this *serverStats) Start(interval time.Duration, logFile string) {
	metricsWriter, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	if interval > 0 {
		metrics.Log(metrics.DefaultRegistry, interval, log.New(metricsWriter, "", log.LstdFlags))
	}
}
