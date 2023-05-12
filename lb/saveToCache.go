package lb

import (
	"net/http"

	"github.com/pelageech/BDUTS/cache"
)

// SaveToCache takes all the necessary information about a response and saves it
// in cache.
func (lb *LoadBalancer) SaveToCache(req *http.Request, resp *http.Response, byteArray []byte) {
	if !(resp.StatusCode >= 200 && resp.StatusCode < 400) {
		return
	}
	logger.Info("Saving response in cache")

	go func() {
		cacheItem := &cache.Page{
			Body:   byteArray,
			Header: resp.Header,
		}

		key, ok := req.Context().Value(cache.Hash).([]byte)
		if !ok {
			logger.Errorf("Couldn't get a hash from request context")
			return
		}
		if err := lb.cacheProps.InsertPageInCache(key, req, resp, cacheItem); err != nil {
			logger.Errorf("Unsuccessful saving: %v", err)
			return
		}
		logger.Info("Successfully saved")
	}()
}
