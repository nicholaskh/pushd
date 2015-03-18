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

	MsgStorage               string
	MaxStorageOutstandingMsg int

	Redis *ConfigRedis
	Mongo *ConfigMongo
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

	this.MsgStorage = cf.String("msg_storage", "")
	if this.MsgStorage != "" {
		this.MaxStorageOutstandingMsg = cf.Int("max_storage_outstanding_msg", 100)
	}

	this.Redis = new(ConfigRedis)
	section, err := cf.Section("redis")
	if err != nil {
		panic("Redis config not found")
	}
	this.Redis.LoadConfig(section)

	this.Mongo = new(ConfigMongo)
	section, err = cf.Section("mongodb")
	if err != nil {
		panic("Mongodb config not found")
	}
	this.Mongo.LoadConfig(section)
}

func (this *ConfigPushd) EnableStorage() bool {
	return this.MsgStorage != ""
}
