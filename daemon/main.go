package main

import (
	"os"

	"github.com/nicholaskh/golib/server"
	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/config"
	"github.com/nicholaskh/pushd/s2s_proxy"
	"github.com/nicholaskh/pushd/s2s_serv"
	"github.com/nicholaskh/pushd/serv"
)

var (
	pushdServ *serv.PushdServ
	s2sServ   *s2s_serv.S2sServ
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

	s2s_proxy.Proxy = s2s_proxy.NewS2sProxy()
	go s2s_proxy.Proxy.WaitMsg()

	s2sServ = NewS2sServ()
	s2sServ.LaunchProxyServ()

	shutdown()
}

func shutdown() {
	pushdServ.StopTcpServ()
	log.Info("Terminated")
	os.Exit(0)
}
