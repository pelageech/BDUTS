package backend

import (
	"errors"
	"sync"
)

type ServerPool struct {
	Mux     sync.Mutex
	Servers []*Backend
	Current int32
}

func (serverPool *ServerPool) GetNextPeer() (*Backend, error) {
	serverList := serverPool.Servers

	serverPool.Mux.Lock()
	defer serverPool.Mux.Unlock()

	for i := 0; i < len(serverList); i++ {
		serverPool.Current++
		if serverPool.Current == int32(len(serverList)) {
			serverPool.Current = 0
		}
		if serverList[serverPool.Current].Alive {
			return serverList[serverPool.Current], nil
		}
	}

	return nil, errors.New("all backends are turned down")
}
