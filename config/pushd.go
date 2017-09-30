package config

import (
	"fmt"
	"strings"
	"time"

	conf "github.com/nicholaskh/jsconf"
)

const (
	MSG_FLUSH_EVERY_TRX = iota
	MSG_FLUSH_EVERY_SECOND

	S2S_PORT = 2223
)

var (
	PushdConf *ConfigPushd
)

type ConfigPushd struct {
	TcpListenAddr           string
	SessionTimeout          time.Duration
	ServInitialGoroutineNum int

	LongPollingListenAddr     string
	LongPollingSessionTimeout time.Duration

	StatsListenAddr string
	ProfListenAddr  string

	S2sListenAddr         string
	S2sSessionTimeout     time.Duration
	S2sIntialGoroutineNum int

	S2sChannelPeersMaxItems int

	EtcServers []string


	OffSenderPoolNum int
	MsgQueueAddrs []string

	MetricsLogfile      string
	StatsOutputInterval time.Duration

	PubsubChannelMaxItems int

	MsgStorage                string
	MaxStorageOutstandingMsg  int
	MsgFlushPolicy            int
	MsgStorageWriteBufferSize int
	MaxCacheMsgsEveryChannel  int

	Redis *ConfigRedis
	Mongo *ConfigMongo
}

func (this *ConfigPushd) LoadConfig(cf *conf.Conf) {
	this.TcpListenAddr = cf.String("tcp_listen_addr", ":2222")
	this.SessionTimeout = cf.Duration("session_timeout", time.Minute*2)
	this.ServInitialGoroutineNum = cf.Int("serv_initial_goroutine_num", 200)

	this.LongPollingListenAddr = cf.String("long_polling_listen_addr", "")
	if this.LongPollingListenAddr != "" {
		this.LongPollingSessionTimeout = cf.Duration("long_polling_session_timeout", time.Minute)
	}

	this.StatsListenAddr = cf.String("stats_listen_addr", ":9020")
	this.ProfListenAddr = cf.String("prof_listen_addr", ":9021")

	this.S2sListenAddr = GetS2sAddr(this.TcpListenAddr)
	this.S2sSessionTimeout = cf.Duration("s2s_conn_timeout", time.Minute*2)
	this.S2sIntialGoroutineNum = cf.Int("s2s_initial_goroutine_num", 8)

	this.S2sChannelPeersMaxItems = cf.Int("s2s_channel_peers_max_items", 200000)

	this.EtcServers = cf.StringList("etc_servers", nil)
	if len(this.EtcServers) == 0 {
		this.EtcServers = nil
	}

	this.MsgQueueAddrs = cf.StringList("message_queue_addrs", nil)
	if len(this.MsgQueueAddrs) == 0 {
		this.MsgQueueAddrs = nil
	}

	this.OffSenderPoolNum = cf.Int("offline_sender_pool_num", 1)

	this.MetricsLogfile = cf.String("metrics_logfile", "metrics.log")
	this.StatsOutputInterval = cf.Duration("stats_output_interval", time.Minute*10)

	this.PubsubChannelMaxItems = cf.Int("pubsub_channel_max_items", 200000)

	this.MsgStorage = cf.String("msg_storage", "")
	if this.MsgStorage != "" {
		this.MaxStorageOutstandingMsg = cf.Int("max_storage_outstanding_msg", 100)
		this.MsgFlushPolicy = cf.Int("msg_storage_flush_policy", MSG_FLUSH_EVERY_TRX)
		if this.MsgFlushPolicy != MSG_FLUSH_EVERY_TRX && this.MsgFlushPolicy != MSG_FLUSH_EVERY_SECOND {
			panic("invalid msg flush policy")
		}
		this.MsgStorageWriteBufferSize = cf.Int("msg_storage_write_buffer_size", 10000)
		this.MaxCacheMsgsEveryChannel = cf.Int("max_cache_msgs_every_channel", 3000)
	}

	this.Redis = new(ConfigRedis)
	section, err := cf.Section("redis")
	if err == nil {
		this.Redis.LoadConfig(section)
	}

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

func GetS2sAddr(servAddr string) (s2sAddr string) {
	parts := strings.Split(servAddr, ":")
	ip := parts[0]
	s2sPort := S2S_PORT
	s2sAddr = fmt.Sprintf("%s:%d", ip, s2sPort)
	return
}

func (this *ConfigPushd) IsDistMode() bool {
	return this.EtcServers != nil
}
