package config

import (
	"encoding/json"
	"io"
	"os"
)

// LoadBalancerReader is a struct for reading load balancer config.
type LoadBalancerReader struct {
	file *os.File
}

// LoadBalancerConfig is a struct for load balancer config.
type LoadBalancerConfig struct {
	Port              int
	HealthCheckPeriod int64
	MaxCacheSize      int64
	ObserveFrequency  int64
}

// NewLoadBalancerReader is a constructor for LoadBalancerReader.
func NewLoadBalancerReader(configPath string) (*LoadBalancerReader, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	return &LoadBalancerReader{file}, nil
}

// Close is a method for closing load balancer config file.
func (r *LoadBalancerReader) Close() error {
	return r.file.Close()
}

// ReadLoadBalancerConfig reads load balancer config.
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
