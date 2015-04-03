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
	s2sServ   *server.TcpServer
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
	clientHandler := engine.NewClientHandler(pushdServ, servStats)
	if !options.aclCheck {
		clientHandler.DisableAclCheck()
	}
	go pushdServ.LaunchTcpServer(config.PushdConf.TcpListenAddr, clientHandler, config.PushdConf.SessionTimeout, config.PushdConf.ServInitialGoroutineNum)

	engine.Proxy = engine.NewS2sProxy()
	go engine.Proxy.WaitMsg()

	s2sServ = server.NewTcpServer("pushd_s2s")
	go s2sServ.LaunchTcpServer(engine.GetS2sAddr(config.PushdConf.TcpListenAddr), &engine.S2sClientHandler{}, config.PushdConf.S2sSessionTimeout, config.PushdConf.S2sIntialGoroutineNum)

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
