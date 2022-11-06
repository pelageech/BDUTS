package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

const (
	Attempts int = iota
	Retry
)

// JsonBackend - backend structures
type JsonBackend struct {
	Url string
}

type LbConfig struct {
	Port int
}

// Backend holds the data about a server
type Backend struct {
	URL          *url.URL
	Alive        bool
	mux          sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
}

// SetAlive for this backend
func (b *Backend) SetAlive(alive bool) {
	b.mux.Lock()
	b.Alive = alive
	b.mux.Unlock()
}

// IsAlive returns true when backend is alive
func (b *Backend) IsAlive() (alive bool) {
	b.mux.RLock()
	alive = b.Alive
	b.mux.RUnlock()
	return
}

// ServerPool holds information about reachable backends
type ServerPool struct {
	backends []*Backend
	current  int64
}

// AddBackend to the server pool
func (s *ServerPool) AddBackend(backend *Backend) {
	s.backends = append(s.backends, backend)
}

// NextIndex atomically increase the counter and return an index
func (s *ServerPool) NextIndex() int {
	return int(atomic.AddInt64(&s.current, int64(1)) % int64(len(s.backends)))
}

// MarkBackendStatus changes a status of a backend
func (s *ServerPool) MarkBackendStatus(backendUrl *url.URL, alive bool) {
	for _, b := range s.backends {
		if b.URL.String() == backendUrl.String() {
			b.SetAlive(alive)
			break
		}
	}
}

// GetNextPeer returns next active peer to take a connection
func (s *ServerPool) GetNextPeer() *Backend {
	// loop entire backends to find out an Alive backend
	next := s.NextIndex()
	l := len(s.backends) + next // start from next and move a full cycle
	for i := next; i < l; i++ {
		idx := i % len(s.backends)     // take an index by modding
		if s.backends[idx].IsAlive() { // if we have an alive backend, use it and store if it's not the original one
			if i != next {
				atomic.StoreInt64(&s.current, int64(idx))
			}
			return s.backends[idx]
		}
	}
	return nil
}

// HealthCheck pings the backends and update the status
func (s *ServerPool) HealthCheck() {
	for _, b := range s.backends {
		status := "up"
		alive := isBackendAlive(b.URL)
		b.SetAlive(alive)
		if !alive {
			status = "down"
		}
		log.Printf("%s [%s]\n", b.URL, status)
	}
}

// GetAttemptsFromContext returns the attempts for request
func GetAttemptsFromContext(r *http.Request) int {
	if attempts, ok := r.Context().Value(Attempts).(int); ok {
		return attempts
	}
	return 1
}

// GetRetryFromContext returns the retries for request
func GetRetryFromContext(r *http.Request) int {
	if retry, ok := r.Context().Value(Retry).(int); ok {
		return retry
	}
	return 0
}

// lb load balances the incoming request
func lb(w http.ResponseWriter, r *http.Request) {

	attempts := GetAttemptsFromContext(r)
	if attempts > 3 {
		log.Printf("%s(%s) Max attempts reached, terminating\n", r.RemoteAddr, r.URL.Path)
		http.Error(w, "Service not available", http.StatusServiceUnavailable)
		return
	}

	peer := serverPool.GetNextPeer()
	if peer != nil {
		log.Printf("Connecting to %s\n", peer.URL) // логирование подключения

		// Send a request
		peer.ReverseProxy.ServeHTTP(w, r)

		log.Printf("Success %s\n", peer.URL)
		}
		return
	}
	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}

// isAlive checks whether a backend is Alive by establishing a TCP connection
func isBackendAlive(u *url.URL) bool {
	timeout := 2 * time.Second
	conn, err := net.DialTimeout("tcp", u.Host, timeout)
	if err != nil {
		log.Println("Site unreachable, error: ", err)
		return false
	}
	defer conn.Close()
	return true
}

// healthCheck runs a routine for check status of the backends every 2 minutes
func healthCheck() {
	t := time.NewTicker(time.Minute * 2)
	for {
		select {
		case <-t.C:
			log.Println("Starting health check...")
			serverPool.HealthCheck()
			log.Println("Health check completed")
		}
	}
}

// for each url in
func config(tokens []JsonBackend) {
	for _, tok := range tokens {

		// Parse url
		serverUrl, err := url.Parse(tok.Url)
		if err != nil {
			log.Fatal(err)
		}

		// Create new proxy-singleton
		proxy := httputil.NewSingleHostReverseProxy(serverUrl)

		// Set a behaviour on errors
		proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, e error) {

			// Print an error to log
			log.Printf("[%s] %s\n", serverUrl.Host, e.Error())

			// If context is cancelled, do nothing
			if e == context.Canceled {
				return
			}

			// Retry connection on fail
			retries := GetRetryFromContext(request)
			if retries < 3 {
				select {
				case <-time.After(10 * time.Millisecond):
					ctx := context.WithValue(request.Context(), Retry, retries+1)
					proxy.ServeHTTP(writer, request.WithContext(ctx))
				}
				return
			}

			// After 3 retries, mark this backend as down
			serverPool.MarkBackendStatus(serverUrl, false)

			// If the same request routing for few attempts with different backends, increase the count
			attempts := GetAttemptsFromContext(request)
			log.Printf("%s(%s) Attempting retry %d\n", request.RemoteAddr, request.URL.Path, attempts)
			ctx := context.WithValue(request.Context(), Attempts, attempts+1)
			log.Printf("Unsuccess %s", serverUrl.Host)
			lb(writer, request.WithContext(ctx))
		}

		// Configure a following backend
		serverPool.AddBackend(&Backend{
			URL:          serverUrl,
			Alive:        true,
			ReverseProxy: proxy,
		})
		log.Printf("Configured server: %s\n", serverUrl)
	}
}

var serverPool ServerPool

func main() {
	// Read Load-Balancer config
	buffLoadBalancerConfig, _ := os.ReadFile("resources/lb_config.json")

	var loadBalancerConfig LbConfig
	err := json.Unmarshal(buffLoadBalancerConfig, &loadBalancerConfig)
	if err != nil {
		log.Fatal("Bad lb_config.json", err)
	}

	// Read Backends
	buffBackends, _ := os.ReadFile("resources/backends.json")

	var backends []JsonBackend
	err = json.Unmarshal(buffBackends, &backends)
	if err != nil {
		log.Fatal("Bad backends.json", err)
	}

	// Configure all the backends
	config(backends)

	// Set handle for http load-balancer
	http.HandleFunc("/", lb)

	// Config TLS: setting a pair crt-key
	Crt, _ := tls.LoadX509KeyPair("MyCertificate.crt", "MyKey.key")
	tsconfig := &tls.Config{Certificates: []tls.Certificate{Crt}}

	// Set a port is listened by server to
	port := fmt.Sprintf(":%d", loadBalancerConfig.Port)

	// Start listening
	ln, err := tls.Listen("tcp", port, tsconfig)
	if err != nil {
		log.Fatal("There's problem on the lb")
	}

	// current is -1, it's automatically will turn into 0
	serverPool.current = -1

	// start health checking
	go healthCheck()

	// Serving
	log.Printf("Load Balancer started at %s\n", port)
	if err := http.Serve(ln, nil); err != nil {
		log.Fatal(err)
	}
}
