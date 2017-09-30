package offpush

import (
	"sync/atomic"
	"github.com/go-redis/redis"
	"errors"
)

var (
	senderPool *OffSenderPool
)


type IOffSender interface {
	send(pushIds *[]string, message, ownerId string)
}

type OffSenderPool struct {
	pool []IOffSender
	baton int32
	size int32
}

func (this *OffSenderPool)ObtainOneOffSender() IOffSender {
	var now int32
	var hitLoc int32
	for ;; {
		now = atomic.LoadInt32(&this.baton)
		now = senderPool.baton

		hitLoc = (now+1) % this.size
		if atomic.CompareAndSwapInt32(&this.baton, now, hitLoc) {
			break
		}
	}
	return this.pool[hitLoc]
}

type Sender struct {
	address string
	redis redis.Client
}

func newSender(address string) (*Sender, error) {
	sender := new(Sender)
	sender.redis = redis.NewClient(&redis.Options{
		Addr: address,
		Password: "",
		DB: 0,
	})

	_, err := sender.redis.Ping().Result()
	if err != nil {
		return nil, err
	}
	return sender, nil

}

func (this *Sender) Send(pushIds *[]string, message, ownerId string) error {

	//TODO 实现发送逻辑
	return nil
}

func initOffSenderPool(capacity int32, addrs []string) error {
	if len(addrs) < 1 {
		return errors.New("地址为空")
	}

	senderPool = new(OffSenderPool)
	senderPool.size = capacity
	senderPool.baton = 0
	senderPool.pool = make([]*IOffSender, len(addrs))

	length := len(addrs)
	for i:= 0; i<capacity; i++ {
		address := addrs[i % length]
		t, err := newSender(address)
		if err != nil {
			// 关掉所有建立好的连接
			for _, temp := range senderPool.pool {
				if temp != nil {
					sender, ok := interface{}(temp).(Sender)
					if ok {
						sender.redis.Close()
					}
				}
			}
			return err
		}
		senderPool.pool[i] = t
	}

	return

}
