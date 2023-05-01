package lb

import (
	"net/http"
	"os"
	"time"

	"github.com/charmbracelet/log"
	"github.com/pelageech/BDUTS/backend"
	"github.com/pelageech/BDUTS/cache"
	"github.com/pelageech/BDUTS/config"
)

// LoadBalancerConfig is parse from `config.json` file.
// It contains all the necessary information of the load balancer.
type LoadBalancerConfig struct {
	port              int
	healthCheckPeriod time.Duration
	maxCacheSize      int64
	observeFrequency  time.Duration
}

var (
	logger = log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
	})
)

func LoggerConfig(prefix string) {
	logger.SetPrefix(prefix)
}

func NewLoadBalancerConfig(
	port int,
	healthCheckPeriod time.Duration,
	maxCacheSize int64,
	observeFrequency time.Duration,
) *LoadBalancerConfig {
	return &LoadBalancerConfig{
		port:              port,
		healthCheckPeriod: healthCheckPeriod,
		maxCacheSize:      maxCacheSize,
		observeFrequency:  observeFrequency,
	}
}

func (c *LoadBalancerConfig) Port() int {
	return c.port
}

func (c *LoadBalancerConfig) HealthCheckPeriod() time.Duration {
	return c.healthCheckPeriod
}

func (c *LoadBalancerConfig) MaxCacheSize() int64 {
	return c.maxCacheSize
}

func (c *LoadBalancerConfig) ObserveFrequency() time.Duration {
	return c.observeFrequency
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

func NewLoadBalancerWithPool(
	config *LoadBalancerConfig,
	cachingProperties *cache.CachingProperties,
	healthChecker func(*backend.Backend),
	servers []config.ServerConfig,
) *LoadBalancer {
	lb := NewLoadBalancer(
		config,
		cachingProperties,
		healthChecker,
	)
	lb.pool.ConfigureServerPool(servers)

	return lb
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
		logger.Info("Health Check has been started!")
		for _, server := range lb.Pool().Servers() {
			lb.healthCheckFunc(server)
		}
		logger.Info("All the checks has been completed!")
	}
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
