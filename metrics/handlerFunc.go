package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shirou/gopsutil/cpu"
	"io"
	"log"
	"net/http"
	"runtime"
	"time"
)

const timeObserve = 1 * time.Second

type Metrics struct {
	CPU              prometheus.Gauge
	MaxMemory        prometheus.Gauge
	AllocatedMemory  prometheus.Gauge
	CacheSize        prometheus.Gauge
	CachePagesCount  prometheus.Gauge
	RequestsNow      prometheus.Gauge
	Requests         prometheus.Counter
	RequestsByCache  prometheus.Counter
	RequestBodySize  prometheus.Histogram
	ResponseBodySize prometheus.Histogram
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		CPU: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "bduts_cpu_usage",
			Help: "CPU usage",
		}),
		Requests: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "bduts_requests_were_processed",
			Help: "How many requests were processed on the backends summary",
		}),
		RequestsByCache: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "bduts_requests_by_cache_were_processed",
			Help: "How many requests were processed by the cache summary",
		}),
		RequestsNow: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "bduts_requests_are_being_processed",
			Help: "How many requests are being processed",
		}),
		AllocatedMemory: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "bduts_allocated_memory",
		}),
		CacheSize: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "bduts_cache_size_used",
		}),
		CachePagesCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "bduts_cache_pages_count",
			Help: "How many pages are stored in cache now?",
		}),
		RequestBodySize: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "request_body_size",
			Buckets: []float64{0.5, 0.9, 0.99},
		}),
		ResponseBodySize: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "response_body_size",
			Buckets: []float64{0.5, 0.9, 0.99},
		}),
	}
	reg.MustRegister(
		m.CPU,
		m.Requests,
		m.RequestsNow,
		m.AllocatedMemory,
		m.CacheSize,
		m.CachePagesCount,
		m.RequestsByCache,
		m.RequestBodySize,
		m.ResponseBodySize,
	)
	return m
}

var reg *prometheus.Registry
var GlobalMetrics *Metrics

func UpdateCPU() {
	p, err := cpu.Percent(0, false)
	if err == nil {
		GlobalMetrics.CPU.Set(p[0])
	}
}

func UpdateMemory() {
	m := runtime.MemStats{}
	runtime.ReadMemStats(&m)
	GlobalMetrics.AllocatedMemory.Set(float64(m.Alloc))
}

func UpdateCacheSize(size int64) {
	GlobalMetrics.CacheSize.Set(float64(size))
}

func UpdateCachePagesCount(delta int) {
	GlobalMetrics.CachePagesCount.Add(float64(delta))
}

func UpdateRequestBodySize(req *http.Request) {
	b, err := io.ReadAll(req.Body)
	if err != nil {
		return
	}
	GlobalMetrics.RequestBodySize.Observe(float64(len(b)))
}

func UpdateResponseBodySize(size float64) {
	GlobalMetrics.ResponseBodySize.Observe(size)
	log.Println("logged ", size)
}

func Init(initCacheSize int64, initPagesCount int) {
	reg = prometheus.NewRegistry()
	GlobalMetrics = NewMetrics(reg)
	go func() {
		t := time.NewTicker(timeObserve)
		for {
			<-t.C
			// cpu
			UpdateCPU()

			// memory
			UpdateMemory()
		}
	}()

	UpdateCacheSize(initCacheSize)
	UpdateCachePagesCount(initPagesCount)
}

func Handler() http.Handler {
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg})
}
