package main

import (
	"flag"
)

var (
	options struct {
		addr string
	}
)

func parseFlags() {
	flag.StringVar(&options.addr, "a", "127.0.0.1:2222", "server address")

	flag.Parse()
}
