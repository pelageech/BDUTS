package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
)

type addRequestBodyJSON struct {
	Url                   string
	HealthCheckTcpTimeout int
	MaximalRequests       int
}

type removeRequestBodyJSON struct {
	Url string
}

type authRequestBodyJSON struct {
	Username string
	Password string
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
	authRequestPath   = "/admin/signin"

	defaultLogin    = ""
	defaultPassword = ""
)

var (
	help = flag.Bool("help", false, "show this message")

	host  = flag.String("H", defaultHost, "host:port of the load balancer for sending a request (without a protocol)")
	token = flag.String("t", defaultToken, `jwt token without "Bearer " for an authorization`)

	add     = flag.String("add", defaultUrl, "adds a new backend to server pool, requires URL (-tout and -max are optional params)")
	timeout = flag.Int("tout", defaultTimeout, "tcp timeout for backend replying in milliseconds")
	maxReq  = flag.Int("max", defaultMaxReq, "amount of request able to be being processed in the same time")

	remove = flag.String("remove", defaultUrl, "remove the backend from server pool, requires URL")

	getToken = flag.Bool("get-token", false, "get jwt-token, requires -login and -password")
	login    = flag.String("login", defaultLogin, "login for getting jwt-token")
	password = flag.String("password", defaultPassword, "password for getting jwt-token")

	tr = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}} // todo: configure tls in right way
	c  = &http.Client{Transport: tr}
)

func addHandle() {
	addStruct := addRequestBodyJSON{
		Url:                   *add,
		HealthCheckTcpTimeout: *timeout,
		MaximalRequests:       *maxReq,
	}
	body, err := json.Marshal(addStruct)
	if err != nil {
		fmt.Println("Failed to marshal JSON: ", err)
		os.Exit(1)
	}

	r := bytes.NewReader(body)

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
	removeStruct := removeRequestBodyJSON{
		Url: *remove,
	}
	body, err := json.Marshal(removeStruct)
	if err != nil {
		fmt.Println("Failed to marshal JSON: ", err)
		os.Exit(1)
	}

	r := bytes.NewReader(body)

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

func getTokenHandle() {
	authStruct := authRequestBodyJSON{
		Username: *login,
		Password: *password,
	}
	body, err := json.Marshal(authStruct)
	if err != nil {
		fmt.Println("Failed to marshal JSON: ", err)
		os.Exit(1)
	}

	r := bytes.NewReader(body)

	req, err := http.NewRequest(http.MethodPost, proto+*host+authRequestPath, r)
	if err != nil {
		fmt.Println("An error occurred while creating a request: ", err)
		os.Exit(1)
	}

	resp, err := c.Do(req)
	if err != nil {
		fmt.Println("An error occurred while processing the request: ", err)
		os.Exit(1)
	}

	token := resp.Header.Get("Authorization")
	if token != "" {
		fmt.Println(token)
		return
	}

	err = handleResponse(resp)
	if err != nil {
		fmt.Println("Something went wrong: ", err)
	}
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

	if *help {
		flag.Usage()
		return
	}

	if *getToken {
		getTokenHandle()
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
