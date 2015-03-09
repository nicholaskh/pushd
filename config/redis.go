package config

import (
	conf "github.com/nicholaskh/jsconf"
	"time"
)

type ConfigRedis struct {
	Addr         string
	ConnTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func (this *ConfigRedis) LoadConfig(cf *conf.Conf) {
	this.Addr = cf.String("addr", ":6379")
	this.ConnTimeout = cf.Duration("conn_timeout", time.Second*5)
	this.ReadTimeout = cf.Duration("read_timeout", time.Second*5)
	this.WriteTimeout = cf.Duration("write_timeout", time.Second*5)
}
