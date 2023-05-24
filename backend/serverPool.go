package backend

import (
	"errors"
	"os"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/pelageech/BDUTS/config"
)

const initCurrentBackend = -1

var logger = log.NewWithOptions(os.Stderr, log.Options{
	ReportTimestamp: true,
	ReportCaller:    true,
})

func LoggerConfig(prefix string) {
	logger.SetPrefix(prefix)
}

// ServerPool is a struct that contains all the configuration
// of the backend servers.
type ServerPool struct {
	mux     sync.Mutex
	servers []*Backend
	current int32
}

// NewServerPool creates a new ServerPool.
func NewServerPool() *ServerPool {
	var s []*Backend
	return &ServerPool{
		mux:     sync.Mutex{},
		servers: s,
		current: initCurrentBackend,
	}
}

// ConfigureServerPool creates a new ServerPool from config.ServerConfig.
func (p *ServerPool) ConfigureServerPool(servers []config.ServerConfig) {
	for _, server := range servers {
		if b := NewBackendConfig(server); b != nil {
			p.AddServer(b)
		}
	}
}

// Lock is used to lock the server pool.
func (p *ServerPool) Lock() {
	p.mux.Lock()
}

// Unlock is used to unlock the server pool.
func (p *ServerPool) Unlock() {
	p.mux.Unlock()
}

// Servers returns the servers of the server pool.
func (p *ServerPool) Servers() []*Backend {
	return p.servers
}

// Current returns the index of the current server.
func (p *ServerPool) Current() int32 {
	return p.current
}

// IncrementCurrent increments the current server index.
func (p *ServerPool) IncrementCurrent() {
	p.current++
	if p.current >= int32(len(p.Servers())) {
		p.current = 0
	}
}

// DecrementCurrent decrements the current server index.
func (p *ServerPool) DecrementCurrent() {
	p.current--
	if p.current <= 0 {
		p.current = int32(len(p.Servers())) - 1
	}
}

// GetCurrentServer returns the current server.
func (p *ServerPool) GetCurrentServer() *Backend {
	return p.servers[p.current]
}

// AddServer adds a new server to the server pool.
func (p *ServerPool) AddServer(b *Backend) {
	p.Lock()
	defer p.Unlock()
	logger.Infof("Adding server: %s\n", b.URL().String())
	p.servers = append(p.servers, b)
}

// FindServerByUrl finds a server by its URL.
func (p *ServerPool) FindServerByUrl(url string) *Backend {
	for _, v := range p.servers {
		if v.URL().String() == url {
			return v
		}
	}
	return nil
}

// RemoveServerByUrl removes a server by its URL.
func (p *ServerPool) RemoveServerByUrl(url string) error {
	p.Lock()
	defer p.Unlock()
	for k, v := range p.servers {
		if v.URL().String() == url {
			p.servers = append(p.servers[:k], p.servers[k+1:]...)
			logger.Infof("[%s] removed from server pool\n", url)
			return nil
		}
	}
	return errors.New("server not found")
}

// GetNextPeer returns the next server in the server pool.
func (p *ServerPool) GetNextPeer() (*Backend, error) {
	p.Lock()
	defer p.Unlock()

	serverList := p.Servers()
	for i := 0; i < len(serverList); i++ {
		p.IncrementCurrent()
		if serverList[p.Current()].alive {
			return p.GetCurrentServer(), nil
		}
	}

	return nil, errors.New("all backends are turned down")
}

// ServersURLs returns the URLs of the servers in the server pool.
func (p *ServerPool) ServersURLs() []string {
	urls := make([]string, 0, len(p.Servers()))
	for _, v := range p.Servers() {
		urls = append(urls, v.URL().String())
	}
	return urls
}
