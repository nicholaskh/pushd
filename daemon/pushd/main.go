package main

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"runtime/debug"

	"github.com/nicholaskh/golib/server"
	"github.com/nicholaskh/golib/signal"
	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/config"
	"github.com/nicholaskh/pushd/engine"
	"github.com/nicholaskh/pushd/engine/storage"
	"github.com/nicholaskh/pushd/serv"
)

var (
	pushdServ *serv.PushdServ
	s2sServ   *engine.S2sServ
)

func init() {
	parseFlags()

	if options.showVersion {
		server.ShowVersionAndExit()
	}

	server.SetupLogging(options.logFile, options.logLevel, options.crashLogFile)
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
			debug.PrintStack()
		}
	}()

	pushdServ = serv.NewPushdServ()
	pushdServ.LoadConfig(options.configFile)
	pushdServ.Launch()
	go server.RunSysStats(time.Now(), time.Duration(options.tick)*time.Second)

	config.PushdConf = new(config.ConfigPushd)
	config.PushdConf.LoadConfig(pushdServ.Conf)
	go pushdServ.LaunchTcpServ(config.PushdConf.TcpListenAddr, pushdServ, config.PushdConf.SessionTimeout, config.PushdConf.ServInitialGoroutineNum)

	engine.Proxy = engine.NewS2sProxy()
	go engine.Proxy.WaitMsg()

	s2sServ = engine.NewS2sServ()
	go s2sServ.LaunchTcpServ(engine.GetS2sAddr(config.PushdConf.TcpListenAddr), s2sServ, config.PushdConf.S2sSessionTimeout, config.PushdConf.S2sIntialGoroutineNum)

	if config.PushdConf.EnableStorage() {
		storage.Init()
		go storage.Serv()
	}

	signal.RegisterSignalHandler(syscall.SIGINT, func(sig os.Signal) {
		shutdown()
	})

	pushdServ.Stats.Start(config.PushdConf.StatsOutputInterval, config.PushdConf.MetricsLogfile)

	shutdown()
}

func shutdown() {
	pushdServ.StopTcpServ()
	log.Info("Terminated")
	os.Exit(0)
}
