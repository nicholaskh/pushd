package engine

import (
	"errors"
	"time"

	"github.com/nicholaskh/golib/cache"
	"github.com/nicholaskh/golib/str"
	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/db"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"crypto/sha256"
	"encoding/hex"
)

var (
	loginUsers *cache.LruCache = cache.NewLruCache(200000) //username => 1
)

//Auth for client
func authClient(token string) (string, error) {
	var result interface{}
	db.MgoSession().SetMode(mgo.Monotonic, true)
	_, err := db.MgoSession().DB("pushd").C("token").Find(bson.M{"tk": token}).Select(bson.M{"_id": 0, "tk": 1}).Apply(mgo.Change{Remove: true}, &result)
	if err == mgo.ErrNotFound {
		return "", errors.New("Client auth fail")
	} else if err != nil {
		log.Error("get token from db error: %s", err.Error())
		return "", errors.New("Client auth fail")
	}

	return "Client auth succeed", nil
}

func authServer(appkey string) (string, error) {
	c := db.MgoSession().DB("pushd").C("appkey")

	var result interface{}
	err := c.Find(bson.M{"appkey": appkey}).One(&result)
	if err == mgo.ErrNotFound {
		return "", errors.New("Server auth fail")
	} else if err != nil {
		log.Error("Error occured when query mongodb: %s", err.Error())
	}

	key := result.(bson.M)["appkey"]
	if key == appkey {
		return "Server auth succeed", nil
	}

	return "", errors.New("Server auth fail")
}

// TODO more secure token generator
func getToken() (token string) {
	token = str.Rand(32)

	err := db.MgoSession().DB("pushd").C("token").Insert(bson.M{"tk": token, "expire": time.Now()})
	if err != nil {
		log.Error("generate token error: %s", err.Error())
		return ""
	}
	return
}

func getAppKey() string {
	hash := sha256.New()
	hash.Write([]byte(str.Rand(32)))
	appkey := hex.EncodeToString(hash.Sum(nil))
	err := db.MgoSession().DB("pushd").C("appkey").Insert(bson.M{"appkey": appkey, "creat_time": time.Now()})
	if err != nil {
		return ""
	}
	return appkey
}
