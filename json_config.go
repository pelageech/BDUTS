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
	"net/url"
	"os"
	"time"
)

type serverJSON struct {
	URL                   string
	HealthCheckTcpTimeout time.Duration
	MaximalRequests       int32
}

type configJSON struct {
	Port              int
	HealthCheckPeriod time.Duration
}

var loadBalancerConfig LoadBalancerConfig
var serverPool ServerPool

func readServersConfig() []serverJSON {
	serversFile, err := os.Open("resources/servers.json")
	if err != nil {
		log.Fatal("Failed to open servers config: ", err)
	}
	defer func() {
		err = serversFile.Close()
		if err != nil {
			log.Fatal("Failed to close servers config: ", err)
		}
	}()

	serversFileByte, err := io.ReadAll(serversFile)
	if err != nil {
		log.Fatal("Failed to read servers config file: ", err)
	}

	var serversJSON []serverJSON
	err = json.Unmarshal(serversFileByte, &serversJSON)
	if err != nil {
		log.Fatal("Failed to unmarshal servers config: ", err)
	}

	return serversJSON
}

/*
Configure server pool.
For each backend we set up
  - URL,
  - Alive bool

and then add it to the `serverPool“ var.
*/
func configureServerPool(servers []serverJSON) {
	for _, server := range servers {
		log.Printf("%v", server)

		var backend Backend
		var err error

		backend.URL, err = url.Parse(server.URL)
		if err != nil {
			log.Printf("Failed to parse server URL: %s\n", err)
			continue
		}

		backend.healthCheckTcpTimeout = server.HealthCheckTcpTimeout * time.Millisecond
		backend.alive = false

		backend.currentRequests = 0
		backend.maximalRequests = server.MaximalRequests

		serverPool.servers = append(serverPool.servers, &backend)
	}
}

func readLoadBalancerConfig() configJSON {
	lbConfigFile, err := os.Open("resources/config.json")
	if err != nil {
		log.Fatal("Failed to open load balancer config file: ", err)
	}
	defer func() {
		err = lbConfigFile.Close()
		if err != nil {
			log.Fatal("Failed to close load balancer config file: ", err)
		}
	}()

	lbConfigFileByte, err := io.ReadAll(lbConfigFile)
	if err != nil {
		log.Fatal("Failed to read load balancer config file: ", err)
	}

	var lbConfig configJSON
	err = json.Unmarshal(lbConfigFileByte, &lbConfig)
	if err != nil {
		log.Fatal("Failed to unmarshal load balancer config: ", err)
	}

	return lbConfig
}

func configureLoadBalancer(cfg configJSON) {
	loadBalancerConfig.port = cfg.Port
	loadBalancerConfig.healthCheckPeriod = cfg.HealthCheckPeriod * time.Second
}

func readConfig() {
	servers := readServersConfig()
	configureServerPool(servers)
	lbCfg := readLoadBalancerConfig()
	configureLoadBalancer(lbCfg)
}
