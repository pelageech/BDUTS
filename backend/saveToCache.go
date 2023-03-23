package backend

import (
	"log"
	"net/http"

	"github.com/boltdb/bolt"
	"github.com/pelageech/BDUTS/cache"
)

// SaveToCache takes all the necessary information about a response and saves it
// in cache
func SaveToCache(db *bolt.DB, req *http.Request, resp *http.Response, byteArray []byte) {
	if !(resp.StatusCode >= 200 && resp.StatusCode < 400) {
		return
	}
	log.Println("Saving response in cache")

	go func() {
		cacheItem := &cache.Page{
			Body:   byteArray,
			Header: resp.Header,
		}
		err := cache.InsertPageInCache(db, req, resp, cacheItem)
		if err != nil {
			log.Println("Unsuccessful operation: ", err)
			return
		}
		log.Println("Successfully saved")
	}()
}
