package engine

import (
	"github.com/nicholaskh/etclib"
	log "github.com/nicholaskh/log4go"
	"github.com/nicholaskh/pushd/config"
)

func RegisterEtc() error {
	err := etclib.Dial(config.PushdConf.EtcServers)
	if err != nil {
		return err
	}
	err = etclib.BootService(config.PushdConf.S2sListenAddr, etclib.SERVICE_PUSHD)
	return err
}

func watchPeers(proxy *S2sProxy) {
	eventChan := make(chan []string)
	go func() {
		for {
			select {
			case servers := <-eventChan:
				log.Debug(servers)
				for _, server := range servers {
					proxy.connectPeer(server)
				}
			}
		}
	}()
	etclib.WatchService(etclib.SERVICE_PUSHD, eventChan)
}

func UnregisterEtc() error {
	return etclib.ShutdownService(config.PushdConf.S2sListenAddr, etclib.SERVICE_PUSHD)
}
