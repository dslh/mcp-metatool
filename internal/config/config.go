package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// MCPServerConfig represents a single MCP server configuration
type MCPServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Hidden  bool              `json:"hidden,omitempty"`
}

// Config represents the full metatool configuration
type Config struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

// GetMetatoolDirectory returns the directory where metatool files are stored
func GetMetatoolDirectory() (string, error) {
	var metatoolDir string
	
	// Check for environment variable override first
	if envDir := os.Getenv("MCP_METATOOL_DIR"); envDir != "" {
		metatoolDir = envDir
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		metatoolDir = filepath.Join(homeDir, ".mcp-metatool")
	}
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(metatoolDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create metatool directory: %w", err)
	}
	
	return metatoolDir, nil
}

// LoadConfig loads and parses the MCP configuration file
func LoadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	// Expand environment variables
	if err := expandEnvVars(&config); err != nil {
		return nil, fmt.Errorf("failed to expand environment variables: %w", err)
	}

	return &config, nil
}

// LoadDefaultConfig loads the configuration from the default location
func LoadDefaultConfig() (*Config, error) {
	metatoolDir, err := GetMetatoolDirectory()
	if err != nil {
		return nil, err
	}
	
	configPath := filepath.Join(metatoolDir, "servers.json")
	return LoadConfig(configPath)
}

// expandEnvVars performs ${VAR} expansion on all string values in the config
func expandEnvVars(config *Config) error {
	for serverName, serverConfig := range config.MCPServers {
		// Expand command
		expanded, err := expandString(serverConfig.Command)
		if err != nil {
			return fmt.Errorf("error expanding command for server %s: %w", serverName, err)
		}
		serverConfig.Command = expanded

		// Expand args
		for i, arg := range serverConfig.Args {
			expanded, err := expandString(arg)
			if err != nil {
				return fmt.Errorf("error expanding arg %d for server %s: %w", i, serverName, err)
			}
			serverConfig.Args[i] = expanded
		}

		// Expand env values
		for key, value := range serverConfig.Env {
			expanded, err := expandString(value)
			if err != nil {
				return fmt.Errorf("error expanding env var %s for server %s: %w", key, serverName, err)
			}
			serverConfig.Env[key] = expanded
		}

		// Update the config with expanded values
		config.MCPServers[serverName] = serverConfig
	}

	return nil
}

// envVarPattern matches ${VAR_NAME} patterns
var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// expandString expands ${VAR} environment variable references in a string
func expandString(s string) (string, error) {
	return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		// Extract variable name (remove ${ and })
		varName := match[2 : len(match)-1]
		
		// Get environment variable value
		value := os.Getenv(varName)
		if value == "" {
			// For now, we'll allow empty values but this could be made configurable
			return ""
		}
		
		return value
	}), nil
}

// ShouldHideProxiedTools returns true if proxied tools should be hidden globally
func ShouldHideProxiedTools() bool {
	return os.Getenv("MCP_METATOOL_HIDE_PROXIED_TOOLS") != ""
}

// Validate checks the configuration for basic validity
func (c *Config) Validate() error {
	if len(c.MCPServers) == 0 {
		return fmt.Errorf("no MCP servers configured")
	}

	for serverName, serverConfig := range c.MCPServers {
		if strings.TrimSpace(serverConfig.Command) == "" {
			return fmt.Errorf("server %s has empty command", serverName)
		}
	}

	return nil
}