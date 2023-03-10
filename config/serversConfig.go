package config

import (
	"encoding/json"
	"io"
	"os"
)

type ServersReader struct {
	file *os.File
}

type ServerConfig struct {
	URL                   string
	HealthCheckTcpTimeout int64
	MaximalRequests       int32
}

func NewServersReader(serversPath string) (*ServersReader, error) {
	file, err := os.Open(serversPath)
	if err != nil {
		return nil, err
	}
	return &ServersReader{file}, nil
}

func (r *ServersReader) Close() error {
	return r.file.Close()
}

func (r *ServersReader) ReadServersConfig() ([]ServerConfig, error) {
	serversFileByte, err := io.ReadAll(r.file)
	if err != nil {
		return nil, err
	}

	var SeversConfigs []ServerConfig
	err = json.Unmarshal(serversFileByte, &SeversConfigs)
	if err != nil {
		return nil, err
	}

	return SeversConfigs, nil
}
