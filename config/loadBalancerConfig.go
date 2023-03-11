package config

import (
	"encoding/json"
	"io"
	"os"
)

type LoadBalancerReader struct {
	file *os.File
}

type LoadBalancerConfig struct {
	Port              int
	HealthCheckPeriod int64
}

func NewLoadBalancerReader(configPath string) (*LoadBalancerReader, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	return &LoadBalancerReader{file}, nil
}

func (r *LoadBalancerReader) Close() error {
	return r.file.Close()
}

func (r *LoadBalancerReader) ReadLoadBalancerConfig() (*LoadBalancerConfig, error) {
	configFileByte, err := io.ReadAll(r.file)
	if err != nil {
		return nil, err
	}

	var config LoadBalancerConfig
	err = json.Unmarshal(configFileByte, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
