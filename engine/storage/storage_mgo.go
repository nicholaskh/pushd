package storage

import (
	"errors"

	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/db"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

type MongoStorage struct {
}

func newMongodbDriver() (this *MongoStorage) {
	this = new(MongoStorage)
	return
}

func (this *MongoStorage) store(mt *MsgTuple) error {
	err := db.MgoSession().DB("pushd").C("msg_log").Insert(bson.M{"channel": mt.Channel, "msg": mt.Msg, "ts": mt.Ts, "uuid": mt.Uuid})

	return err
}

func (this *MongoStorage) storeMulti(mts []*MsgTuple) error {
	records := make([]interface{}, 0)
	for _, mt := range mts {
		records = append(records, bson.M{"channel": mt.Channel, "msg": mt.Msg, "ts": mt.Ts, "uuid": mt.Uuid})
	}
	err := db.MgoSession().DB("pushd").C("msg_log").Insert(records...)

	return err
}

func (this *MongoStorage) fetchByChannelAndTs(channel string, ts int64) (result []interface{}, err error) {
	if channel == "" {
		return nil, errors.New("No channel specified")
	}
	err = db.MgoSession().DB("pushd").C("msg_log").Find(bson.M{"ts": bson.M{"$gte": ts}, "channel": channel}).Select(bson.M{"_id": 0}).All(&result)
	if err != nil && err != mgo.ErrNotFound {
		log.Error("fetch messages log from db error: %s", err.Error())
	}

	if len(result) == 0 {
		return []interface{}{}, nil
	}

	return
}

func (this *MongoStorage) bindUuidToChannel(channel string, uuids ...string) error {
	if channel == "" {
		return errors.New("channel is empty")
	}
	if len(uuids) == 0 {
		return errors.New("uuids are empty")
	}

	_, err := db.MgoSession().DB("pushd").C("channel_uuids").
		Upsert(bson.M{"channel": channel}, bson.M{"$addToSet": bson.M{"uuids": bson.M{"$each": uuids}}})

	return err
}

func (this *MongoStorage) rmUuidFromChannel(channel string, uuids ...string) error {
	if channel == "" {
		return errors.New("channel is empty")
	}
	if len(uuids) == 0 {
		return errors.New("uuids is empty")
	}

	return db.MgoSession().DB("pushd").C("channel_uuids").Update(bson.M{"channel": channel}, bson.M{"$pullAll": bson.M{"uuids": uuids}})
}
