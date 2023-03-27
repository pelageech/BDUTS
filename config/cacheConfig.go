package config

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type CacheReader struct {
	file *os.File
}

type cachePairLocationRequestKey struct {
	Location   string
	RequestKey string
}

type CacheConfig struct {
	pairs []cachePairLocationRequestKey
}

func (c *CacheConfig) Pairs() []cachePairLocationRequestKey {
	return c.pairs
}

func NewCacheReader(configPath string) (*CacheReader, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	return &CacheReader{file}, nil
}

func (r *CacheReader) Close() error {
	return r.file.Close()
}

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
