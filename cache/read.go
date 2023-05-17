package cache

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/boltdb/bolt"
)

// GetPageFromCache gets corresponding page and its metadata
// and returns it if it exists. Uses some parameters for building
// a request key, see in cache package and cacheConfig file.
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
		if errors.Is(err, os.ErrNotExist) {
			_, _ = p.removePageMetadata(key)
		}
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

	_ = p.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(key)
		bs := b.Get([]byte(usesKey))
		if bs == nil {
			return errors.New("value ot found")
		}

		incr := make([]byte, sizeOfInt32)
		binary.LittleEndian.PutUint32(incr, binary.LittleEndian.Uint32(bs)+uint32(1))
		_ = b.Put([]byte(usesKey), incr)
		return nil
	})

	var meta PageMetadata
	if err = json.Unmarshal(result, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

// Reads a page from disk.
func readPageFromDisk(key []byte) (*Page, error) {
	path := makePath(key, subHashCount)
	path += "/" + string(key)

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	r := bufio.NewReaderSize(file, bufferSize)
	bytesPage, _ := r.ReadBytes(byte(0))

	var page Page
	if err := json.Unmarshal(bytesPage, &page); err != nil {
		return nil, err
	}
	return &page, err
}
