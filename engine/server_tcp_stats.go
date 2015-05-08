package engine

import (
	logger "log"
	"net/http"
	"os"
	"runtime"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/nicholaskh/golib/server"
	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/metrics"
	"github.com/nicholaskh/pushd/config"
)

type ServerStats struct {
	*server.HttpJsonServer
	CallLatencies metrics.Histogram
	CallPerSecond metrics.Meter
}

func NewServerStats() (this *ServerStats) {
	this = new(ServerStats)
	this.HttpJsonServer = server.NewHttpJsonServer()
	this.CallLatencies = metrics.NewHistogram(metrics.NewExpDecaySample(1028, 0.015))
	metrics.Register("latency.call", this.CallLatencies)
	this.CallPerSecond = metrics.NewMeter()
	metrics.Register("qps.call", this.CallPerSecond)

	return
}

func (this *ServerStats) Start(interval time.Duration, logFile string) {
	this.launchHttpServer()
	defer this.stopHttpServer()
	metricsWriter, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	if interval > 0 {
		metrics.Log(metrics.DefaultRegistry, interval, logger.New(metricsWriter, "", logger.LstdFlags))
	}
}

func (this *ServerStats) launchHttpServer() {
	if config.PushdConf.StatsListenAddr == "" {
		return
	}

	this.LaunchHttpServer(config.PushdConf.StatsListenAddr, config.PushdConf.ProfListenAddr)
	this.RegisterHttpApi("/s/{cmd}",
		func(w http.ResponseWriter, req *http.Request,
			params map[string]interface{}) (interface{}, error) {
			return this.handleHttpQuery(w, req, params)
		}).Methods("GET")
}

func (this *ServerStats) handleHttpQuery(w http.ResponseWriter, req *http.Request,
	params map[string]interface{}) (interface{}, error) {
	var (
		vars   = mux.Vars(req)
		cmd    = vars["cmd"]
		output = make(map[string]interface{})
	)

	switch cmd {
	case "ping":
		output["status"] = "ok"

	case "trace":
		stack := make([]byte, 1<<20)
		stackSize := runtime.Stack(stack, true)
		output["callstack"] = string(stack[:stackSize])

	case "sys":
		output["goroutines"] = runtime.NumGoroutine()

		memStats := new(runtime.MemStats)
		runtime.ReadMemStats(memStats)
		output["memory"] = *memStats

		rusage := syscall.Rusage{}
		syscall.Getrusage(0, &rusage)
		output["rusage"] = rusage

	case "stat":
		output["ver"] = server.VERSION
		output["build"] = server.BuildID
		output["stats"] = map[string]interface{}{
			"pubsub_channel_count": PubsubChannels.Len(),
			"s2s_channel_count":    Proxy.Router.cache.Len(),
		}
		output["conf"] = *config.PushdConf

	default:
		return nil, server.ErrHttp404
	}

	if config.PushdConf.ProfListenAddr != "" {
		output["pprof"] = "http://" +
			config.PushdConf.ProfListenAddr + "/debug/pprof/"
	}

	return output, nil
}

func (this *ServerStats) stopHttpServer() {
	log.Info("stats httpd stopped")
	this.StopHttpServer()
}
