package storage

import (
	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/config"
)

type msgTuple struct {
	channel string
	msg     string
	ts      int64
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

// TODO - use flush mechanism
func Serv() {
	for {
		select {
		case mt := <-msgQueue:
			err := driver.store(mt)
			if err != nil {
				log.Error("Store msg log error: %s", err.Error())
			}
		}
	}
}

func EnqueueMsg(channel, msg string, ts int64) {
	msgQueue <- &msgTuple{channel, msg, ts}
}
