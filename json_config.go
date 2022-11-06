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
)

type serverJSON struct {
	URL string
}

type configJSON struct {
	Port            int
	NumberOfRetries int
}

var port int
var numberOfRetries int

func readConfig() {
	/*
		read servers config
	*/
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

	for _, server := range serversJSON {
		serverURL, err := url.Parse(server.URL)
		if err != nil {
			log.Fatal("Failed to parse server URL", err)
		}

		servers = append(servers, serverURL)
	}

	/*
		read load balancer config
	*/
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

	port = lbConfig.Port
	numberOfRetries = lbConfig.NumberOfRetries
}
