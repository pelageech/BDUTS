package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

type Metrics struct {
	CPU               prometheus.Gauge
	RequestsNow       prometheus.Gauge
	RequestsPerSecond prometheus.Gauge
}

var reg *prometheus.Registry
var GlobalMetrics *Metrics

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		CPU: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "cpu_usage",
		}),
		RequestsNow: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "requests_are_being_processed",
			Help: "How many are being processed on the backends summary",
		}),
		RequestsPerSecond: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "requests_per_second",
			Help: "An average value of requests are processed per second",
		}),
	}
	reg.MustRegister(m.CPU, m.RequestsNow, m.RequestsPerSecond)
	return m
}

func Init() {
	reg = prometheus.NewRegistry()
	GlobalMetrics = NewMetrics(reg)
}

func Handler() http.Handler {
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg})
}
