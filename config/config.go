package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config holds the configuration for a node
type Config struct {
	NodeID        string   `json:"node_id"`
	ListenAddress string   `json:"listen_address"`
	Peers         []string `json:"peers"`
	GossipInterval int     `json:"gossip_interval_ms"`
	HealthCheckInterval int `json:"health_check_interval_ms"`
	RequestTimeout int    `json:"request_timeout_ms"`
}

// LoadConfig loads configuration from a file
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	
	// Set defaults if not provided
	if config.GossipInterval == 0 {
		config.GossipInterval = 1000 // 1 second
	}
	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = 5000 // 5 seconds
	}
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 5000 // 5 seconds
	}
	
	return &config, nil
}

// SaveConfig saves configuration to a file
func SaveConfig(config *Config, filename string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// DefaultConfig returns a default configuration
func DefaultConfig(nodeID, listenAddress string) *Config {
	return &Config{
		NodeID:        nodeID,
		ListenAddress: listenAddress,
		Peers:         []string{},
		GossipInterval: 1000,
		HealthCheckInterval: 5000,
		RequestTimeout: 5000,
	}
}
