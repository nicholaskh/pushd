package main

import (
	"os"

	"github.com/nicholaskh/golib/server"
	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/config"
	//"github.com/nicholaskh/pushd/s2s"
	"github.com/nicholaskh/pushd/serv"
)

var (
	pushdServ *serv.PushdServ
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
	pushdServ.LaunchTcpServ(config.PushdConf.TcpListenAddr, pushdServ, config.PushdConf.SessionTimeout)
	/*
		s2s.Proxy = NewS2sProxy()
		s2s.Proxy.startProxyServ()
	*/
	shutdown()
}

func shutdown() {
	pushdServ.StopTcpServ()
	log.Info("Terminated")
	os.Exit(0)
}
