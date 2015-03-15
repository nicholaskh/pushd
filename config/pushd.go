package config

import (
	conf "github.com/nicholaskh/jsconf"
	"time"
)

var (
	PushdConf *ConfigPushd
)

type ConfigPushd struct {
	TcpListenAddr           string
	SessionTimeout          time.Duration
	ServInitialGoroutineNum int

	S2sAddr               string
	S2sSessionTimeout     time.Duration
	S2sIntialGoroutineNum int
	Servers               []string

	MetricsLogfile      string
	StatsOutputInterval time.Duration

	Redis *ConfigRedis
}

func (this *ConfigPushd) LoadConfig(cf *conf.Conf) {
	this.TcpListenAddr = cf.String("tcp_listen_addr", ":2222")
	this.SessionTimeout = cf.Duration("session_timeout", time.Minute*2)
	this.ServInitialGoroutineNum = cf.Int("serv_initial_goroutine_num", 200)

	this.S2sAddr = cf.String("s2s_addr", ":2223")
	this.S2sSessionTimeout = cf.Duration("s2s_conn_timeout", time.Minute*2)
	this.S2sIntialGoroutineNum = cf.Int("s2s_initial_goroutine_num", 8)
	this.Servers = cf.StringList("servers", []string{this.TcpListenAddr})

	this.MetricsLogfile = cf.String("metrics_logfile", "metrics.log")
	this.StatsOutputInterval = cf.Duration("stats_output_interval", time.Minute*10)

	this.Redis = new(ConfigRedis)
	section, err := cf.Section("redis")
	if err != nil {
		panic("Redis config not found")
	}
	this.Redis.LoadConfig(section)
}
