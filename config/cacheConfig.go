package config

import (
	"encoding/json"
	"io"
	"os"
)

type CacheReader struct {
	file *os.File
}

type CacheConfig struct {
	requestKey string
}

func NewCacheReader(configPath string) (*CacheReader, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	return &CacheReader{file}, nil
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
