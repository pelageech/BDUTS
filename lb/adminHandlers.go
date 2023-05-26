package lb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/pelageech/BDUTS/backend"
	"github.com/pelageech/BDUTS/config"
)

const int32BitsAmount = 31 // int32, not uint32

// AddForm is a structure which is parsed from a POST-request
// processed by AddServerHandler.
type AddForm struct {
	Url                   string
	HealthCheckTcpTimeout int
	MaximalRequests       int
}

// RemoveForm is a structure which is parsed from a POST-request
// processed by RemoveServerHandlerRemoveServer.
type RemoveForm struct {
	Url string
}

// AddServerHandler handles adding a new backend into the server pool of the LoadBalancer.
func (lb *LoadBalancer) AddServerHandler(rw http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:

		add := AddForm{}
		if err := json.NewDecoder(req.Body).Decode(&add); err != nil {
			http.Error(rw, "Couldn't parse JSON", http.StatusBadRequest)
		}

		if lb.Pool().FindServerByUrl(add.Url) != nil {
			http.Error(rw, "Server already exists", http.StatusPreconditionFailed)
			return
		}

		if add.HealthCheckTcpTimeout <= 0 {
			http.Error(rw, "Bad Request: timeout is below zero or equal", http.StatusBadRequest)
			return
		}

		if add.MaximalRequests <= 0 {
			http.Error(rw, "Bad Request: maxReq is below zero or equal", http.StatusBadRequest)
			return
		}
		add.MaximalRequests %= 1 << int32BitsAmount

		server := config.ServerConfig{
			URL:                   add.Url,
			HealthCheckTcpTimeout: int64(add.HealthCheckTcpTimeout),
			MaximalRequests:       int32(add.MaximalRequests),
		}
		b := backend.NewBackendConfig(server)
		if b == nil {
			http.Error(rw, "Bad URL", http.StatusBadRequest)
			return
		}

		lb.pool.AddServer(b)
		lb.healthCheckFunc(b)
		_, _ = rw.Write([]byte("Success!"))
	case http.MethodGet:
		http.ServeFile(rw, req, "views/add.html")
	default:
		http.Error(rw, "Only POST and GET requests are supported", http.StatusMethodNotAllowed)
	}
}

// RemoveServerHandler handles removing a backend from the server pool of the LoadBalancer.
func (lb *LoadBalancer) RemoveServerHandler(rw http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodDelete:
		rem := RemoveForm{}

		if err := json.NewDecoder(req.Body).Decode(&rem); err != nil {
			http.Error(rw, "Couldn't parse JSON", http.StatusBadRequest)
		}

		if err := lb.Pool().RemoveServerByUrl(rem.Url); err != nil {
			http.Error(rw, "Server doesn't exist", http.StatusNotFound)
			return
		}
		_, _ = rw.Write([]byte("Success!"))
	case http.MethodGet:
		http.ServeFile(rw, req, "views/remove.html")
	default:
		http.Error(rw, "Only DELETE and GET requests are supported", http.StatusMethodNotAllowed)
	}
}

// GetServersHandler takes all the information about the backends from the server pool and puts
// an HTML page to http.ResponseWriter with the info in <table>...</table> tags.
func (lb *LoadBalancer) GetServersHandler(rw http.ResponseWriter, req *http.Request) {
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
			fmt.Sprintf("<tr>"+
				"<td>%d</td>"+
				"<td>%s</td>"+
				"<td>%d</td>"+
				"<td>%t</td>"+
				"</tr>", k+1, v.URL(), v.HealthCheckTcpTimeout().Milliseconds(), v.Alive()),
		)...)
	}
	b = append(b, footer...)
	if _, err := rw.Write(b); err != nil {
		http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}