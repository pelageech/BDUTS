package main

import (
	"crypto/tls"
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/pelageech/BDUTS/backend"
	"github.com/pelageech/BDUTS/cache"
	"github.com/pelageech/BDUTS/config"
	"github.com/pelageech/BDUTS/lb"
)

const (
	dbFillFactor      = 0.9
	lbConfigPath      = "./resources/config.json"
	serversConfigPath = "./resources/servers.json"
	cacheConfigPath   = "./resources/cache_config.json"
)

func loadBalancerConfigure() *config.LoadBalancerConfig {
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
	return lbConfig
}

func cacheConfigure() *config.CacheConfig {
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
	return cacheConfig
}

func serversConfigure() []config.ServerConfig {
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
	return serversConfig
}

func cacheCleanerConfigure(dbControllerTicker *time.Ticker, maxCacheSize int64) *cache.CacheCleaner {
	err := os.Mkdir(cache.DbDirectory, 0777)
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
	return cache.NewCacheCleaner(dbDir, maxCacheSize, dbFillFactor, dbControllerTicker)
}

func main() {
	lbConfJSON := loadBalancerConfigure()
	lbConfig := lb.NewLoadBalancerConfig(
		lbConfJSON.Port,
		time.Duration(lbConfJSON.HealthCheckPeriod)*time.Millisecond,
		lbConfJSON.MaxCacheSize,
		time.Duration(lbConfJSON.ObserveFrequency)*time.Millisecond,
	)

	cacheConfig := cacheConfigure()

	// database
	log.Println("Opening cache database")
	if err := os.Mkdir(cache.DbDirectory, 0700); err != nil && os.IsExist(err) {
		log.Fatalln("couldn't create a directory " + cache.DbDirectory + ": " + err.Error())
	}
	boltdb, err := cache.OpenDatabase(cache.DbDirectory + "/" + cache.DbName)
	if err != nil {
		log.Fatalln("DB error: ", err)
	}
	defer cache.CloseDatabase(boltdb)

	// thread that clears the cache
	dbControllerTicker := time.NewTicker(lbConfig.ObserveFrequency())
	controller := cacheCleanerConfigure(dbControllerTicker, lbConfig.MaxCacheSize())
	defer dbControllerTicker.Stop()

	cacheProps := cache.NewCachingProperties(boltdb, cacheConfig, controller)
	cacheProps.CalculateSize()

	// health checker configuration
	healthCheckFunc := func(server *backend.Backend) {
		alive := server.CheckIfAlive()
		server.SetAlive(alive)
		if alive {
			log.Printf("[%s] is alive.\n", server.URL().Host)
		} else {
			log.Printf("[%s] is down.\n", server.URL().Host)
		}
	}

	// creating new load balancer
	loadBalancer := lb.NewLoadBalancerWithPool(
		lbConfig,
		cacheProps,
		healthCheckFunc,
		serversConfigure(),
	)

	// wait while other containers will be ready
	time.Sleep(5 * time.Second)

	// Firstly, identify the working servers
	log.Println("Configured! Now setting up the first health check...")
	for _, server := range loadBalancer.Pool().Servers() {
		loadBalancer.HealthCheckFunc()(server)
	}
	log.Println("Ready!")

	// set up health check
	go loadBalancer.HealthChecker()
	go loadBalancer.CacheProps().Observe()

	// Serving
	http.HandleFunc("/", loadBalancer.LoadBalancer)
	http.HandleFunc("/favicon.ico", http.NotFound)
	http.HandleFunc("/serverPool/add", loadBalancer.AddServer)
	http.HandleFunc("/serverPool/remove", loadBalancer.RemoveServer)
	http.HandleFunc("/serverPool", loadBalancer.GetServers)

	// Config TLS: setting a pair crt-key
	Crt, _ := tls.LoadX509KeyPair("resources/Cert.crt", "resources/Key.key")
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{Crt}}

	ln, err := tls.Listen("tcp", fmt.Sprintf(":%d", loadBalancer.Config().Port()), tlsConfig)
	if err != nil {
		log.Fatal("There's problem with listening")
	}

	wg := sync.WaitGroup{}
	wg.Add(2)
	log.Printf("Load Balancer started at :%d\n", loadBalancer.Config().Port())
	go func() {
		if err := http.Serve(ln, nil); err != nil {
			log.Fatalln(err)
		}
		wg.Done()
	}()

	// prometheus part

	server := http.Server{
		Addr:    ":8081",
		Handler: promhttp.Handler(),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatalln(err)
		}
		wg.Done()
	}()

	wg.Wait()
}
