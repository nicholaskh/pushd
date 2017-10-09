package engine

import (
	"errors"
	"time"

	"github.com/nicholaskh/golib/cache"
	"github.com/nicholaskh/golib/str"
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
	err := db.MgoSession().DB("pushd").C("client_token").
		Find(bson.M{"tk": token}).Select(bson.M{"_id": 0, "tk": 1}).One(&result)

	if err != nil || result == nil{
		return "", errors.New("Client auth fail")
	}

	return "Client auth succeed", nil
}

func checkServerToken(token string) bool {
	var result interface{}
	err := db.MgoSession().DB("pushd").C("server_token").
		Find(bson.M{"tk": token}).Select(bson.M{"_id": 0, "expire": 1}).One(&result)
	if err != nil || result == nil {
		return false;
	}

	expire := result.(bson.M)["expire"].(time.Time)
	if time.Now().Unix() - expire.Unix() > 7200 {
		db.MgoSession().DB("pushd").C("server_token").Remove(bson.M{"tk": token})
		return false;
	}

	return true
}

func checkClientToken(client *Client, tokenChecking string) error {
	if client.tokenInfo.token == ""{
		return errors.New("Token Illegal")
	}

	if client.tokenInfo.token != tokenChecking {
		db.MgoSession().DB("pushd").C("client_token").Remove(bson.M{"uuid": client.uuid})
		return errors.New("Token Illegal")
	}

	if time.Now().Unix() - client.tokenInfo.expire > 36000000000 {
		db.MgoSession().DB("pushd").C("client_token").Remove(bson.M{"uuid": client.uuid})
		return errors.New("Token expired")
	}

	client.updateTokenExpire(time.Now().Unix())
	return nil
}

func authServer(appkey string) (string, error) {
	var result interface{}
	err := db.MgoSession().DB("pushd").C("appkey").Find(bson.M{"appkey": appkey}).One(&result)
	if err != nil || result == nil{
		return "", errors.New("Server auth fail")
	}

	return getServerToken(), nil
}

// TODO more secure token generator
func getClientToken() (token string) {
	token = str.Rand(32)
	db.MgoSession().DB("pushd").C("client_token").Insert(bson.M{"uuid": "", "tk": token, "expire": time.Now()})
	return
}

func getServerToken() string {
	token := str.Rand(32)
	db.MgoSession().DB("pushd").C("server_token").Insert(bson.M{"tk": token, "expire": time.Now()})
	return token
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

