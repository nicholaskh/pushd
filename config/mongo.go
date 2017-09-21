package config

import (
	conf "github.com/nicholaskh/jsconf"
	"time"
)

type ConfigMongo struct {
	Addr             string
	Database 	 string
	UserName	 string
	Password	 string
	ConnTimeout      time.Duration
	OperationTimeout time.Duration
	SyncTimeout      time.Duration
}

func (this *ConfigMongo) LoadConfig(cf *conf.Conf) {
	this.Addr = cf.String("addr", ":27017")
	this.Database = cf.String("database", "")
	this.UserName = cf.String("username", "")
	this.Password = cf.String("password", "")
	if this.Database == "" || this.UserName == "" || this.Password == "" {
		panic("database、username or password in pushd.conf is empty")
	}
	this.ConnTimeout = cf.Duration("conn_timeout", time.Second*5)
	this.OperationTimeout = cf.Duration("operation_timeout", time.Second*5)
	this.SyncTimeout = cf.Duration("sync_timeout", time.Second*5)
}
