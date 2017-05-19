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
	"github.com/nicholaskh/golib/concurrent/map"
)

var (
	loginUsers *cache.LruCache = cache.NewLruCache(200000) //username => 1
	uuidTokenMap *UuidTokenMap
)

func InitAuth(){
	uuidTokenMap = newUuidTokenMap()
}

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

func checkClientToken(uuid, tokenChecking string) error {
	token, expire, exists := uuidTokenMap.getTokenInfo(uuid)
	if !exists || token != tokenChecking{
		uuidTokenMap.rmTokenInfo(uuid)
		return errors.New("Token Illegal")
	}

	if time.Now().Unix() - expire > 7200 {
		uuidTokenMap.rmTokenInfo(uuid)
		return errors.New("Token expired")
	}

	uuidTokenMap.updateExpire(uuid, time.Now().Unix())
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

type tokenInfo struct {
	token string
	expire int64
}

type UuidTokenMap struct {
	uuidToToken cmap.ConcurrentMap
}

func newUuidTokenMap() (this *UuidTokenMap) {
	this = new(UuidTokenMap)
	this.uuidToToken = cmap.New()
	return
}

func (this *UuidTokenMap) getTokenInfo(uuid string) (token string, expire int64, exists bool) {
	info, exists := this.uuidToToken.Get(uuid)
	if exists {
		tnfo := info.(*tokenInfo)
		token = tnfo.token
		expire = tnfo.expire
	}

	return
}

func (this *UuidTokenMap) updateExpire(uuid string, expire int64) {
	info, exists := this.uuidToToken.Get(uuid)
	if exists {
		info.(*tokenInfo).expire = expire
		db.MgoSession().DB("pushd").C("client_token").UpdateId(uuid, bson.M{"expire": expire})
	}
}

func (this *UuidTokenMap) setTokenInfo(uuid string, token string, expire int64) error {
	info := new(tokenInfo)
	info.token = token
	info.expire = expire
	this.uuidToToken.Set(uuid, info)

	db.MgoSession().DB("pushd").C("client_token").RemoveAll(bson.M{"uuid": uuid})

	return db.MgoSession().DB("pushd").C("client_token").
		Update(bson.M{"tk": token}, bson.M{"$set": bson.M{"uuid": uuid, "expire": expire}})

}

func (this *UuidTokenMap) rmTokenInfo(uuid string) bool {

	exists := this.uuidToToken.Has(uuid)
	if !exists {
		return false
	}
	this.uuidToToken.Remove(uuid)
	err := db.MgoSession().DB("pushd").C("client_token").Remove(bson.M{"uuid": uuid})
	if err == nil {
		return true
	}
	return false
}
