package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shirou/gopsutil/cpu"
	"net/http"
	"time"
)

const timeObserve = 1 * time.Second

type Metrics struct {
	CPU             prometheus.Gauge
	MaxMemory       prometheus.Gauge
	AllocatedMemory prometheus.Gauge
	DiskSpaceMax    prometheus.Gauge
	DiskSpaceUsed   prometheus.Gauge
	Status200       prometheus.Summary
	Status500       prometheus.Summary
	RequestsNow     prometheus.Gauge
	Requests        prometheus.Counter
	AliveBackends   prometheus.Counter
	AllBackends     prometheus.Gauge
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		CPU: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "bduts_cpu_usage",
		}),
		Requests: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "bduts_requests_were_processed",
			Help: "How many requests were processed on the backends summary",
		}),
		RequestsNow: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "bduts_requests_are_being_processed",
			Help: "How many requests are being processed",
		}),
	}
	reg.MustRegister(m.CPU, m.Requests, m.RequestsNow)
	return m
}

var reg *prometheus.Registry
var GlobalMetrics *Metrics

func Init() {
	reg = prometheus.NewRegistry()
	GlobalMetrics = NewMetrics(reg)
	go func() {
		t := time.NewTicker(timeObserve)
		for {
			select {
			case <-t.C:
				p, _ := cpu.Percent(0, false)
				GlobalMetrics.CPU.Set(p[0])
			}
		}
	}()
}

func Handler() http.Handler {
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg})
}
