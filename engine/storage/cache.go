package storage

import (
	"github.com/nicholaskh/skiplist"
)

var (
	msgCache *Cache
)

type Cache struct {
	maxSize  int
	msgCache map[string]*skiplist.SkipList
}

func NewCache(maxSize int) *Cache {
	this := new(Cache)
	this.msgCache = make(map[string]*skiplist.SkipList)
	this.maxSize = maxSize

	return this
}

func (this *Cache) Store(mt *msgTuple) error {
	channelCache, exists := this.msgCache[mt.channel]
	if !exists {
		channelCache = skiplist.New(skiplist.Int64Desc)
		this.msgCache[mt.channel] = channelCache
	}

	channelCache.Set(mt.ts, mt)
	if channelCache.Len() >= this.maxSize {
		channelCache.Remove(channelCache.Front().Value.(*msgTuple).ts)
	}

	return nil
}

func (this *Cache) GetRange(channel string, ts int64) []interface{} {
	channelCache, exists := this.msgCache[channel]
	if !exists {
		return nil
	}
	return channelCache.GetValuesGreaterThan(ts)
}
