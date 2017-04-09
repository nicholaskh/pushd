package engine


import (
	"github.com/nicholaskh/golib/concurrent/map"
)

var (
	UuidToClient *UuidClientMap
)

type UuidClientMap struct {
	uuid_to_client cmap.ConcurrentMap
}

func NewUuidClientMap()(this *UuidClientMap){
	this = new(UuidClientMap)
	this.uuid_to_client = cmap.New()
	return
}

func (this *UuidClientMap)AddClient(uuid string, client *Client){
	_, exists := this.uuid_to_client.Get(uuid)
	if exists {
		return
	}
	this.uuid_to_client.Set(uuid, client)
}

func (this *UuidClientMap)GetClient(uuid string)(client *Client, exists bool){
	temp, exists := this.uuid_to_client.Get(uuid)
	if exists {
		client = temp.(*Client)
	}
	return
}

func (this *UuidClientMap)Remove(uuid string){
	this.uuid_to_client.Remove(uuid)
}
