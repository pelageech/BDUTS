package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/boltdb/bolt"
	"github.com/pelageech/BDUTS/cache"
	"github.com/pelageech/BDUTS/config"
	"github.com/pelageech/BDUTS/timer"
)

type myKey int

const (
	dbDirectory       = "./cache-data"
	dbName            = "database.db"
	lbConfigPath      = "./resources/config.json"
	serversConfigPath = "./resources/servers.json"
	cacheConfigPath   = "./resources/cache_config.json"

	maxDBSize          = 100 * 1024 * 1024 // 100 MB
	DBFillFactor       = 0.9
	dbObserveFrequency = 10 * time.Second

	keyStart = myKey(iota)
)

type LoadBalancer struct {
	config LoadBalancerConfig
	pool   ServerPool
}

func NewLoadBalancer(config LoadBalancerConfig) *LoadBalancer {
	return &LoadBalancer{
		config: config,
		pool:   ServerPool{},
	}
}

func (balancer *LoadBalancer) configureServerPool(servers []config.ServerConfig) {
	for _, server := range servers {
		log.Printf("%v", server)

		var backend Backend
		var err error

		backend.URL, err = url.Parse(server.URL)
		if err != nil {
			log.Printf("Failed to parse server URL: %s\n", err)
			continue
		}

		backend.healthCheckTcpTimeout = time.Duration(server.HealthCheckTcpTimeout) * time.Millisecond
		backend.alive = false

		backend.currentRequests = 0
		backend.maximalRequests = server.MaximalRequests

		balancer.pool.servers = append(balancer.pool.servers, &backend)
	}
}

// LoadBalancerConfig is parse from `config.json` file.
// It contains all the necessary information of the load balancer.
type LoadBalancerConfig struct {
	port              int
	healthCheckPeriod time.Duration
}

type Backend struct {
	URL                   *url.URL
	healthCheckTcpTimeout time.Duration
	mux                   sync.Mutex
	alive                 bool
	currentRequests       int32
	maximalRequests       int32
}

func (server *Backend) setAlive(b bool) {
	server.mux.Lock()
	server.alive = b
	server.mux.Unlock()
}

type ServerPool struct {
	mux     sync.Mutex
	servers []*Backend
	current int32
}

type ResponseError struct {
	request    *http.Request
	statusCode int
	err        error
}

var db *bolt.DB

func (server *Backend) makeRequest(r *http.Request) (*http.Response, *ResponseError) {
	newReq := *r
	req := &newReq
	respError := &ResponseError{request: req}
	serverUrl := server.URL

	// set req Host, URL and Request URI to forward a request to the origin server
	req.Host = serverUrl.Host
	req.URL.Host = serverUrl.Host
	req.URL.Scheme = serverUrl.Scheme

	// https://go.dev/src/net/http/client.go:217
	req.RequestURI = ""

	// save the response from the origin server
	originServerResponse, err := http.DefaultClient.Do(req)

	// error handler
	if err != nil {
		if uerr, ok := err.(*url.Error); ok {
			respError.err = uerr.Err

			if uerr.Err == context.Canceled {
				respError.statusCode = -1
			} else { // server error
				respError.statusCode = http.StatusInternalServerError
			}
		}
		return nil, respError
	}
	status := originServerResponse.StatusCode
	if status >= 500 && status < 600 &&
		status != http.StatusHTTPVersionNotSupported &&
		status != http.StatusNotImplemented {
		respError.statusCode = status
		return nil, respError
	}
	return originServerResponse, nil
}

func (serverPool *ServerPool) getNextPeer() (*Backend, error) {
	serverList := serverPool.servers

	serverPool.mux.Lock()
	defer serverPool.mux.Unlock()

	for i := 0; i < len(serverList); i++ {
		serverPool.current++
		if serverPool.current == int32(len(serverList)) {
			serverPool.current = 0
		}
		if serverList[serverPool.current].alive {
			return serverList[serverPool.current], nil
		}
	}

	return nil, errors.New("all backends are turned down")
}

func isHTTPVersionSupported(req *http.Request) bool {
	if maj, min, ok := http.ParseHTTPVersion(req.Proto); ok {
		if maj == 1 && min == 1 {
			return true
		}
	}
	return false
}

func checkCache(rw http.ResponseWriter, req *http.Request) error {
	log.Println("Try to get a response from cache...")

	cacheItem, err := cache.GetPageFromCache(db, req)
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

	if start, ok := req.Context().Value(keyStart).(time.Time); ok {
		finish := time.Since(start)
		timer.SaveTimerDataGotFromCache(&finish)
	} else {
		log.Println("Couldn't estimate transferring time")
	}

	return nil
}

func saveToCache(req *http.Request, resp *http.Response, byteArray []byte) {
	if !(resp.StatusCode >= 200 && resp.StatusCode < 400) {
		return
	}
	log.Println("Saving response in cache")

	go func() {
		cacheItem := &cache.Item{
			Body:   byteArray,
			Header: resp.Header,
		}
		err := cache.PutPageInCache(db, req, resp, cacheItem)
		if err != nil {
			log.Println("Unsuccessful operation: ", err)
			return
		}
		log.Println("Successfully saved")
	}()
}

