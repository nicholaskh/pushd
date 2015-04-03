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
)

var (
	pushdServ *server.TcpServer
	s2sServer *server.TcpServer
)

func init() {
	parseFlags()

	if options.showVersion {
		server.ShowVersionAndExit()
	}

	server.SetupLogging(options.logFile, options.logLevel, options.crashLogFile)

	conf := server.LoadConfig(options.configFile)
	config.PushdConf = new(config.ConfigPushd)
	config.PushdConf.LoadConfig(conf)

	engine.PubsubChannels = engine.NewPubsubChannels(config.PushdConf.PubsubChannelMaxItems)
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
			debug.PrintStack()
		}
		shutdown()
	}()

	pushdServ = server.NewTcpServer("pushd")
	go server.RunSysStats(time.Now(), time.Duration(options.tick)*time.Second)

	servStats := engine.NewServerStats()
	pushdClientProcessor := engine.NewPushdClientProcessor(pushdServ, servStats)
	if !options.aclCheck {
		pushdClientProcessor.DisableAclCheck()
	}
	go pushdServ.LaunchTcpServer(config.PushdConf.TcpListenAddr, pushdClientProcessor, config.PushdConf.SessionTimeout, config.PushdConf.ServInitialGoroutineNum)

	engine.Proxy = engine.NewS2sProxy()
	go engine.Proxy.WaitMsg()

	s2sServer = server.NewTcpServer("pushd_s2s")
	go s2sServer.LaunchTcpServer(engine.GetS2sAddr(config.PushdConf.TcpListenAddr), engine.NewS2sClientProcessor(s2sServer), config.PushdConf.S2sSessionTimeout, config.PushdConf.S2sIntialGoroutineNum)

	if config.PushdConf.EnableStorage() {
		storage.Init()
		go storage.Serv()
	}

	signal.RegisterSignalHandler(syscall.SIGINT, func(sig os.Signal) {
		shutdown()
	})

	servStats.Start(config.PushdConf.StatsOutputInterval, config.PushdConf.MetricsLogfile)

}

func shutdown() {
	pushdServ.StopTcpServ()
	log.Info("Terminated")
	os.Exit(0)
}
