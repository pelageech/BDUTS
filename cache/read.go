package cache

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/boltdb/bolt"
)

// GetPageFromCache gets corresponding page and its metadata
// and returns it if it exists. Uses some parameters for building
// a request key, see in cache package and cacheConfig file
func GetPageFromCache(db *bolt.DB, req *http.Request) (*Page, error) {
	var info *PageMetadata
	var item Page
	var err error

	requestDirectives := loadRequestDirectives(req.Header)
	// doesn't modify the request but adds a context key-value item
	*req = *req.WithContext(context.WithValue(req.Context(), OnlyIfCachedKey, requestDirectives.OnlyIfCached))

	keyString := constructKeyFromRequest(req)
	requestHash := hash([]byte(keyString))

	if info, err = getPageMetadata(db, requestHash); err != nil {
		return nil, err
	}

	afterDeath := time.Duration(requestDirectives.MaxStale)

	maxAge := info.ResponseDirectives.MaxAge
	if !info.ResponseDirectives.SMaxAge.IsZero() {
		maxAge = info.ResponseDirectives.SMaxAge
	}

	freshTime := requestDirectives.MinFresh

	if info.ResponseDirectives.MustRevalidate {
		afterDeath = time.Duration(0)
	}
	if time.Now().After(maxAge.Add(afterDeath)) || !time.Now().After(freshTime) {
		return nil, errors.New("not fresh")
	}

	var byteItem []byte
	if byteItem, err = readPageFromDisk(requestHash); err != nil {
		return nil, err
	}

	if err = json.Unmarshal(byteItem, &item); err != nil {
		return nil, err
	}

	return &item, nil
}

// Accesses the database to get meta information about the cache.
func getPageMetadata(db *bolt.DB, requestHash []byte) (*PageMetadata, error) {
	var result []byte = nil

	err := db.View(func(tx *bolt.Tx) error {
		treeBucket, err := getBucket(tx, requestHash)
		if err != nil {
			return err
		}

		result = treeBucket.Get([]byte(pageInfo))
		if result == nil {
			return errors.New("no record in cache")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	var info PageMetadata
	if err = json.Unmarshal(result, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// Reads a page from disk
func readPageFromDisk(requestHash []byte) ([]byte, error) {
	subhashLength := hashLength / subHashCount

	var subHashes [][]byte
	for i := 0; i < subHashCount; i++ {
		subHashes = append(subHashes, requestHash[i*subhashLength:(i+1)*subhashLength])
	}

	path := PagesPath
	for _, v := range subHashes {
		path += "/" + string(v)
	}
	path += "/" + string(requestHash[:])

	bytes, err := os.ReadFile(path)
	return bytes, err
}

// a universal function for getting a bucket
func getBucket(tx *bolt.Tx, key []byte) (*bolt.Bucket, error) {
	if bucket := tx.Bucket(key); bucket != nil {
		return bucket, nil
	}

	return nil, errors.New("miss cache")
}
