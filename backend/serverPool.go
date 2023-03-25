package backend

import (
	"errors"
	"sync"
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

func (p *ServerPool) GetCurrentServer() *Backend {
	return p.servers[p.current]
}

func (p *ServerPool) AddServer(b *Backend) {
	p.servers = append(p.servers, b)
}

func (p *ServerPool) GetNextPeer() (*Backend, error) {
	serverList := p.Servers()

	p.Lock()
	defer p.Unlock()

	for i := 0; i < len(serverList); i++ {
		p.IncrementCurrent()
		if serverList[p.Current()].Alive {
			return p.GetCurrentServer(), nil
		}
	}

	return nil, errors.New("all backends are turned down")
}
