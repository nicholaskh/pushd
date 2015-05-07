package engine

import (
	"errors"

	"github.com/nicholaskh/golib/cache"
	"github.com/nicholaskh/golib/str"
	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/db"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

var (
	loginUsers *cache.LruCache = cache.NewLruCache(200000) //username => 1
)

//Auth for client
func authClient(token string) (string, error) {
	var result interface{}
	db.MgoSession().SetMode(mgo.Monotonic, true)
	_, err := db.MgoSession().DB("pushd").C("token").Find(bson.M{"tk": token}).Apply(mgo.Change{Remove: true}, &result)
	if err == mgo.ErrNotFound {
		return "", errors.New("Client auth fail")
	} else if err != nil {
		log.Error("get token from db error: %s", err.Error())
		return "", errors.New("Client auth fail")
	}

	return "Client auth succeed", nil
}

func authServer(appId, secretKey string) (string, error) {
	c := db.MgoSession().DB("pushd").C("user")

	var result interface{}
	err := c.Find(bson.M{"appId": "test_app"}).One(&result)
	if err != nil {
		log.Error("Error occured when query mongodb: %s", err.Error())
	}

	key := result.(bson.M)["secretKey"]
	if key == secretKey {
		return "Server auth succeed", nil
	}

	return "", errors.New("Server auth fail")
}

// TODO more secure token generator
func getToken() (token string) {
	token = str.Rand(32)
	err := db.MgoSession().DB("pushd").C("token").Insert(bson.M{"tk": token})
	if err != nil {
		log.Error("generate token error: %s", err.Error())
		return ""
	}
	return
}
