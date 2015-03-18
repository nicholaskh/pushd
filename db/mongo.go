package db

import (
	"fmt"
	"github.com/nicholaskh/pushd/config"
	"labix.org/v2/mgo"
)

var (
	mgoSession *mgo.Session
)

func MgoSession() *mgo.Session {
	if mgoSession == nil {
		var err error
		mgoSession, err = mgo.Dial(config.PushdConf.Mongo.Addr)
		if err != nil {
			panic(fmt.Sprintf("Connect to mongo error: %s", err.Error()))
		}
	}
	return mgoSession
}
