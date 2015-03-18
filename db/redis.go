package db

import (
	"fmt"
	"github.com/nicholaskh/pushd/config"
	"github.com/nicholaskh/redigo/redis"
)

var (
	redisConn redis.Conn
)

func RedisConn() redis.Conn {
	var err error
	if redisConn == nil {
		redisConn, err = redis.DialTimeout("tcp", config.PushdConf.Redis.Addr, config.PushdConf.Redis.ConnTimeout,
			config.PushdConf.Redis.ReadTimeout, config.PushdConf.Redis.WriteTimeout)
		if err != nil {
			panic(fmt.Sprintf("Connect to redis error: %s", err.Error()))
		}
	}
	return redisConn
}
