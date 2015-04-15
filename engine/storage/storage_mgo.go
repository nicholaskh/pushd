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

func (this *MongoStorage) store(mt *msgTuple) error {
	err := db.MgoSession().DB("pushd").C("msg_log").Insert(bson.M{"channel": mt.channel, "msg": mt.msg, "ts": mt.ts})

	return err
}

func (this *MongoStorage) storeMulti(mts []*msgTuple) error {
	records := make([]interface{}, 0)
	for _, mt := range mts {
		records = append(records, bson.M{"channel": mt.channel, "msg": mt.msg, "ts": mt.ts})
	}
	err := db.MgoSession().DB("pushd").C("msg_log").Insert(records...)

	return err
}
