package lb

import (
	"fmt"
	"github.com/pelageech/BDUTS/backend"
	"github.com/pelageech/BDUTS/config"
	"net/http"
	"os"
	"strconv"
)

func (lb *LoadBalancer) AddServer(rw http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		if err := req.ParseForm(); err != nil {
			http.Error(rw, "Bad Request", http.StatusBadRequest)
			return
		}
		url := req.FormValue("url")
		if lb.Pool().FindServerByUrl(url) != nil {
			http.Error(rw, "Server already exists", http.StatusPreconditionFailed)
			return
		}

		timeout, err := strconv.Atoi(req.FormValue("healthCheckTcpTimeout"))
		if err != nil {
			http.Error(rw, "Bad Request: numbers are only permitted", http.StatusBadRequest)
			return
		}
		if timeout <= 0 {
			http.Error(rw, "Bad Request: timeout is below zero or equal", http.StatusBadRequest)
			return
		}

		maxReq, err := strconv.Atoi(req.FormValue("maximalRequests"))
		if err != nil {
			http.Error(rw, "Bad Request: numbers are only permitted", http.StatusBadRequest)
			return
		}
		if maxReq <= 0 {
			http.Error(rw, "Bad Request: maxReq is below zero or equal", http.StatusBadRequest)
			return
		}
		maxReq %= 1 << 31 // int32

		server := config.ServerConfig{
			URL:                   url,
			HealthCheckTcpTimeout: int64(timeout),
			MaximalRequests:       int32(maxReq),
		}
		b := backend.NewBackendConfig(server)
		if b == nil {
			http.Error(rw, "Bad URL", http.StatusBadRequest)
			return
		}

		lb.pool.AddServer(b)
		lb.healthCheckFunc(b)
		_, _ = rw.Write([]byte("Success!"))
		rw.WriteHeader(http.StatusCreated)
	case http.MethodGet:
		http.ServeFile(rw, req, "views/add.html")
	default:
		http.Error(rw, "Only POST and GET requests are supported", http.StatusMethodNotAllowed)
	}
}

func (lb *LoadBalancer) RemoveServer(rw http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		if err := req.ParseForm(); err != nil {
			http.Error(rw, "Bad Request", http.StatusBadRequest)
			return
		}
		if err := req.ParseForm(); err != nil {
			http.Error(rw, "Bad Request", http.StatusBadRequest)
			return
		}
		url := req.FormValue("url")
		if err := lb.Pool().RemoveServerByUrl(url); err != nil {
			http.Error(rw, "Server doesn't exist", http.StatusNotFound)
			return
		}
		_, _ = rw.Write([]byte("Success!"))
	case http.MethodGet:
		http.ServeFile(rw, req, "views/remove.html")
	default:
		http.Error(rw, "Only POST and GET requests are supported", http.StatusMethodNotAllowed)
	}
}

func (lb *LoadBalancer) GetServers(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(rw, "Only GET requests are supported", http.StatusMethodNotAllowed)
		return
	}
	var b []byte
	header, _ := os.ReadFile("views/header.html")

	b = append(b, header...)
	footer, _ := os.ReadFile("views/footer.html")

	urls := lb.Pool().Servers()

	for k, v := range urls {
		b = append(b, []byte(
			fmt.Sprintf("<tr><td>%d</td><td>%s</td><td>%d</td><td>%t</td></tr>", k+1, v.URL(), v.HealthCheckTcpTimeout().Milliseconds(), v.Alive()),
		)...)
	}
	b = append(b, footer...)
	if _, err := rw.Write(b); err != nil {
		http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
