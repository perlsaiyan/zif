package config

import (
	"os"
	"path/filepath"
)

// GetConfigDir returns the XDG config directory for zif.
// Uses $XDG_CONFIG_HOME/zif, or falls back to ~/.config/zif
func GetConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		// Fallback to ~/.config/zif
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(home, ".config")
	}
	
	zifDir := filepath.Join(configDir, "zif")
	return zifDir, nil
}

// GetGlobalModulesDir returns the path to the global modules directory.
func GetGlobalModulesDir() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "modules"), nil
}

// GetSessionDir returns the path to a session's configuration directory.
func GetSessionDir(sessionName string) (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "sessions", sessionName), nil
}

// GetSessionModulesDir returns the path to a session's modules directory.
func GetSessionModulesDir(sessionName string) (string, error) {
	sessionDir, err := GetSessionDir(sessionName)
	if err != nil {
		return "", err
	}
	return filepath.Join(sessionDir, "modules"), nil
}

// EnsureConfigDirs creates the necessary directory structure if it doesn't exist.
func EnsureConfigDirs() error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}
	
	// Create main config directory
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	
	// Create global modules directory
	globalModulesDir, err := GetGlobalModulesDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(globalModulesDir, 0755); err != nil {
		return err
	}
	
	// Create sessions directory
	sessionsDir := filepath.Join(configDir, "sessions")
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return err
	}
	
	return nil
}

