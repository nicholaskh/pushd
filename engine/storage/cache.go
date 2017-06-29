package storage

import (
	"github.com/nicholaskh/skiplist"
)

var (
	MsgCache *Cache
)

type Cache struct {
	maxSize int
	list    map[string]*skiplist.SkipList
}

func NewCache(maxSize int) *Cache {
	this := new(Cache)
	this.list = make(map[string]*skiplist.SkipList)
	this.maxSize = maxSize

	return this
}

func (this *Cache) Store(mt *MsgTuple) error {
	channelCache, exists := this.list[mt.Channel]
	if !exists {
		channelCache = skiplist.New(skiplist.Int64Asc)
		this.list[mt.Channel] = channelCache
	}

	channelCache.Set(mt.Ts, mt)
	if channelCache.Len() >= this.maxSize {
		channelCache.Remove(channelCache.Front().Value.(*MsgTuple).Ts)
	}

	return nil
}

func (this *Cache) GetRange(channel string, ts int64) []interface{} {
	channelCache, exists := this.list[channel]
	if !exists {
		return []interface{}{}
	}
	return channelCache.GetValuesGreaterThan(ts)
}
