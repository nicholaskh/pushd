package config

import (
	conf "github.com/nicholaskh/jsconf"
	"time"
)

var (
	PushdConf *ConfigPushd
)

type ConfigPushd struct {
	TcpListenAddr string
	ConnTimeout   time.Duration

	S2sAddr        string
	S2sConnTimeout time.Duration
	Servers        []string

	MetricsLogfile      string
	StatsOutputInterval time.Duration
}

func (this *ConfigPushd) LoadConfig(cf *conf.Conf) {
	this.TcpListenAddr = cf.String("tcp_listen_addr", ":2222")
	this.ConnTimeout = cf.Duration("conn_timeout", time.Minute*2)

	this.S2sAddr = cf.String("s2s_addr", ":2223")
	this.S2sConnTimeout = cf.Duration("s2s_conn_timeout", time.Minute*2)
	this.Servers = cf.StringList("servers", []string{this.TcpListenAddr})

	this.MetricsLogfile = cf.String("metrics_logfile", "metrics.log")
	this.StatsOutputInterval = cf.Duration("stats_output_interval", time.Minute*10)
}
