package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/boltdb/bolt"
	"github.com/pelageech/BDUTS/backend"
	"github.com/pelageech/BDUTS/cache"
	"github.com/pelageech/BDUTS/config"
	"github.com/pelageech/BDUTS/timer"
)

const (
	dbFillFactor      = 0.9
	lbConfigPath      = "./resources/config.json"
	serversConfigPath = "./resources/servers.json"
	cacheConfigPath   = "./resources/cache_config.json"

	maxDBSize          = 100 * 1024 * 1024 // 100 MB
	dbObserveFrequency = 10 * time.Second
)

// LoadBalancerConfig is parse from `config.json` file.
// It contains all the necessary information of the load balancer.
type LoadBalancerConfig struct {
	port              int
	healthCheckPeriod time.Duration
}

// LoadBalancer is a struct that contains all the configuration
// of the load balancer.
type LoadBalancer struct {
	config          LoadBalancerConfig
	pool            backend.ServerPool
	db              *bolt.DB
	healthCheckFunc func(*backend.Backend)
}

// NewLoadBalancer is the constructor of the load balancer
func NewLoadBalancer(config LoadBalancerConfig, boltDB *bolt.DB, healthChecker func(*backend.Backend)) *LoadBalancer {
	return &LoadBalancer{
		config:          config,
		pool:            backend.ServerPool{},
		db:              boltDB,
		healthCheckFunc: healthChecker,
	}
}

// feels the balancer pool with servers
func (balancer *LoadBalancer) configureServerPool(servers []config.ServerConfig) {
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

		balancer.pool.Servers = append(balancer.pool.Servers, &b)
	}
}

// uses balancer db for taking the page from cache and writing it to http.ResponseWriter
// if such a page is in cache
func (balancer *LoadBalancer) writePageIfIsInCache(rw http.ResponseWriter, req *http.Request) error {
	log.Println("Try to get a response from cache...")

	cacheItem, err := cache.GetPageFromCache(balancer.db, req)
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

// the main Handle func
func (balancer *LoadBalancer) loadBalancer(rw http.ResponseWriter, req *http.Request) {
	if !isHTTPVersionSupported(req) {
		http.Error(rw, "Expected HTTP/1.1", http.StatusHTTPVersionNotSupported)
	}

	start := time.Now()

	// getting a response from cache
	err := balancer.writePageIfIsInCache(rw, req)
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
	server, err := balancer.pool.GetNextPeer()
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

	resp, err := server.SendRequestToBackend(rw, req)
	if err == context.Canceled {
		return
	} else if err != nil {
		goto ChooseServer
	}
	defer resp.Body.Close()

	byteArray, err := backend.WriteBodyAndReturn(rw, resp)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	go backend.SaveToCache(balancer.db, req, resp, byteArray)

	finishRoundTrip := time.Since(start)
	timer.SaveTimeDataBackend(backendTime, &finishRoundTrip)
}

// HeathChe
func (balancer *LoadBalancer) HealthChecker() {
	ticker := time.NewTicker(balancer.config.healthCheckPeriod)

	for {
		<-ticker.C
		log.Println("Health Check has been started!")
		for _, server := range balancer.pool.Servers {
			balancer.healthCheckFunc(server)
		}
		log.Println("All the checks has been completed!")
	}
}

func main() {

	// load balancer configuration
	loadBalancerReader, err := config.NewLoadBalancerReader(lbConfigPath)
	if err != nil {
		log.Fatal(err)
	}
	defer func(loadBalancerReader *config.LoadBalancerReader) {
		err := loadBalancerReader.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(loadBalancerReader)

	lbConfig, err := loadBalancerReader.ReadLoadBalancerConfig()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Opening cache database")
	boltdb, err := cache.OpenDatabase(cache.DbDirectory + "/" + cache.DbName)
	if err != nil {
		log.Fatalln("DB error: ", err)
	}
	defer cache.CloseDatabase(boltdb)

	healthCheckFunc := func(server *backend.Backend) {
		alive := server.IsAlive()
		server.SetAlive(alive)
		if alive {
			log.Printf("[%s] is alive.\n", server.URL.Host)
		} else {
			log.Printf("[%s] is down.\n", server.URL.Host)
		}
	}

	loadBalancer := NewLoadBalancer(LoadBalancerConfig{
		port:              lbConfig.Port,
		healthCheckPeriod: time.Duration(lbConfig.HealthCheckPeriod) * time.Second,
	}, boltdb, healthCheckFunc)

	// backends configuration
	serversReader, err := config.NewServersReader(serversConfigPath)
	if err != nil {
		log.Fatal(err)
	}
	defer func(serversReader *config.ServersReader) {
		err := serversReader.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(serversReader)

	serversConfig, err := serversReader.ReadServersConfig()
	if err != nil {
		log.Fatal(err)
	}

	loadBalancer.configureServerPool(serversConfig)

	err = os.Mkdir(cache.DbDirectory, 0777)
	if err != nil && !os.IsExist(err) {
		log.Fatalln("Cache files directory creation error: ", err)
	}

	// create directory for cache files
	err = os.Mkdir(cache.PagesPath, 0777)
	if err != nil && !os.IsExist(err) {
		log.Fatalln("DB files directory creation error: ", err)
	}

	// open directory with cache files
	dbDir, err := os.Open(cache.PagesPath)
	if err != nil {
		log.Fatalln("DB files opening error: ", err)
	}

	// thread that clears the cache
	dbControllerTicker := time.NewTicker(dbObserveFrequency)
	defer dbControllerTicker.Stop()
	controller := cache.New(boltdb, dbDir, maxDBSize, dbFillFactor, dbControllerTicker)
	go controller.Observe()
	log.Println("Cache controller has been started!")

	cacheReader, err := config.NewCacheReader(cacheConfigPath)
	if err != nil {
		log.Fatal(err)
	}
	defer func(cacheReader *config.CacheReader) {
		err := cacheReader.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(cacheReader)

	cacheConfig, err := config.ReadCacheConfig(cacheReader)
	if err != nil {
		log.Fatal(err)
	}
	config.RequestKey = config.ParseRequestKey(cacheConfig.RequestKey)

	// current is -1, it's automatically will turn into 0
	loadBalancer.pool.Current = -1

	// Serving
	http.HandleFunc("/", loadBalancer.loadBalancer)
	http.HandleFunc("/favicon.ico", http.NotFound)

	// wait while other containers will be ready
	time.Sleep(5 * time.Second)

	// Firstly, identify the working servers
	log.Println("Configured! Now setting up the first health check...")
	for _, server := range loadBalancer.pool.Servers {
		loadBalancer.healthCheckFunc(server)
	}

	log.Println("Ready!")

	// set up health check
	go loadBalancer.HealthChecker()

	// Config TLS: setting a pair crt-key
	Crt, _ := tls.LoadX509KeyPair("MyCertificate.crt", "MyKey.key")
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{Crt}}

	// Start listening
	ln, err := tls.Listen("tcp", fmt.Sprintf(":%d", loadBalancer.config.port), tlsConfig)
	if err != nil {
		log.Fatal("There's problem with listening")
	}

	log.Printf("Load Balancer started at :%d\n", loadBalancer.config.port)
	if err := http.Serve(ln, nil); err != nil {
		log.Fatal(err)
	}
}
