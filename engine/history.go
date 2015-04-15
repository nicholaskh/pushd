package engine

import (
	"github.com/nicholaskh/pushd/db"
	"github.com/nicholaskh/pushd/engine/storage"
	"labix.org/v2/mgo/bson"
)

//get from cache
func history(channel string, ts int64) (result []interface{}, err error) {
	result = storage.MsgCache.GetRange(channel, ts)

	return
}

//fetch from db
func fullHistory(channel string, ts int64) (result []interface{}, err error) {
	c := db.MgoSession().DB("pushd").C("msg_log")

	noId := bson.M{"_id": 0}

	if channel == "" {
		err = c.Find(bson.M{"ts": bson.M{"$gte": ts}}).Select(noId).All(&result)
	} else {
		err = c.Find(bson.M{"ts": bson.M{"$gte": ts}, "channel": channel}).Select(noId).All(&result)
	}

	if len(result) == 0 {
		return []interface{}{}, nil
	}

	return
}
