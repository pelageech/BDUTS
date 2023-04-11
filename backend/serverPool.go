package backend

import (
	"errors"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/pelageech/BDUTS/config"
)

type ServerPool struct {
	mux     sync.Mutex
	servers []*Backend
	current int32
}

func NewServerPool() *ServerPool {
	var s []*Backend
	return &ServerPool{
		mux:     sync.Mutex{},
		servers: s,
		current: -1,
	}
}

func (p *ServerPool) CreateBackend(server config.ServerConfig) *Backend {
	parsed, err := url.Parse(server.URL)
	if err != nil {
		log.Printf("Failed to parse server URL: %s\n", err)
		return nil
	}

	u := parsed
	h := time.Duration(server.HealthCheckTcpTimeout) * time.Millisecond
	max := server.MaximalRequests
	b := NewBackend(u, h, max)

	return b
}

func (p *ServerPool) ConfigureServerPool(servers []config.ServerConfig) {
	for _, server := range servers {
		log.Printf("%v", server)
		if b := p.CreateBackend(server); b != nil {
			p.AddServer(b)
		}
	}
}

func (p *ServerPool) Lock() {
	p.mux.Lock()
}

func (p *ServerPool) Unlock() {
	p.mux.Unlock()
}

func (p *ServerPool) Servers() []*Backend {
	return p.servers
}

func (p *ServerPool) Current() int32 {
	return p.current
}

func (p *ServerPool) IncrementCurrent() {
	p.current++
	if p.current == int32(len(p.Servers())) {
		p.current = 0
	}
}

func (p *ServerPool) DecrementCurrent() {
	p.current--
	if p.current == 0 {
		p.current = int32(len(p.Servers())) - 1
	}
}

func (p *ServerPool) GetCurrentServer() *Backend {
	return p.servers[p.current]
}

func (p *ServerPool) AddServer(b *Backend) {
	p.Lock()
	defer p.Unlock()
	log.Printf("Adding server: %s\n", b.URL().String())
	p.servers = append(p.servers, b)
}

func (p *ServerPool) RemoveServerByUrl(url *url.URL) error {
	backends := p.servers

	for k, v := range backends {
		if v.URL().String() == url.String() {
			p.servers = append(p.servers[:k], p.servers[k+1:]...)
			log.Printf("[%s] removed from server pool\n", url)
			return nil
		}
	}
	return errors.New("server not found")
}

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
