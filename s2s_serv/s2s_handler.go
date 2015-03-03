package s2s_serv

import (
	"github.com/nicholaskh/golib/set"
)

func subscribe() {
	//s2s
	_, exists = Proxy.getPeersByChannel(channel)
	if !exists {
		s2sProxy.RegisterChannel(channel)
	}
}

func publish() {
	//s2s
	var peers set.Set
	peers, exists = s2sProxy.getPeersByChannel(channel)
	if exists {
		s2sProxy.Publish(peers, channel, msg)
	}
}
