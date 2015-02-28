package main

import (
	"os"

	"github.com/nicholaskh/golib/server"
	log "github.com/nicholaskh/log4go"
)

func init() {
	parseFlags()

	if options.showVersion {
		server.ShowVersionAndExit()
	}

	server.SetupLogging(options.logFile, options.logLevel, options.crashLogFile)
}

func main() {
	s := server.NewServer("pushd")
	s.LoadConfig(options.configFile)
	s.Launch()
	s.LaunchTcpServ(NewProcessor())

	shutdown()
}

func shutdown() {
	log.Info("Terminated")
	os.Exit(0)
}
