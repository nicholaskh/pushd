package main

import (
	"fmt"
)

func init() {
	parseFlags()

	server.SetupLogging(options.logFile, options.logLevel, options.crashLogFile)
}

func main() {

	signal.RegisterSignalHandler(syscall.SIGINT, func(sig os.Signal) {
		shutdown()
	})

	shutdown()
}

func shutdown() {
	pushdServ.StopTcpServ()
	log.Info("Terminated")
	os.Exit(0)
}
