package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/go-playground/validator/v10"
	"github.com/pelageech/BDUTS/auth"
	"github.com/pelageech/BDUTS/backend"
	"github.com/pelageech/BDUTS/cache"
	"github.com/pelageech/BDUTS/config"
	"github.com/pelageech/BDUTS/db"
	"github.com/pelageech/BDUTS/email"
	"github.com/pelageech/BDUTS/lb"
	"github.com/pelageech/BDUTS/metrics"
	"github.com/pelageech/BDUTS/timer"
)

const (
	dbFillFactor      = 0.9
	lbConfigPath      = "./resources/config.json"
	serversConfigPath = "./resources/servers.json"
	cacheConfigPath   = "./resources/cache_config.json"

	loggerPrefixMain  = "BDUTS"
	loggerPrefixCache = "BDUTS_CACHE"
	loggerPrefixLB    = "BDUTS_LB"
	loggerPrefixTimer = "BDUTS_TIMER"
	loggerPrefixPool  = "BDUTS_POOL"
)

var logger *log.Logger

func loadBalancerConfigure() *config.LoadBalancerConfig {
	loadBalancerReader, err := config.NewLoadBalancerReader(lbConfigPath)
	if err != nil {
		logger.Fatal("Failed to create LoadBalancerReader", "err", err)
	}
	defer func(loadBalancerReader *config.LoadBalancerReader) {
		err := loadBalancerReader.Close()
		if err != nil {
			logger.Fatal("Failed to close LoadBalancerReader", "err", err)
		}
	}(loadBalancerReader)

	lbConfig, err := loadBalancerReader.ReadLoadBalancerConfig()
	if err != nil {
		logger.Fatal("Failed to read LoadBalancerConfig", "err", err)
	}
	return lbConfig
}

func cacheConfigure() *config.CacheConfig {
	cacheReader, err := config.NewCacheReader(cacheConfigPath)
	if err != nil {
		logger.Fatal("Failed to create CacheReader", "err", err)
	}
	defer func(cacheReader *config.CacheReader) {
		err := cacheReader.Close()
		if err != nil {
			logger.Fatal("Failed to close CacheReader", "err", err)
		}
	}(cacheReader)

	cacheConfig, err := config.ReadCacheConfig(cacheReader)
	if err != nil {
		logger.Fatal("Failed to read CacheConfig", "err", err)
	}
	return cacheConfig
}

func serversConfigure() []config.ServerConfig {
	serversReader, err := config.NewServersReader(serversConfigPath)
	if err != nil {
		logger.Fatal("Failed to create ServersReader", "err", err)
	}
	defer func(serversReader *config.ServersReader) {
		err := serversReader.Close()
		if err != nil {
			logger.Fatal("Failed to close ServersReader", "err", err)
		}
	}(serversReader)

	serversConfig, err := serversReader.ReadServersConfig()
	if err != nil {
		logger.Fatal("Failed to read ServersConfig", "err", err)
	}
	return serversConfig
}

func cacheCleanerConfigure(dbControllerTicker *time.Ticker, maxCacheSize int64) *cache.CacheCleaner {
	err := os.Mkdir(cache.DbDirectory, 0777)
	if err != nil && !os.IsExist(err) {
		logger.Fatal("Cache files directory creation error", "err", err)
	}

	// create directory for cache files
	err = os.Mkdir(cache.PagesPath, 0777)
	if err != nil && !os.IsExist(err) {
		logger.Fatal("DB files directory creation error", "err", err)
	}

	// open directory with cache files
	dbDir, err := os.Open(cache.PagesPath)
	if err != nil {
		logger.Fatal("DB files opening error", "err", err)
	}
	return cache.NewCacheCleaner(dbDir, maxCacheSize, dbFillFactor, dbControllerTicker)
}

