package storage

import (
	"github.com/nicholaskh/pushd/config"
)

type msgTuple struct {
	channel string
	msg     string
}

type storageDriver interface {
	store(*msgTuple) error
}

var (
	msgQueue chan *msgTuple
	driver   storageDriver
)

func Init() {
	msgQueue = make(chan *msgTuple, config.PushdConf.MaxStorageOutstandingMsg)
	driver = factory(config.PushdConf.MsgStorage)
}

func factory(driverType string) storageDriver {
	switch driverType {
	case "mongodb":
		return newMongodbDriver()

	default:
		return nil
	}
}

func Serv() {
	for {
		select {
		case mt := <-msgQueue:
			driver.store(mt)
		}
	}
}

func EnqueueMsg(channel, msg string) {
	msgQueue <- &msgTuple{channel, msg}
}
