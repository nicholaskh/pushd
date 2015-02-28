package main

import (
	conf "github.com/nicholaskh/jsconf"
	"time"
)

type ConfigPushd struct {
	tcpListenAddr    string
	servPingInterval time.Duration
}

func (this *ConfigPushd) LoadConfig(cf *conf.Conf) {
	this.tcpListenAddr = cf.String("tcp_listen_addr", ":2222")
	this.servPingInterval = cf.Duration("serv_ping_interval", time.Minute*2)
}
