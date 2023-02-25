package config

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
)

type CacheReader struct {
	file *os.File
}

type CacheConfig struct {
	RequestKey string
}

var RequestKey []func(r *http.Request) string

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
	err = json.Unmarshal(cacheFileByte, &cacheConfig)
	if err != nil {
		return nil, err
	}

	return &cacheConfig, nil
}

func ParseRequestKey(requestKey string) (result []func(r *http.Request) string) {
	if len(requestKey) == 0 {
		return nil
	}

	keys := strings.Split(requestKey, ";")
	for _, v := range keys {
		var m func(r *http.Request) string
		switch v {
		case "REQ_METHOD":
			m = func(r *http.Request) string { return r.Method }
			break
		case "REQ_HOST":
			m = func(r *http.Request) string { return r.Host }
			break
		case "REQ_URI":
			m = func(r *http.Request) string { return r.URL.Path }
			break
		}
		result = append(result, m)
	}
	return
}
