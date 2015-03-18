package config

import (
	conf "github.com/nicholaskh/jsconf"
	"time"
)

type ConfigMongo struct {
	Addr             string
	ConnTimeout      time.Duration
	OperationTimeout time.Duration
	SyncTimeout      time.Duration
}

func (this *ConfigMongo) LoadConfig(cf *conf.Conf) {
	this.Addr = cf.String("addr", ":27017")
	this.ConnTimeout = cf.Duration("conn_timeout", time.Second*5)
	this.OperationTimeout = cf.Duration("operation_timeout", time.Second*5)
	this.SyncTimeout = cf.Duration("sync_timeout", time.Second*5)
}