func main() {
	logger = log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
	})
	logger.SetPrefix(loggerPrefixMain)
	cache.LoggerConfig(loggerPrefixCache)
	backend.LoggerConfig(loggerPrefixPool)
	timer.LoggerConfig(loggerPrefixTimer)
	lb.LoggerConfig(loggerPrefixLB)

	lbConfJSON := loadBalancerConfigure()
	lbConfig := lb.NewLoadBalancerConfig(
		lbConfJSON.Port,
		time.Duration(lbConfJSON.HealthCheckPeriod)*time.Millisecond,
		lbConfJSON.MaxCacheSize,
		time.Duration(lbConfJSON.ObserveFrequency)*time.Millisecond,
	)

	cacheConfig := cacheConfigure()

	// database
	logger.Info("Opening cache database")
	if err := os.Mkdir(cache.DbDirectory, 0700); err != nil && !os.IsExist(err) {
		logger.Fatal("Couldn't create a directory "+cache.DbDirectory, "err", err)
	}
	boltdb, err := cache.OpenDatabase(cache.DbDirectory + "/" + cache.DbName)
	if err != nil {
		logger.Fatal("Failed to open boltdb", "err", err)
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
			logger.Infof("[%s] is alive.\n", server.URL().Host)
		} else {
			logger.Warnf("[%s] is down.\n", server.URL().Host)
		}
	}

	// creating new load balancer
	loadBalancer := lb.NewLoadBalancerWithPool(
		lbConfig,
		cacheProps,
		healthCheckFunc,
		serversConfigure(),
	)

	// Firstly, identify the working servers
	logger.Info("Configured! Now setting up the first health check...")
	for _, server := range loadBalancer.Pool().Servers() {
		loadBalancer.HealthCheckFunc()(server)
	}
	logger.Info("Ready!")

	// set up health check
	go loadBalancer.HealthChecker()
	go loadBalancer.CacheProps().Observe()

	// connect to lb_admins database
	postgresUser := os.Getenv("POSTGRES_USER")
	password := os.Getenv("USER_PASSWORD")
	host := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")
	dbService := db.Service{}
	dbService.SetLogger(logger)
	err = dbService.Connect(postgresUser, password, host, dbPort, dbName)
	if err != nil {
		logger.Fatal("Unable to connect to postgresql database", "err", err)
	}
	defer func(dbService *db.Service) {
		// Do not log the error since it has already been logged in dbService.Close()
		// However, return the error in case someone wants to implement multiple attempts to close the database
		_ = dbService.Close()
	}(&dbService)

	// set up email
	smtpUser := os.Getenv("SMTP_USER")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	sender := email.New(smtpUser, smtpPassword, smtpHost, smtpPort, logger)

	// set up auth
	validate := validator.New()
	signKey, found := os.LookupEnv("JWT_SIGNING_KEY")
	if !found {
		logger.Fatal("JWT signing key is not found")
	}
	authSvc := auth.New(dbService, sender, validate, []byte(signKey), logger)

	// Create a CORS middleware handler function
	cors := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Add CORS headers to the response
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			w.Header().Set("Access-Control-Expose-Headers", "Authorization")

			// If the request method is OPTIONS, return a successful response with no body
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			// Call the next handler function in the chain
			h.ServeHTTP(w, r)
		})
	}

	// Serving
	http.HandleFunc("/", loadBalancer.LoadBalancerHandler)
	http.HandleFunc("/favicon.ico", http.NotFound)
	http.Handle("/serverPool/add", cors(authSvc.AuthenticationMiddleware(http.HandlerFunc(loadBalancer.AddServerHandler))))
	http.Handle("/serverPool/remove", cors(authSvc.AuthenticationMiddleware(http.HandlerFunc(loadBalancer.RemoveServerHandler))))
	http.Handle("/serverPool", cors(authSvc.AuthenticationMiddleware(http.HandlerFunc(loadBalancer.GetServersHandler))))
	http.Handle("/admin/signup", cors(authSvc.AuthenticationMiddleware(http.HandlerFunc(authSvc.SignUp))))
	http.Handle("/admin/password", cors(authSvc.AuthenticationMiddleware(http.HandlerFunc(authSvc.ChangePassword))))
	http.Handle("/admin/signin", cors(http.HandlerFunc(authSvc.SignIn)))

	// Config TLS: setting a pair crt-key
	Crt, _ := tls.LoadX509KeyPair("resources/Cert.crt", "resources/Key.key")
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{Crt}}

	ln, err := tls.Listen("tcp", fmt.Sprintf(":%d", loadBalancer.Config().Port()), tlsConfig)
	if err != nil {
		logger.Fatal("Failed to start tcp listener", "err", err)
	}

	wg := sync.WaitGroup{}
	wg.Add(2)
	logger.Infof("Load Balancer started at :%d\n", loadBalancer.Config().Port())
	go func() {
		if err := http.Serve(ln, nil); err != nil {
			logger.Fatal("Failed to serve tcp listener", "err", err)
		}
		wg.Done()
	}()

	// prometheus part
	metrics.Init(loadBalancer.CacheProps().Size, loadBalancer.CacheProps().PagesCount)
	server := http.Server{
		Addr:    ":8081",
		Handler: metrics.Handler(),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			logger.Fatal("Failed to start prometheus server", "err", err)
		}
		wg.Done()
	}()
	wg.Wait()
}
