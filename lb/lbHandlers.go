package lb

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/pelageech/BDUTS/backend"
	"github.com/pelageech/BDUTS/cache"
	"github.com/pelageech/BDUTS/metrics"
	"github.com/pelageech/BDUTS/timer"
)

// LoadBalancerHandler is the main handler for load balancer
func (lb *LoadBalancer) LoadBalancerHandler(rw http.ResponseWriter, req *http.Request) {
	if err := timer.MakeRequestTimeTracker(lb.loadBalancerHandler, timer.SaveTimeFullTrip, true)(rw, req); err != nil {
		logger.Error("Unsuccessful request processing: ", "err", err)
	}
}

// LoadBalancerHandler is the main Handle func
func (lb *LoadBalancer) loadBalancerHandler(rw http.ResponseWriter, req *http.Request) error {
	if !isHTTPVersionSupported(req) {
		http.Error(rw, "Expected HTTP/1.1", http.StatusHTTPVersionNotSupported)
	}

	requestHash := lb.cacheProps.RequestHashKey(req)
	*req = *req.WithContext(context.WithValue(req.Context(), cache.Hash, requestHash))

	// getting a response from cache
	err := timer.MakeRequestTimeTracker(lb.getPageHandler, timer.SaveTimerDataGotFromCache, false)(rw, req)
	if err == nil {
		logger.Info("Transferred")
		metrics.GlobalMetrics.RequestsByCache.Inc()
		return nil
	} else {
		logger.Infof("Checking cache unsuccessful: %v", err)
		if r := req.Context().Value(cache.OnlyIfCachedKey).(bool); r {
			return cache.ErrOnlyIfCached
		}
	}

	// on cache miss make request to backend
	return lb.backendHandler(rw, req)
}

// uses balancer db for taking the page from cache and writing it to http.ResponseWriter
// if such a page is in cache
func (lb *LoadBalancer) getPageHandler(rw http.ResponseWriter, req *http.Request) error {
	if lb.cacheProps == nil {
		return errors.New("cache properties weren't set")
	}

	logger.Info("Trying to get a response from cache...")

	key := req.Context().Value(cache.Hash).([]byte)
	cacheItem, err := lb.cacheProps.GetPageFromCache(key, req)
	if err != nil {
		return err
	}
	logger.Info("Successfully got a response")

	for key, values := range cacheItem.Header {
		for _, value := range values {
			rw.Header().Add(key, value)
		}
	}

	_, err = rw.Write(cacheItem.Body)

	return err
}

func (lb *LoadBalancer) backendHandler(rw http.ResponseWriter, req *http.Request) error {
ChooseServer:
	server, err := lb.pool.GetNextPeer()
	if err != nil {
		return err
	}
	if ok := server.AssignRequest(); !ok {
		goto ChooseServer
	}

	metrics.GlobalMetrics.RequestsNow.Inc()
	defer metrics.GlobalMetrics.RequestsNow.Dec()
	defer metrics.GlobalMetrics.Requests.Inc()

	var resp *http.Response
	err = timer.MakeRequestTimeTracker(func(rw http.ResponseWriter, req *http.Request) error {
		var err error
		resp, err = server.SendRequestToBackend(req)
		server.Free()
		return err
	}, timer.SaveTimeDataBackend, false)(rw, req)

	// on cancellation
	if err == context.Canceled {
		return fmt.Errorf("[%s]: %w", server.URL(), err)
	} else if err != nil {
		logger.Errorf("[%s] %s", server.URL(), err)
		server.SetAlive(false) // СДЕЛАТЬ СЧЁТЧИК ИЛИ ПОЧИТАТЬ КАК У НДЖИНКС
		goto ChooseServer
	}

	logger.Infof("[%s] returned %s\n", server.URL(), resp.Status)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("[%s] %s", server.URL(), err)
		}
	}(resp.Body)

	byteArray, err := backend.WriteBodyAndReturn(rw, resp)
	if err != nil {
		return fmt.Errorf("[%s]: %w", server.URL(), err)
	}

	go lb.SaveToCache(req, resp, byteArray)
	return nil
}
