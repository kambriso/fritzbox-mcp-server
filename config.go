package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// config holds the FRITZ!Box connection configuration
type config struct {
	Host     string
	Port     int
	Username string
	Password string
	TLS      bool
}

// getConfigDir returns the config directory path
func getConfigDir() string {
	// Check XDG_CONFIG_HOME first
	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		return filepath.Join(configHome, "fritzbox-mcp-server")
	}
	// Fall back to ~/.config
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".config", "fritzbox-mcp-server")
	}
	return ""
}

// getCacheDir returns the cache directory path for storing XML descriptors
// This is exported so it can be used by cmd/fritz-mcp
func getCacheDir() string {
	// Check XDG_CACHE_HOME first (Linux/BSD)
	if cacheHome := os.Getenv("XDG_CACHE_HOME"); cacheHome != "" {
		return filepath.Join(cacheHome, "fritzbox-mcp-server")
	}
	// Fall back to ~/.cache (Linux/BSD)
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".cache", "fritzbox-mcp-server")
	}
	// Fallback to current directory if no home directory found
	return "xml"
}

// loadEnvFile reads a .env file and sets environment variables
func loadEnvFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if os.Getenv(key) == "" {
				os.Setenv(key, value)
			}
		}
	}
	return nil
}

// load reads configuration from .env file and environment variables
// Returns error if required fields are missing
// Looks for .env in:
// 1. Current directory (.env)
// 2. Global config directory (~/.config/fritzbox-mcp-server/.env)
func load() (*config, error) {
	// Try loading from current directory first (for development/testing)
	if err := loadEnvFile(".env"); err != nil {
		// If not found in current dir, try global config
		if configDir := getConfigDir(); configDir != "" {
			globalEnv := filepath.Join(configDir, ".env")
			_ = loadEnvFile(globalEnv)
		}
	}

	cfg := &config{
		Host:     os.Getenv("FRITZ_HOST"),
		Username: os.Getenv("FRITZ_USERNAME"),
		Password: os.Getenv("FRITZ_PASSWORD"),
	}

	// Parse port (default 49000)
	portStr := os.Getenv("FRITZ_PORT")
	if portStr == "" {
		cfg.Port = 49000
	} else {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid FRITZ_PORT: %w", err)
		}
		cfg.Port = port
	}

	// Parse TLS flag (default false)
	tlsStr := os.Getenv("FRITZ_TLS")
	if tlsStr != "" {
		tls, err := strconv.ParseBool(tlsStr)
		if err != nil {
			return nil, fmt.Errorf("invalid FRITZ_TLS: %w", err)
		}
		cfg.TLS = tls
	}

	// Validate required fields
	if cfg.Host == "" {
		return nil, fmt.Errorf("FRITZ_HOST is required")
	}
	if cfg.Username == "" {
		return nil, fmt.Errorf("FRITZ_USERNAME is required")
	}
	if cfg.Password == "" {
		return nil, fmt.Errorf("FRITZ_PASSWORD is required")
	}

	return cfg, nil
}

// baseURL returns the base URL for TR-064 requests
func (c *config) baseURL() string {
	scheme := "http"
	if c.TLS {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, c.Host, c.Port)
}
