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
	session := db.MgoSession()
	c := session.DB("pushd").C("msg_log")
	c.Insert(bson.M{"channel": mt.channel, "msg": mt.msg})

	return nil
}
