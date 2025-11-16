package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// SessionsConfig represents the top-level configuration for auto-start sessions
type SessionsConfig struct {
	Sessions []SessionConfig `yaml:"sessions"`
}

// SessionConfig represents a single session configuration
type SessionConfig struct {
	Name      string `yaml:"name"`
	Address   string `yaml:"address"`
	Autostart bool   `yaml:"autostart"`
}

// GetSessionsConfigPath returns the path to sessions.yaml in the XDG config directory
func GetSessionsConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "sessions.yaml"), nil
}

// SaveDefaultSessionsConfig creates a blank sessions.yaml file with an empty sessions array
func SaveDefaultSessionsConfig() error {
	configPath, err := GetSessionsConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %v", err)
	}

	// Ensure config directory exists
	if err := EnsureConfigDirs(); err != nil {
		return fmt.Errorf("failed to ensure config directories: %v", err)
	}

	// Create default config with empty sessions
	defaultConfig := SessionsConfig{
		Sessions: []SessionConfig{},
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(&defaultConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal default config: %v", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write default config: %v", err)
	}

	return nil
}

// LoadSessionsConfig reads and parses sessions.yaml, creating a default file if it doesn't exist
func LoadSessionsConfig() (*SessionsConfig, error) {
	configPath, err := GetSessionsConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get config path: %v", err)
	}

	// Check if file exists
	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		// File doesn't exist, create default
		if err := SaveDefaultSessionsConfig(); err != nil {
			return nil, fmt.Errorf("failed to create default config: %v", err)
		}
		// Return empty config
		return &SessionsConfig{Sessions: []SessionConfig{}}, nil
	}

	// Read existing file
	yamlFile, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	// Unmarshal YAML
	var config SessionsConfig
	if err := yaml.Unmarshal(yamlFile, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	return &config, nil
}

