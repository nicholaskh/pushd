package engine

import (
	"github.com/nicholaskh/pushd/db"
	"labix.org/v2/mgo/bson"
)

// TODO cache

func history(ts int64, channel string) (result []interface{}, err error) {
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
