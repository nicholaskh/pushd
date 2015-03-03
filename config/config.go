package config

import (
	conf "github.com/nicholaskh/jsconf"
	"time"
)

var (
	PushdConf *ConfigPushd
)

type ConfigPushd struct {
	TcpListenAddr  string
	SessionTimeout time.Duration

	S2sAddr         string
	S2sPingInterval time.Duration
	Servers         []string
}

func (this *ConfigPushd) LoadConfig(cf *conf.Conf) {
	this.TcpListenAddr = cf.String("tcp_listen_addr", ":2222")
	this.SessionTimeout = cf.Duration("session_timeout", time.Minute*2)

	this.S2sAddr = cf.String("s2s_addr", ":2223")
	this.S2sPingInterval = cf.Duration("s2s_ping_interval", time.Minute*2)
	this.Servers = cf.StringList("servers", []string{this.TcpListenAddr})
}
