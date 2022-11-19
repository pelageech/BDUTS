package main

/*
	Reading config from json files:
	- config.json
	- servers.json
*/

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"
)

type serverJSON struct {
	URL                   string
	HealthCheckTcpTimeout time.Duration
}

type configJSON struct {
	Port              int
	HealthCheckPeriod time.Duration
}

var loadBalancerConfig LoadBalancerConfig
var serverPool ServerPool

func readConfig() {
	/*
		read servers config
	*/
	serversFile, err := os.Open("resources/servers.json")
	if err != nil {
		log.Fatal("Failed to open servers config: ", err)
	}
	defer func() {
		if err := serversFile.Close(); err != nil {
			log.Fatal("Failed to close servers config: ", err)
		}
	}()

	serversFileByte, err := io.ReadAll(serversFile)
	if err != nil {
		log.Fatal("Failed to read servers config file: ", err)
	}

	var serversJSON []serverJSON
	if err := json.Unmarshal(serversFileByte, &serversJSON); err != nil {
		log.Fatal("Failed to unmarshal servers config: ", err)
	}
	configureServers(serversJSON)

	/*
		read load balancer config
	*/
	lbConfigFile, err := os.Open("resources/config.json")
	if err != nil {
		log.Fatal("Failed to open load balancer config file: ", err)
	}
	defer func() {
		if err := lbConfigFile.Close(); err != nil {
			log.Fatal("Failed to close load balancer config file: ", err)
		}
	}()

	lbConfigFileByte, err := io.ReadAll(lbConfigFile)
	if err != nil {
		log.Fatal("Failed to read load balancer config file: ", err)
	}

	var lbConfig configJSON
	if err := json.Unmarshal(lbConfigFileByte, &lbConfig); err != nil {
		log.Fatal("Failed to unmarshal load balancer config: ", err)
	}

	loadBalancerConfig.port = lbConfig.Port
	loadBalancerConfig.healthCheckPeriod = lbConfig.HealthCheckPeriod * time.Second
}

/*
		Configure server pool.
		For each backend we set up
		  - URL    *url.Url
		  - alive  *atomic.Bool
	      - server *httputil.ReverseProxy

		and then add it to the `serverPool` var.
*/
func configureServers(serversJSON []serverJSON) {
	for _, server := range serversJSON {
		var backend Backend
		var err error

		backend.URL, err = url.Parse(server.URL)
		if err != nil {
			log.Printf("Failed to parse server URL: %s\n", err)
			continue
		}
		backend.healthCheckTcpTimeout = server.HealthCheckTcpTimeout * time.Millisecond
		backend.alive.Store(false)
		backend.server = httputil.NewSingleHostReverseProxy(backend.URL)
		backend.server.Director = lbDirector

		serverPool.servers = append(serverPool.servers, &backend)
	}
}

func lbDirector(req *http.Request) {
	req = req
}