func (balancer *LoadBalancer) loadBalancer(rw http.ResponseWriter, req *http.Request) {
	if !isHTTPVersionSupported(req) {
		http.Error(rw, "Expected HTTP/1.1", http.StatusHTTPVersionNotSupported)
	}

	start := time.Now()
	req = req.WithContext(context.WithValue(req.Context(), keyStart, start))

	// getting a response from cache
	if err := checkCache(rw, req); err == nil {
		return
	} else {
		log.Println("Checking cache unsuccessful: ", err)
	}

	// on cache miss make request to backend

	for {
		// get next server to send a request
		server, err := balancer.pool.getNextPeer()

		if err != nil {
			log.Println(err)
			http.Error(rw, "Service not available", http.StatusServiceUnavailable)
			return
		}
		defer atomic.AddInt32(&server.currentRequests, int32(-1))

		log.Printf("[%s] received a request\n", server.URL)

		var backendTime *time.Duration
		req, backendTime = timer.MakeRequestTimeTracker(req)
		defer func() {
			finishRoundTrip := time.Since(start)
			timer.SaveTimeDataBackend(backendTime, &finishRoundTrip)
		}()

		// send it to the backend
		resp, respError := server.makeRequest(req)
		if respError != nil {
			// on cancellation
			if respError.err == context.Canceled {
				//	cancel()
				log.Printf("[%s] %s", server.URL, respError.err)
				return
			}

			server.setAlive(false) // СДЕЛАТЬ СЧЁТЧИК ИЛИ ПОЧИТАТЬ КАК У НДЖИНКС
			log.Println(respError.err)
			continue
		}

		log.Printf("[%s] returned %s\n", server.URL, resp.Status)

		for key, values := range resp.Header {
			for _, value := range values {
				rw.Header().Add(key, value)
			}
		}

		byteArray, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
			log.Println(err)
		}
		resp.Body.Close()

		_, err = rw.Write(byteArray)
		if err != nil {
			http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
			log.Println(err)
		}

		// caching
		saveToCache(req, resp, byteArray)

		return
	}
}

func (balancer *LoadBalancer) healthChecker() {
	ticker := time.NewTicker(balancer.config.healthCheckPeriod)

	for {
		<-ticker.C
		log.Println("Health Check has been started!")
		balancer.healthCheck()
		log.Println("All the checks has been completed!")
	}
}

func (balancer *LoadBalancer) healthCheck() {
	for _, server := range balancer.pool.servers {
		alive := server.isAlive()
		server.setAlive(alive)
		if alive {
			log.Printf("[%s] is alive.\n", server.URL.Host)
		} else {
			log.Printf("[%s] is down.\n", server.URL.Host)
		}
	}
}

func (server *Backend) isAlive() bool {
	conn, err := net.DialTimeout("tcp", server.URL.Host, server.healthCheckTcpTimeout)

	if err != nil {
		log.Println("Connection problem: ", err)
		return false
	}

	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			log.Println("Failed to close connection: ", err)
		}
	}(conn)
	return true
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

	loadBalancer := NewLoadBalancer(LoadBalancerConfig{
		port:              lbConfig.Port,
		healthCheckPeriod: time.Duration(lbConfig.HealthCheckPeriod) * time.Second,
	})

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

	err = os.Mkdir(dbDirectory, 0777)
	if err != nil && !os.IsExist(err) {
		log.Fatalln("Cache files directory creation error: ", err)
	}

	// cache configuration
	log.Println("Opening cache database")
	db, err = cache.OpenDatabase(dbDirectory + "/" + dbName)
	if err != nil {
		log.Fatalln("DB error: ", err)
	}
	defer cache.CloseDatabase(db)

	// create directory for cache files
	err = os.Mkdir(cache.CachePath, 0777)
	if err != nil && !os.IsExist(err) {
		log.Fatalln("DB files directory creation error: ", err)
	}

	// open directory with cache files
	dbDir, err := os.Open(cache.CachePath)
	if err != nil {
		log.Fatalln("DB files opening error: ", err)
	}

	dbControllerTicker := time.NewTicker(dbObserveFrequency)
	defer dbControllerTicker.Stop()
	controller := cache.New(db, dbDir, maxDBSize, DBFillFactor, dbControllerTicker)
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

	// Config TLS: setting a pair crt-key
	Crt, _ := tls.LoadX509KeyPair("MyCertificate.crt", "MyKey.key")
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{Crt}}

	// Start listening
	ln, err := tls.Listen("tcp", fmt.Sprintf(":%d", loadBalancer.config.port), tlsConfig)
	if err != nil {
		log.Fatal("There's problem with listening")
	}

	// current is -1, it's automatically will turn into 0
	loadBalancer.pool.current = -1

	// Serving
	http.HandleFunc("/", loadBalancer.loadBalancer)
	http.HandleFunc("/favicon.ico", http.NotFound)

	// wait while other containers will be ready
	time.Sleep(5 * time.Second)

	// Firstly, identify the working servers
	log.Println("Configured! Now setting up the first health check...")
	loadBalancer.healthCheck()

	log.Println("Ready!")

	// set up health check
	go loadBalancer.healthChecker()

	log.Printf("Load Balancer started at :%d\n", loadBalancer.config.port)
	if err := http.Serve(ln, nil); err != nil {
		log.Fatal(err)
	}
}
