package main

import (
	"os"

	"github.com/nicholaskh/golib/server"
	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/config"
	"github.com/nicholaskh/pushd/engine"
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
	pushdServ = serv.NewPushdServ()
	pushdServ.LoadConfig(options.configFile)
	pushdServ.Launch()

	config.PushdConf = new(config.ConfigPushd)
	config.PushdConf.LoadConfig(pushdServ.Conf)
	go pushdServ.LaunchTcpServ(config.PushdConf.TcpListenAddr, pushdServ, config.PushdConf.ConnTimeout)

	engine.Proxy = engine.NewS2sProxy()
	go engine.Proxy.WaitMsg()

	s2sServ = engine.NewS2sServ()
	s2sServ.LaunchProxyServ()

	shutdown()
}

func shutdown() {
	pushdServ.StopTcpServ()
	log.Info("Terminated")
	os.Exit(0)
}
