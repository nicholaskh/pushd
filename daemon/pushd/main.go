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
	pushdServer *server.TcpServer
	s2sServer   *server.TcpServer
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
	engine.UuidToClient = engine.NewUuidClientMap()

	signal.RegisterSignalHandler(syscall.SIGINT, func(sig os.Signal) {
		shutdown()
	})
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
			debug.PrintStack()
		}
		shutdown()
	}()

	pushdServer = server.NewTcpServer("pushd")
	pushdServer.SetProtoType(1)
	go server.RunSysStats(time.Now(), time.Duration(options.tick)*time.Second)

	servStats := engine.NewServerStats()
	pushdClientProcessor := engine.NewPushdClientProcessor(pushdServer, servStats)
	if !options.aclCheck {
		pushdClientProcessor.DisableAclCheck()
	}
	go pushdServer.LaunchTcpServer(config.PushdConf.TcpListenAddr, pushdClientProcessor, config.PushdConf.SessionTimeout, config.PushdConf.ServInitialGoroutineNum)
	go servStats.Start(config.PushdConf.StatsOutputInterval, config.PushdConf.MetricsLogfile)

	if config.PushdConf.LongPollingListenAddr != "" {
		longPollingServer := engine.NewPushdLongPollingServer("pushd(long polling)")
		go longPollingServer.Launch(config.PushdConf.LongPollingListenAddr, config.PushdConf.LongPollingSessionTimeout)
	}

	if config.PushdConf.IsDistMode() {
		s2sServer = server.NewTcpServer("pushd_s2s")
		go s2sServer.LaunchTcpServer(config.PushdConf.S2sListenAddr, engine.NewS2sClientProcessor(s2sServer), config.PushdConf.S2sSessionTimeout, config.PushdConf.S2sIntialGoroutineNum)

		err := engine.RegisterEtc()
		if err != nil {
			panic(err)
		}

		engine.Proxy = engine.NewS2sProxy()
		go engine.Proxy.WaitMsg()
	}

	if config.PushdConf.EnableStorage() {
		storage.Init()
		go storage.Serv()
	}

	<-make(chan int)
}

func shutdown() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
			debug.PrintStack()
		}
	}()
	log.Info("Terminated")

	if config.PushdConf.IsDistMode() {
		err := engine.UnregisterEtc()
		if err != nil {
			log.Error(err)
		}
	}

	if config.PushdConf.IsDistMode() {
		s2sServer.StopTcpServer()
	}
	pushdServer.StopTcpServer()
	time.Sleep(time.Second * 2)
	os.Exit(0)
}
