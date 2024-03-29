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

type signInBodyJSON struct {
	Username string
	Password string
}

type signUpBodyJSON struct {
	Username string
	Email    string
}

type changeBodyJSON struct {
	OldPassword        string
	NewPassword        string
	NewPasswordConfirm string
}

const (
	empty          = ""
	defaultHost    = "localhost"
	defaultTimeout = 2000
	defaultMaxReq  = 1

	proto             = "https://"
	addRequestPath    = "/serverPool/add"
	removeRequestPath = "/serverPool/remove"
	signInRequestPath = "/admin/signin"
	signUpRequestPath = "/admin/signup"
	changeRequestPath = "/admin/password"
	deleteRequestPath = "/admin"
	clearRequestPath  = "/admin/clear"
)

var (
	help = flag.Bool("help", false, "show this message")

	host  = flag.String("H", defaultHost, "host:port of the load balancer for sending a request (without a protocol)")
	token = flag.String("t", empty, `jwt token without "Bearer " for an authorization`)

	add     = flag.String("add", empty, "adds a new backend to server pool, requires URL (-tout and -max are optional params)")
	timeout = flag.Int("timeout", defaultTimeout, "tcp timeout for backend replying in milliseconds")
	maxReq  = flag.Int("max", defaultMaxReq, "amount of request able to be being processed in the same time")

	remove = flag.String("remove", empty, "remove the backend from server pool, requires URL")

	signIn   = flag.Bool("signin", false, "sign in and get jwt-token, requires -login and -password")
	login    = flag.String("login", empty, "login for getting jwt-token")
	password = flag.String("password", empty, "password for getting jwt-token")

	signUp = flag.Bool("signup", false, "registers a new admin in a system, requires login and email")
	email  = flag.String("email", empty, "email is required to sign up. You get a password there")

	change  = flag.Bool("change", false, "a request for server to change password. Requires old, new and confirming passwords and token")
	oldPass = flag.String("old", empty, "an old password")
	newPass = flag.String("new", empty, "a new password")
	confirm = flag.String("confirm", empty, "confirm a new password")

	delUser = flag.Bool("del-user", false, "deletes a user, requires login and token")

	clear = flag.Bool("clear", false, "clear cache")

	tr = &http.Transport{TLSClientConfig: &tls.Config{}}
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

func signInHandle() {
	signInStruct := signInBodyJSON{
		Username: *login,
		Password: *password,
	}
	body, err := json.Marshal(signInStruct)
	if err != nil {
		fmt.Println("Failed to marshal JSON: ", err)
		os.Exit(1)
	}

	r := bytes.NewReader(body)

	req, err := http.NewRequest(http.MethodPost, proto+*host+signInRequestPath, r)
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

func signUpHandle() {
	signUpStruct := signUpBodyJSON{
		Username: *login,
		Email:    *email,
	}
	body, err := json.Marshal(signUpStruct)
	if err != nil {
		fmt.Println("Failed to marshal JSON: ", err)
		os.Exit(1)
	}

	r := bytes.NewReader(body)

	req, err := http.NewRequest(http.MethodPost, proto+*host+signUpRequestPath, r)
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
		fmt.Println("Something went wrong: ", err)
		os.Exit(1)
	}
}

func changeHandle() {
	changeStruct := changeBodyJSON{
		OldPassword:        *oldPass,
		NewPassword:        *newPass,
		NewPasswordConfirm: *confirm,
	}
	body, err := json.Marshal(changeStruct)
	if err != nil {
		fmt.Println("Failed to marshal JSON: ", err)
		os.Exit(1)
	}

	r := bytes.NewReader(body)

	req, err := http.NewRequest(http.MethodPatch, proto+*host+changeRequestPath, r)
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
		fmt.Println("Something went wrong: ", err)
		os.Exit(1)
	}
}

func deleteHandle() {
	req, err := http.NewRequest(http.MethodDelete, proto+*host+deleteRequestPath, nil)
	if err != nil {
		fmt.Println("An error occurred while creating a request: ", err)
		os.Exit(1)
	}

	q := req.URL.Query()
	q.Add("username", *login)
	req.URL.RawQuery = q.Encode()

	req.Header.Add("Authorization", "Bearer "+*token)

	resp, err := c.Do(req)
	if err != nil {
		fmt.Println("An error occurred while processing the request: ", err)
		os.Exit(1)
	}

	err = handleResponse(resp)
	if err != nil {
		fmt.Println("Something went wrong: ", err)
		os.Exit(1)
	}
}

func clearHandler() {
	req, err := http.NewRequest(http.MethodDelete, proto+*host+clearRequestPath, nil)
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
		fmt.Println("Something went wrong: ", err)
		os.Exit(1)
	}
}

func handleResponse(resp *http.Response) error {
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("401 Unauthorized")
	}

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		b, _ := io.ReadAll(resp.Body) // returns an empty slice on error
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(b))
	}

	return nil
}

func handleArgs() {
	if *help {
		fmt.Printf("\n\t---| BDUTS Admin panel |---\n\n")
		flag.Usage()
		fmt.Println("\nThis app provides you with dealing with server pool of the balancer.\n" +
			"Before it you must get a token which permits you adding and removing backends.\n" +
			"A simple usage:\n" +
			"\t-signin -H localhost:8080 -login admin -password admin\n" +
			"where, of course, your own host, login and password. There will be a bearer token.\n\n" +
			"To add a new backend use this:\n" +
			"\t-H localhost:8080 -add http://192.168.15.1:9090 -timeout 1000 -max 10 -t <token>\n" +
			"Notice that -tout and -max are optional.\n\n" +
			"To remove a backend use this:\n" +
			"\t-H localhost:8080 -remove http://192.168.15.1:9090 -t <token>\n\n" +
			"To clear cache:" +
			"\t-H localhost:8080 -clear -t <token>" +
			"Administrating:\n" +
			"- Sign In\n" +
			"\t-signin -H localhost:8080 -login admin -password admin\n\n" +
			"- Sign Up\n" +
			"\t-signup -H localhost:8080 -login admin -email example@mail.ru -t <token>\n\n" +
			"- Change Password\n" +
			"\t-change -H localhost:8080 -old oldPass -new newPass -confirm newPass -t <token>\n\n" +
			"- Delete user\n" +
			"\t-del-user -H localhost:8080 -login admin -t <token>")
		return
	}

	if *signIn {
		signInHandle()
		return
	}

	if *token == empty {
		fmt.Println("Warning: you have not defined a bearer token. ")
	}

	if *add != empty {
		addHandle()
		return
	}

	if *remove != empty {
		removeHandle()
		return
	}

	if *signUp {
		signUpHandle()
		return
	}

	if *change {
		changeHandle()
		return
	}

	if *delUser {
		deleteHandle()
		return
	}

	if *clear {
		clearHandler()
		return
	}
}

func main() {
	flag.Parse()

	handleArgs()
}
