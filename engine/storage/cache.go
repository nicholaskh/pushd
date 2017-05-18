package storage

import (
	"github.com/nicholaskh/skiplist"
	"container/list"
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


var (
	MsgId *MsgIdCache
)

type MsgIdCache struct {
	maxSize int
	list    map[string]*list.List
}

func NewMsgIdCache() (*MsgIdCache) {
	this := new(MsgIdCache)
	this.list = make(map[string]*list.List)
	this.maxSize = 10

	return this
}

func (this *MsgIdCache) CheckAndSet(uuid string, msgId int64) bool {
	li, exists := this.list[uuid]
	if !exists {
		li = list.New()
		this.list[uuid] = li
		i := 10
		for ; i>0 ; {
			i--
			li.PushFront(nil)
		}
		li.PushFront(msgId)
		return false
	}

	for e := li.Front(); e != nil; e = e.Next() {
		if e.Value == nil {
			continue
		} else if e.Value.(int64) == msgId {
			return true
		}
	}

	li.PushFront(msgId)
	li.Remove(li.Back())
	return false

}