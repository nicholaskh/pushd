package storage

import (
	"time"

	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/config"
)

type MsgTuple struct {
	Channel string `json:"channel"`
	Msg     string `json:"msg"`
	Ts      int64  `json:"ts"`
}

type storageDriver interface {
	store(*MsgTuple) error
	storeMulti([]*MsgTuple) error
	fetchByChannelAndTs(channel string, ts int64) (result []interface{}, err error)
}

var (
	msgQueue    chan *MsgTuple
	driver      storageDriver
	writeBuffer chan *MsgTuple
)

func Init() {
	msgQueue = make(chan *MsgTuple, config.PushdConf.MaxStorageOutstandingMsg)
	driver = factory(config.PushdConf.MsgStorage)
	MsgCache = NewCache(config.PushdConf.MaxCacheMsgsEveryChannel)
	if config.PushdConf.MsgFlushPolicy != config.MSG_FLUSH_EVERY_TRX {
		writeBuffer = make(chan *MsgTuple, config.PushdConf.MsgStorageWriteBufferSize)
	}
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
	if config.PushdConf.MsgFlushPolicy == config.MSG_FLUSH_EVERY_SECOND {
		go func() {
			for {
				select {
				case <-time.Tick(time.Second):
					//get current buffer length
					cLen := len(writeBuffer)
					if cLen > 0 {
						records := make([]*MsgTuple, 0)
						for i := 0; i < cLen; i++ {
							if mt, ok := <-writeBuffer; ok {
								records = append(records, mt)
							} else {
								// Ghost appears...
								log.Error("Msg gone away...")
								break
							}
						}
						err := driver.storeMulti(records)
						if err != nil {
							log.Error("Multi store msg log error: %s", err.Error())
						}
					}
				}
			}
		}()
	}

	for {
		select {
		case mt := <-msgQueue:
			if config.PushdConf.MsgFlushPolicy == config.MSG_FLUSH_EVERY_TRX {
				err := driver.store(mt)
				if err != nil {
					log.Error("Store msg log error: %s", err.Error())
				}
			} else {
				writeBuffer <- mt
			}
		}
	}
}

func EnqueueMsg(channel, msg string, ts int64) {
	msgQueue <- &MsgTuple{channel, msg, ts}
}

func FetchHistory(channel string, ts int64) (result []interface{}, err error) {
	return driver.fetchByChannelAndTs(channel, ts)
}
