package main

import (
	"crypto/tls"
	"fmt"
	"github.com/pelageech/BDUTS/lb"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/pelageech/BDUTS/backend"
	"github.com/pelageech/BDUTS/cache"
	"github.com/pelageech/BDUTS/config"
)

const (
	dbFillFactor      = 0.9
	lbConfigPath      = "./resources/config.json"
	serversConfigPath = "./resources/servers.json"
	cacheConfigPath   = "./resources/cache_config.json"

	maxDBSize          = 100 * (1 << 20) // 100 MB
	dbObserveFrequency = 10 * time.Second
)

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

	// cache configuration
	log.Println("Opening cache database")
	boltdb, err := cache.OpenDatabase(cache.DbDirectory + "/" + cache.DbName)
	if err != nil {
		log.Fatalln("DB error: ", err)
	}
	defer cache.CloseDatabase(boltdb)

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

	// health checker configuration
	healthCheckFunc := func(server *backend.Backend) {
		alive := server.IsAlive()
		server.SetAlive(alive)
		if alive {
			log.Printf("[%s] is alive.\n", server.URL.Host)
		} else {
			log.Printf("[%s] is down.\n", server.URL.Host)
		}
	}

	// creating new load balancer
	loadBalancer := lb.NewLoadBalancer(
		lb.NewLoadBalancerConfig(lbConfig.Port, time.Duration(lbConfig.HealthCheckPeriod)*time.Second),
		cache.NewCachingProperties(boltdb, cacheConfig),
		healthCheckFunc,
	)

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

	loadBalancer.ConfigureServerPool(serversConfig)

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
	controller := cache.NewCacheController(boltdb, dbDir, maxDBSize, dbFillFactor, dbControllerTicker)
	go controller.Observe()
	log.Println("Cache controller has been started!")

	// Serving
	http.HandleFunc("/", loadBalancer.LoadBalancer)
	http.HandleFunc("/favicon.ico", http.NotFound)

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

	// Config TLS: setting a pair crt-key
	Crt, _ := tls.LoadX509KeyPair("MyCertificate.crt", "MyKey.key")
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{Crt}}

	// Start listening
	ln, err := tls.Listen("tcp", fmt.Sprintf(":%d", loadBalancer.Config().Port()), tlsConfig)
	if err != nil {
		log.Fatal("There's problem with listening")
	}

	log.Printf("Load Balancer started at :%d\n", loadBalancer.Config().Port())
	if err := http.Serve(ln, nil); err != nil {
		log.Fatal(err)
	}
}
