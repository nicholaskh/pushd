package storage

import (
	"time"

	"gopkg.in/mgo.v2/bson"
	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/config"
	"github.com/nicholaskh/pushd/db"
)

type MsgTuple struct {
	Channel string `json:"channel"`
	Msg     string `json:"msg"`
	Ts      int64  `json:"ts"`
	Uuid	string `json:"uuid"`
	MsgId   int64  `json:"msgid"`
}

type ChanUuidsTuple struct {
	ChannelId string
	ChannelName string
	Uuids []string
	IsDel bool
}

type storageDriver interface {
	store(*MsgTuple) error
	storeMulti([]*MsgTuple) error
	fetchByChannelAndTs(channel string, ts int64) (result []interface{}, err error)
	bindUuidToChannel(channelName, channelId string, uuids ...string) error
	rmUuidFromChannel(channelId string, uuids ...string) error
}

var (
	msgQueue    chan *MsgTuple
	driver      storageDriver
	writeBuffer chan *MsgTuple
	chanUuidsQueue chan *ChanUuidsTuple
)

func Init() {
	msgQueue = make(chan *MsgTuple, config.PushdConf.MaxStorageOutstandingMsg)
	chanUuidsQueue = make(chan *ChanUuidsTuple)
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

	go func() {
		for {
			select {
			case cu := <-chanUuidsQueue:
				if cu.IsDel {
					driver.rmUuidFromChannel(cu.ChannelId, cu.Uuids...)
				} else {
					driver.bindUuidToChannel(cu.ChannelName, cu.ChannelId, cu.Uuids...)
				}
			}
		}
	}()

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

func EnqueueMsg(channel, msg , uuid string, ts, msgId int64) {
	msgQueue <- &MsgTuple{channel, msg, ts, uuid, msgId}
}

func FetchHistory(channel string, ts int64) (result []interface{}, err error) {
	return driver.fetchByChannelAndTs(channel, ts)
}

func EnqueueChanUuids(channelName, channelId string, isDel bool, uuids []string) {
	chanUuidsQueue <- &ChanUuidsTuple{channelId, channelName, uuids, isDel}
}
func FetchUuidsAboutChannel(channelId string) []string{

	var result interface{}
	err := db.MgoSession().DB("pushd").C("channel_uuids").
		Find(bson.M{"_id": channelId}).
		Select(bson.M{"uuids":1, "_id":0}).
		One(&result)

	if err == nil {
		uuids := result.(bson.M)["uuids"].([]interface{})
		res := make([]string, 0, len(uuids))
		for _, uuid := range uuids {
			res = append(res, uuid.(string))
		}
		return res
	}

	return nil

}