package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type AddRequestBodyJSON struct {
	Url                   string
	HealthCheckTcpTimeout int
	MaximalRequests       int
}

type RemoveRequestBodyJSON struct {
	Url string
}

const (
	defaultHost    = "localhost"
	defaultUrl     = ""
	defaultTimeout = 2000
	defaultMaxReq  = 1
	defaultToken   = ""

	proto             = "https://"
	addRequestPath    = "/serverPool/add"
	removeRequestPath = "/serverPool/remove"
)

var (
	host    = flag.String("h", defaultHost, "host:port of the load balancer for sending a request without protocol")
	add     = flag.String("add", defaultUrl, "adds a new backend to server pool, requires URL")
	remove  = flag.String("remove", defaultUrl, "remove the backend from server pool by URL")
	timeout = flag.Int("tout", defaultTimeout, "tcp timeout for backend replying, ms")
	maxReq  = flag.Int("max", defaultMaxReq, "amount of request able to be being processed in the same time")
	token   = flag.String("t", defaultToken, `jwt token without "Bearer "`)

	tr = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}} // todo: configure tls in right way
	c  = &http.Client{Transport: tr}
)

func addHandle() {
	addStruct := AddRequestBodyJSON{
		Url:                   *add,
		HealthCheckTcpTimeout: *timeout,
		MaximalRequests:       *maxReq,
	}
	body, err := json.Marshal(addStruct)
	if err != nil {
		fmt.Println("Failed to marshal JSON: ", err)
		os.Exit(1)
	}

	r := strings.NewReader(string(body))

	req, err := http.NewRequest(http.MethodPost, proto+*host+addRequestPath, r)
	if err != nil {
		fmt.Println("An error occurred while creating a request: ", err)
		os.Exit(1)
	}
	req.Header.Add("Authorization", "Bearer "+*token)

	resp, err := c.Do(req)
	if err != nil {
		fmt.Println("An error occurred while processing the request: ", err)
		os.Exit(1)
	}

	err = handleResponse(resp)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Successfully added")
}

func removeHandle() {
	removeStruct := RemoveRequestBodyJSON{
		Url: *remove,
	}
	body, err := json.Marshal(removeStruct)
	if err != nil {
		fmt.Println("Failed to marshal JSON: ", err)
	}

	r := strings.NewReader(string(body))

	req, err := http.NewRequest(http.MethodDelete, proto+*host+removeRequestPath, r)
	if err != nil {
		fmt.Println("An error occurred while creating a request: ", err)
		os.Exit(1)
	}

	req.Header.Add("Authorization", "Bearer "+*token)

	resp, err := c.Do(req)
	if err != nil {
		fmt.Println("An error occurred while processing the request: ", err)
		os.Exit(1)
	}

	err = handleResponse(resp)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Successfully removed")
}

func handleResponse(resp *http.Response) error {
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("401 Unauthorized")
	}

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		b, _ := io.ReadAll(resp.Body) // returns an empty slice on error
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(b))
	}

	return nil
}

func main() {
	flag.Parse()

	if *host == "" {
		fmt.Println("Host is not defined!")
		flag.Usage()
		return
	}

	if *token == defaultToken {
		fmt.Println("Warning: you have not defined a bearer token. ")
	}

	if *add != defaultUrl {
		addHandle()
	}

	if *remove != defaultUrl {
		removeHandle()
	}
}
