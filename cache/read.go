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
func (p *CachingProperties) GetPageFromCache(key []byte, req *http.Request) (*Page, error) {
	var info *PageMetadata
	var page *Page
	var err error

	requestDirectives := loadRequestDirectives(req.Header)
	// doesn't modify the request but adds a context key-value item
	*req = *req.WithContext(context.WithValue(req.Context(), OnlyIfCachedKey, requestDirectives.OnlyIfCached))

	if info, err = p.getPageMetadata(key); err != nil {
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

	if page, err = readPageFromDisk(key); err != nil {
		return nil, err
	}

	return page, nil
}

// Accesses the database to get meta information about the cache.
func (p *CachingProperties) getPageMetadata(key []byte) (*PageMetadata, error) {
	var result []byte = nil

	err := p.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(key)
		if b == nil {
			return errors.New("missed cache")
		}

		result = b.Get([]byte(pageMetadataKey))
		if result == nil {
			return errors.New("no record in cache")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	var meta PageMetadata
	if err = json.Unmarshal(result, &meta); err != nil {
		return nil, err
	}

	go func() {
		newMeta := meta
		newMeta.Uses++

		// there won't be an error, because the key is correct
		_ = p.insertPageMetadataToDB(key, &newMeta)
	}()

	return &meta, nil
}

// Reads a page from disk
func readPageFromDisk(key []byte) (*Page, error) {
	subhashLength := hashLength / subHashCount

	var subHashes [][]byte
	for i := 0; i < subHashCount; i++ {
		subHashes = append(subHashes, key[i*subhashLength:(i+1)*subhashLength])
	}

	path := PagesPath
	for _, v := range subHashes {
		path += "/" + string(v)
	}
	path += "/" + string(key[:])

	bytes, err := os.ReadFile(path)

	var page Page
	if err := json.Unmarshal(bytes, &page); err != nil {
		return nil, err
	}
	return &page, err
}
