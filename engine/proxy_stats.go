package engine

import "github.com/nicholaskh/metrics"

type ProxyStats struct {
	pubCalls   metrics.Meter
	subCalls   metrics.Meter
	unsubCalls metrics.Meter

	outChannels metrics.Meter
}

func newProxyStats() (this *ProxyStats) {
	this = new(ProxyStats)
	this.pubCalls = metrics.NewMeter()
	this.subCalls = metrics.NewMeter()
	this.unsubCalls = metrics.NewMeter()
	this.outChannels = metrics.NewMeter()

	return
}

func (this *ProxyStats) registerMetrics() {
	this.pubCalls = metrics.NewMeter()
	metrics.Register("proxy.pub_calls", this.pubCalls)

	this.subCalls = metrics.NewMeter()
	metrics.Register("proxy.sub_calls", this.subCalls)

	this.unsubCalls = metrics.NewMeter()
	metrics.Register("proxy.unsub_calls", this.unsubCalls)

	this.outChannels = metrics.NewMeter()
	metrics.Register("proxy.out_channels", this.outChannels)
}
