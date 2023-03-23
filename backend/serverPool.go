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

func (serverPool *ServerPool) Lock() {
	serverPool.mux.Lock()
}

func (serverPool *ServerPool) Unlock() {
	serverPool.mux.Unlock()
}

func (serverPool *ServerPool) Servers() []*Backend {
	return serverPool.servers
}

func (serverPool *ServerPool) Current() int32 {
	return serverPool.current
}

func (serverPool *ServerPool) IncrementCurrent() {
	serverPool.current++
	if serverPool.current == int32(len(serverPool.Servers())) {
		serverPool.current = 0
	}
}

func (serverPool *ServerPool) GetCurrentServer() *Backend {
	return serverPool.servers[serverPool.current]
}

func (serverPool *ServerPool) AddServer(b *Backend) {
	serverPool.servers = append(serverPool.servers, b)
}

func (serverPool *ServerPool) GetNextPeer() (*Backend, error) {
	serverList := serverPool.Servers()

	serverPool.Lock()
	defer serverPool.Unlock()

	for i := 0; i < len(serverList); i++ {
		serverPool.IncrementCurrent()
		if serverList[serverPool.Current()].Alive {
			return serverPool.GetCurrentServer(), nil
		}
	}

	return nil, errors.New("all backends are turned down")
}
