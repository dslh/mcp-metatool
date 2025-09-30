package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetMetatoolDir returns the directory where metatool files are stored
// It checks MCP_METATOOL_DIR environment variable first, then falls back to ~/.mcp-metatool
func GetMetatoolDir() (string, error) {
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

// GetToolsDir returns the directory where saved tool definitions are stored
func GetToolsDir() (string, error) {
	metatoolDir, err := GetMetatoolDir()
	if err != nil {
		return "", err
	}

	toolsDir := filepath.Join(metatoolDir, "tools")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(toolsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create tools directory: %w", err)
	}

	return toolsDir, nil
}

// GetConfigPath returns the full path to the servers.json configuration file
func GetConfigPath() (string, error) {
	metatoolDir, err := GetMetatoolDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(metatoolDir, "servers.json"), nil
}