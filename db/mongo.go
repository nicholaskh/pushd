package db

import (
	"fmt"
	"github.com/nicholaskh/pushd/config"
	"gopkg.in/mgo.v2"
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
		database := mgoSession.DB(config.PushdConf.Mongo.Database)
		err = database.Login(config.PushdConf.Mongo.UserName, config.PushdConf.Mongo.Password)
		if err != nil {
			panic("database auth faild")
		}

	}

	return mgoSession
}


func UserCollection() *mgo.Collection {
	return mgoSession.DB("mongo").C("user")
}

func ChatRoomCollection() *mgo.Collection {
	return mgoSession.DB("mongo").C("chatroom")
}
