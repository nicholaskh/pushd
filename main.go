package main

import (
	"os"

	"github.com/nicholaskh/golib/server"
	log "github.com/nicholaskh/log4go"
)

var (
	tcpServer *server.Server
	confPushd *ConfigPushd
)

func init() {
	parseFlags()

	if options.showVersion {
		server.ShowVersionAndExit()
	}

	server.SetupLogging(options.logFile, options.logLevel, options.crashLogFile)
}

func main() {
	tcpServer = server.NewServer("pushd")
	tcpServer.LoadConfig(options.configFile)
	tcpServer.Launch()

	confPushd = new(ConfigPushd)
	confPushd.LoadConfig(tcpServer.Conf)
	tcpServer.LaunchTcpServ(confPushd.tcpListenAddr, NewProcessor(), confPushd.servPingInterval)

	shutdown()
}

func shutdown() {
	tcpServer.StopTcpServ()
	log.Info("Terminated")
	os.Exit(0)
}
