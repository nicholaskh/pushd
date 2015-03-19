package engine

import (
	"encoding/json"

	"github.com/nicholaskh/pushd/db"
	"labix.org/v2/mgo/bson"
)

func history(ts int64, channel string) (ret string, err error) {
	c := db.MgoSession().DB("pushd").C("msg_log")

	var result []interface{}

	noId := bson.M{"_id": 0}

	if channel == "" {
		err = c.Find(bson.M{"ts": bson.M{"$gte": ts}}).Select(noId).All(&result)
	} else {
		err = c.Find(bson.M{"ts": bson.M{"$gte": ts}, "channel": channel}).Select(noId).All(&result)
	}

	var retBytes []byte
	retBytes, err = json.Marshal(result)

	ret = string(retBytes)

	return
}
