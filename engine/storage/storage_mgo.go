package storage

import (
	"errors"

	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/db"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type MongoStorage struct {
}

func newMongodbDriver() (this *MongoStorage) {
	this = new(MongoStorage)
	return
}

func (this *MongoStorage) store(mt *MsgTuple) error {
	err := db.MgoSession().DB("pushd").C("msg_log").Insert(bson.M{"channel": mt.Channel, "msg": mt.Msg, "ts": mt.Ts, "msgid": mt.MsgId, "uuid": mt.Uuid})

	return err
}

func (this *MongoStorage) storeMulti(mts []*MsgTuple) error {
	records := make([]interface{}, 0)
	for _, mt := range mts {
		records = append(records, bson.M{"channel": mt.Channel, "msg": mt.Msg, "ts": mt.Ts, "msgid": mt.MsgId, "uuid": mt.Uuid})
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

func (this *MongoStorage) bindUuidToChannel(channelName, channelId string, uuids ...string) error {
	if channelId == "" {
		return errors.New("channelId is empty")
	}
	if len(uuids) == 0 {
		return errors.New("uuids are empty")
	}

	var err error
	if channelName == "" {
		err = db.MgoSession().DB("pushd").C("channel_uuids").
			Update(bson.M{"_id": channelId}, bson.M{"$addToSet": bson.M{"uuids": bson.M{"$each": uuids}}})
	} else {
		err = db.MgoSession().DB("pushd").C("channel_uuids").
			Insert(bson.M{"_id": channelId, "channel": channelName, "uuids": uuids})
	}

	// bind channelId to uuid too
	if err == nil {
		bulk := db.MgoSession().DB("pushd").C("uuid_channels").Bulk()
		var documents []interface{}
		for _, v := range uuids {
			documents = append(documents, bson.M{"_id": v})
			documents = append(documents, bson.M{"$addToSet": bson.M{"channels": channelId}})
		}
		bulk.Upsert(documents...)
		_, err = bulk.Run()
	}
	return err
}

func (this *MongoStorage) rmUuidFromChannel(channelId string, uuids ...string) error {
	if channelId == "" {
		return errors.New("channel is empty")
	}
	if len(uuids) == 0 {
		return errors.New("uuids is empty")
	}

	err := db.MgoSession().DB("pushd").C("channel_uuids").
		Update(bson.M{"_id": channelId}, bson.M{"$pullAll": bson.M{"uuids": uuids}})

	var emptyUuids []string
	// delete mutil chat room if has no people
	db.MgoSession().DB("pushd").C("channel_uuids").Remove(bson.M{"_id":channelId, "uuids": emptyUuids})

	// rm channelId from uuid too
	if err == nil {
		bulk := db.MgoSession().DB("pushd").C("uuid_channels").Bulk()
		var documents []interface{}
		for _, v := range uuids {
			documents = append(documents, bson.M{"_id": v})
			documents = append(documents, bson.M{"$pull": bson.M{"channels": channelId}})
		}
		bulk.Update(documents...)
		_, err = bulk.Run()
	}

	return err
}
