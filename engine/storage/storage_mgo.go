package storage

import (
	"github.com/nicholaskh/pushd/db"
	"labix.org/v2/mgo/bson"
)

type MongoStorage struct {
}

func newMongodbDriver() (this *MongoStorage) {
	this = new(MongoStorage)
	return
}

func (this *MongoStorage) store(mt *MsgTuple) error {
	err := db.MgoSession().DB("pushd").C("msg_log").Insert(bson.M{"channel": mt.Channel, "msg": mt.Msg, "ts": mt.Ts})

	return err
}

func (this *MongoStorage) storeMulti(mts []*MsgTuple) error {
	records := make([]interface{}, 0)
	for _, mt := range mts {
		records = append(records, bson.M{"channel": mt.Channel, "msg": mt.Msg, "ts": mt.Ts})
	}
	err := db.MgoSession().DB("pushd").C("msg_log").Insert(records...)

	return err
}
