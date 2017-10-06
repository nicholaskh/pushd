package offpush

import (
	"sync/atomic"
	"errors"
	"jpushclient"
	log "github.com/nicholaskh/log4go"
	"fmt"
)

const (
	appKey = "3119b423cd31a4827d9a53cf"
        secret = "2da5c85573ff4e490d3d5d6f"
)

var (
	senderPool *OffSenderPool
)


type IOffSender interface {
	send(pushIds []string, message, ownerId string) error
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
	pushClient *jpushclient.PushClient
}

func newSender(address string) (*Sender, error) {
	sender := new(Sender)
	// TODO 检测是否是http长连接
	sender.pushClient = jpushclient.NewPushClient(secret, appKey)
	return sender, nil

}

func (this *Sender) send(pushIds []string, message, ownerId string) error {

	if len(pushIds) == 0 {
		return nil
	}

	//Platform
	pf := jpushclient.Platform{}
	pf.Add(jpushclient.ANDROID)

	//Audience
	ad := jpushclient.Audience{}
	ad.SetID(pushIds)

	//Notice
	var notice jpushclient.Notice
	notice.SetAlert("alert_test")
	notice.SetAndroidNotice(&jpushclient.AndroidNotice{Alert: "AndroidNotice"})

	var msg jpushclient.Message
	msg.Title = ""
	msg.Content = message

	payload := jpushclient.NewPushPayLoad()
	payload.SetPlatform(&pf)
	payload.SetAudience(&ad)
	payload.SetMessage(&msg)
	payload.SetNotice(&notice)

	bytes, _ := payload.ToBytes()

	log.Info(fmt.Sprintf("offpush: %s", string(bytes)))

	_, err := this.pushClient.Send(bytes)

	// TODO 处理pushId不合法的情况
	return err
}

func (this *Sender) close(){
	// TODO 实现
}

func initOffSenderPool(capacity int, addrs []string) error {
	if len(addrs) < 1 {
		return errors.New("地址为空")
	}

	senderPool = new(OffSenderPool)
	senderPool.size = int32(capacity)
	senderPool.baton = 0
	senderPool.pool = make([]IOffSender, capacity)

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
						sender.close()
					}
				}
			}
			return err
		}
		senderPool.pool[i] = t
	}

	return nil

}
