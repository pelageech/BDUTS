package config

import (
	"encoding/json"
	"io"
	"os"
)

// ServersReader is a struct for reading servers config.
type ServersReader struct {
	file *os.File
}

// ServerConfig is a struct for server config.
type ServerConfig struct {
	URL                   string
	HealthCheckTcpTimeout int64
	MaximalRequests       int32
}

// NewServersReader is a constructor for ServersReader.
func NewServersReader(serversPath string) (*ServersReader, error) {
	file, err := os.Open(serversPath)
	if err != nil {
		return nil, err
	}
	return &ServersReader{file}, nil
}

// Close is a method for closing servers config file.
func (r *ServersReader) Close() error {
	return r.file.Close()
}

// ReadServersConfig reads servers config.
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
