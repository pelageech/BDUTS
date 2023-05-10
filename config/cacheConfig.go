package config

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

// CacheReader is a struct for reading cache config.
type CacheReader struct {
	file *os.File
}

type cachePairLocationRequestKey struct {
	Location   string
	RequestKey string
}

// CacheConfig is a struct for cache config.
type CacheConfig struct {
	pairs []cachePairLocationRequestKey
}

// Pairs is a method for getting pairs from cache config.
func (c *CacheConfig) Pairs() []cachePairLocationRequestKey {
	return c.pairs
}

// NewCacheReader is a constructor for CacheReader.
func NewCacheReader(configPath string) (*CacheReader, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	return &CacheReader{file}, nil
}

// Close is a method for closing cache config file.
func (r *CacheReader) Close() error {
	return r.file.Close()
}

// ReadCacheConfig reads cache config.
func ReadCacheConfig(r *CacheReader) (*CacheConfig, error) {
	cacheFileByte, err := io.ReadAll(r.file)
	if err != nil {
		return nil, err
	}

	var cacheConfig CacheConfig
	err = json.Unmarshal(cacheFileByte, &cacheConfig.pairs)
	if err != nil {
		return nil, err
	}
	return &cacheConfig, nil
}

// ParseRequestKey parses request key.
func ParseRequestKey(requestKey string) (result []func(r *http.Request) string) {
	if len(requestKey) == 0 {
		log.Panic("An empty line was got")
	}

	keys := strings.Split(requestKey, ";")
	for _, v := range keys {
		var m func(r *http.Request) string
		switch v {
		case "REQ_METHOD":
			m = func(r *http.Request) string { return r.Method }
		case "REQ_HOST":
			m = func(r *http.Request) string { return r.Host }
		case "REQ_URI":
			m = func(r *http.Request) string { return r.URL.Path }
		case "REQ_QUERY":
			m = func(r *http.Request) string { return r.URL.RawQuery }
		default:
			continue
		}
		result = append(result, m)
	}
	return
}
