package engine

import (
	"github.com/nicholaskh/pushd/engine/storage"
)

//get from cache
func history(channel string, ts int64) (result []interface{}, err error) {
	result = storage.MsgCache.GetRange(channel, ts)

	return
}

//fetch from db
func fullHistory(channel string, ts int64) (result []interface{}, err error) {
	result, err = storage.FetchHistory(channel, ts)

	return
}
