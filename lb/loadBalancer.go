package lb

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/pelageech/BDUTS/backend"
	"github.com/pelageech/BDUTS/cache"
	"github.com/pelageech/BDUTS/config"
	"github.com/pelageech/BDUTS/timer"
)

// LoadBalancerConfig is parse from `config.json` file.
// It contains all the necessary information of the load balancer.
type LoadBalancerConfig struct {
	port              int
	healthCheckPeriod time.Duration
}

func NewLoadBalancerConfig(port int, healthCheckPeriod time.Duration) *LoadBalancerConfig {
	return &LoadBalancerConfig{
		port:              port,
		healthCheckPeriod: healthCheckPeriod,
	}
}

func (c *LoadBalancerConfig) Port() int {
	return c.port
}

func (c *LoadBalancerConfig) HealthCheckPeriod() time.Duration {
	return c.healthCheckPeriod
}

// LoadBalancer is a struct that contains all the configuration
// of the load balancer.
type LoadBalancer struct {
	config          *LoadBalancerConfig
	pool            *backend.ServerPool
	cacheProps      *cache.CachingProperties
	healthCheckFunc func(*backend.Backend)
}

// NewLoadBalancer is the constructor of the load balancer
func NewLoadBalancer(
	config *LoadBalancerConfig,
	cachingProperties *cache.CachingProperties,
	healthChecker func(*backend.Backend),
) *LoadBalancer {
	setLogPrefixBDUTS()
	return &LoadBalancer{
		config:          config,
		pool:            backend.NewServerPool(),
		cacheProps:      cachingProperties,
		healthCheckFunc: healthChecker,
	}
}

func (lb *LoadBalancer) CacheProps() *cache.CachingProperties {
	return lb.cacheProps
}

func (lb *LoadBalancer) Config() *LoadBalancerConfig {
	return lb.config
}

func (lb *LoadBalancer) Pool() *backend.ServerPool {
	return lb.pool
}

func (lb *LoadBalancer) HealthCheckFunc() func(*backend.Backend) {
	return lb.healthCheckFunc
}

// HealthChecker periodically checks all the backends in balancer pool
func (lb *LoadBalancer) HealthChecker() {
	ticker := time.NewTicker(lb.config.healthCheckPeriod)

	for {
		<-ticker.C
		log.Println("Health Check has been started!")
		for _, server := range lb.Pool().Servers() {
			lb.healthCheckFunc(server)
		}
		log.Println("All the checks has been completed!")
	}
}

// ConfigureServerPool feels the balancer pool with servers
func (lb *LoadBalancer) ConfigureServerPool(servers []config.ServerConfig) {
	for _, server := range servers {
		log.Printf("%v", server)

		var b backend.Backend
		var err error

		b.URL, err = url.Parse(server.URL)
		if err != nil {
			log.Printf("Failed to parse server URL: %s\n", err)
			continue
		}

		b.HealthCheckTcpTimeout = time.Duration(server.HealthCheckTcpTimeout) * time.Millisecond
		b.Alive = false

		b.RequestChan = make(chan bool, server.MaximalRequests)

		lb.pool.AddServer(&b)
	}
}

// uses balancer db for taking the page from cache and writing it to http.ResponseWriter
// if such a page is in cache
func (lb *LoadBalancer) writePageIfIsInCache(rw http.ResponseWriter, req *http.Request) error {
	if lb.cacheProps == nil {
		return errors.New("cache properties weren't set")
	}

	log.Println("Try to get a response from cache...")

	cacheItem, err := lb.cacheProps.GetPageFromCache(req)
	if err != nil {
		return err
	}
	log.Println("Successfully got a response")

	for key, values := range cacheItem.Header {
		for _, value := range values {
			rw.Header().Add(key, value)
		}
	}

	_, err = rw.Write(cacheItem.Body)
	if err != nil {
		return err
	}
	log.Println("Transferred")

	return nil
}

// the balancer supports only HTTP 1.1 version because
// the backends use a common HTTP protocol
func isHTTPVersionSupported(req *http.Request) bool {
	if maj, min, ok := http.ParseHTTPVersion(req.Proto); ok {
		if maj == 1 && min == 1 {
			return true
		}
	}
	return false
}

// LoadBalancer is the main Handle func
func (lb *LoadBalancer) LoadBalancer(rw http.ResponseWriter, req *http.Request) {
	if !isHTTPVersionSupported(req) {
		http.Error(rw, "Expected HTTP/1.1", http.StatusHTTPVersionNotSupported)
	}

	start := time.Now()

	// getting a response from cache
	err := lb.writePageIfIsInCache(rw, req)
	if err == nil {
		finish := time.Since(start)
		timer.SaveTimerDataGotFromCache(&finish)
		return
	} else {
		log.Println("Checking cache unsuccessful: ", err)
		if r := req.Context().Value(cache.OnlyIfCachedKey).(bool); r {
			http.Error(rw, cache.OnlyIfCachedError, http.StatusGatewayTimeout)
			return
		}
	}

	// on cache miss make request to backend
ChooseServer:
	server, err := lb.pool.GetNextPeer()
	if err != nil {
		log.Println(err)
		http.Error(rw, "Service not available", http.StatusServiceUnavailable)
		return
	}
	select {
	case server.RequestChan <- true:
	default:
		goto ChooseServer
	}

	var backendTime *time.Duration
	req, backendTime = timer.MakeRequestTimeTracker(req)

	resp, err := server.SendRequestToBackend(req)
	<-server.RequestChan

	// on cancellation
	if err == context.Canceled {
		log.Printf("[%s] %s", server.URL, err)
		return
	} else if err != nil {
		server.SetAlive(false) // СДЕЛАТЬ СЧЁТЧИК ИЛИ ПОЧИТАТЬ КАК У НДЖИНКС
		goto ChooseServer
	}
	defer resp.Body.Close()

	byteArray, err := backend.WriteBodyAndReturn(rw, resp)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	go lb.SaveToCache(req, resp, byteArray)

	finishRoundTrip := time.Since(start)
	timer.SaveTimeDataBackend(backendTime, &finishRoundTrip)
}

func setLogPrefixBDUTS() {
	log.SetPrefix("[BDUTS] ")
}
