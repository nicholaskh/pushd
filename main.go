package main

import (
	"github.com/nicholaskh/golib/server"
)

func init() {
	parseFlags()
}

func main() {
	server.LaunchTCPServ(SERV_ADDR, NewProcessor())
}
