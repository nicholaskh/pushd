package main

import (
	"flag"
)

var (
	options struct {
		concurrency  int
		requests     int
		addr         string
		logFile      string
		logLevel     string
		crashLogFile string
		showVersion  bool
	}
)

func parseFlags() {
	flag.IntVar(&options.concurrency, "c", 10000, "connection concurrency")
	flag.IntVar(&options.requests, "n", 30, "how many requests one connection perform")
	flag.StringVar(&options.addr, "h", "127.0.0.1:2222", "which server to benchmark")
	flag.BoolVar(&options.showVersion, "v", false, "show version and exit")
	flag.StringVar(&options.logFile, "log", "stdout", "log file")
	flag.StringVar(&options.logLevel, "level", "info", "log level")
	flag.StringVar(&options.crashLogFile, "crashlog", "panic.dump", "crash log file")

	flag.Parse()

}
