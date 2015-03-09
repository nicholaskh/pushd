package engine

import (
	"errors"
	"fmt"

	"github.com/nicholaskh/golib/cache"
	//log "github.com/nicholaskh/log4go"
)

var (
	tokenPool  *cache.LruCache = cache.NewLruCache(200000) //token => 1
	loginUsers *cache.LruCache = cache.NewLruCache(200000) //username => 1
)

//Auth for client
//TODO maybe sasl is more secure
func authClient(token string) (string, error) {
	if _, exists := tokenPool.Get(token); exists {
		tokenPool.Del(token)
		return fmt.Sprintf("Auth succeed"), nil
	} else {
		return "", errors.New("Client auth fail")
	}
}

func authServer(appId, secretKey string) (string, error) {
	conn := RedisConn()
	resInterface, err := conn.Do("Get", fmt.Sprintf("appId:%s", appId))
	if err != nil {
		panic("Get secret key from redis error")
	}
	resByte, _ := resInterface.([]byte)
	key := string(resByte)
	if key == secretKey {
		return fmt.Sprintf("Auth succeed"), nil
	}
	return "", errors.New("Server auth fail")

}
