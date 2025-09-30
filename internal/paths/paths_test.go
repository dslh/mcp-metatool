package paths

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetMetatoolDir(t *testing.T) {
	// Save original env var and restore after test
	originalDir := os.Getenv("MCP_METATOOL_DIR")
	defer func() {
		if originalDir != "" {
			os.Setenv("MCP_METATOOL_DIR", originalDir)
		} else {
			os.Unsetenv("MCP_METATOOL_DIR")
		}
	}()

	t.Run("returns default directory when env var not set", func(t *testing.T) {
		os.Unsetenv("MCP_METATOOL_DIR")

		dir, err := GetMetatoolDir()
		if err != nil {
			t.Fatalf("GetMetatoolDir() error = %v", err)
		}

		// Should be in user's home directory
		homeDir, _ := os.UserHomeDir()
		expectedDir := filepath.Join(homeDir, ".mcp-metatool")

		if dir != expectedDir {
			t.Errorf("GetMetatoolDir() = %v, want %v", dir, expectedDir)
		}

		// Directory should be created
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("GetMetatoolDir() did not create directory: %v", dir)
		}
	})

	t.Run("returns env var directory when set", func(t *testing.T) {
		// Use a temp directory for testing
		tempDir := t.TempDir()
		testDir := filepath.Join(tempDir, "custom-metatool")
		os.Setenv("MCP_METATOOL_DIR", testDir)

		dir, err := GetMetatoolDir()
		if err != nil {
			t.Fatalf("GetMetatoolDir() error = %v", err)
		}

		if dir != testDir {
			t.Errorf("GetMetatoolDir() = %v, want %v", dir, testDir)
		}

		// Directory should be created
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("GetMetatoolDir() did not create directory: %v", dir)
		}
	})

	t.Run("creates directory if it does not exist", func(t *testing.T) {
		tempDir := t.TempDir()
		testDir := filepath.Join(tempDir, "new-metatool-dir")
		os.Setenv("MCP_METATOOL_DIR", testDir)

		// Ensure directory doesn't exist yet
		os.RemoveAll(testDir)

		dir, err := GetMetatoolDir()
		if err != nil {
			t.Fatalf("GetMetatoolDir() error = %v", err)
		}

		// Verify directory was created
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("Directory was not created: %v", err)
		}

		if !info.IsDir() {
			t.Errorf("Path exists but is not a directory: %v", dir)
		}
	})
}

func TestGetToolsDir(t *testing.T) {
	// Save original env var and restore after test
	originalDir := os.Getenv("MCP_METATOOL_DIR")
	defer func() {
		if originalDir != "" {
			os.Setenv("MCP_METATOOL_DIR", originalDir)
		} else {
			os.Unsetenv("MCP_METATOOL_DIR")
		}
	}()

	t.Run("returns tools subdirectory", func(t *testing.T) {
		tempDir := t.TempDir()
		testDir := filepath.Join(tempDir, "test-metatool")
		os.Setenv("MCP_METATOOL_DIR", testDir)

		dir, err := GetToolsDir()
		if err != nil {
			t.Fatalf("GetToolsDir() error = %v", err)
		}

		expectedDir := filepath.Join(testDir, "tools")
		if dir != expectedDir {
			t.Errorf("GetToolsDir() = %v, want %v", dir, expectedDir)
		}

		// Verify directory was created
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("Tools directory was not created: %v", err)
		}

		if !info.IsDir() {
			t.Errorf("Path exists but is not a directory: %v", dir)
		}
	})

	t.Run("creates parent and tools directory", func(t *testing.T) {
		tempDir := t.TempDir()
		testDir := filepath.Join(tempDir, "new-metatool-dir")
		os.Setenv("MCP_METATOOL_DIR", testDir)

		// Ensure directories don't exist yet
		os.RemoveAll(testDir)

		dir, err := GetToolsDir()
		if err != nil {
			t.Fatalf("GetToolsDir() error = %v", err)
		}

		// Verify both parent and tools directories were created
		if _, err := os.Stat(testDir); os.IsNotExist(err) {
			t.Errorf("Parent directory was not created: %v", testDir)
		}

		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Tools directory was not created: %v", dir)
		}

		// Verify it ends with "tools"
		if !strings.HasSuffix(dir, "tools") {
			t.Errorf("GetToolsDir() should end with 'tools', got %v", dir)
		}
	})
}

func TestGetConfigPath(t *testing.T) {
	// Save original env var and restore after test
	originalDir := os.Getenv("MCP_METATOOL_DIR")
	defer func() {
		if originalDir != "" {
			os.Setenv("MCP_METATOOL_DIR", originalDir)
		} else {
			os.Unsetenv("MCP_METATOOL_DIR")
		}
	}()

	t.Run("returns correct config path", func(t *testing.T) {
		tempDir := t.TempDir()
		testDir := filepath.Join(tempDir, "test-metatool")
		os.Setenv("MCP_METATOOL_DIR", testDir)

		path, err := GetConfigPath()
		if err != nil {
			t.Fatalf("GetConfigPath() error = %v", err)
		}

		expectedPath := filepath.Join(testDir, "servers.json")
		if path != expectedPath {
			t.Errorf("GetConfigPath() = %v, want %v", path, expectedPath)
		}

		// Verify parent directory was created
		if _, err := os.Stat(testDir); os.IsNotExist(err) {
			t.Errorf("Parent directory was not created: %v", testDir)
		}
	})

	t.Run("config path ends with servers.json", func(t *testing.T) {
		os.Unsetenv("MCP_METATOOL_DIR")

		path, err := GetConfigPath()
		if err != nil {
			t.Fatalf("GetConfigPath() error = %v", err)
		}

		if !strings.HasSuffix(path, "servers.json") {
			t.Errorf("GetConfigPath() should end with 'servers.json', got %v", path)
		}
	})

	t.Run("config path is in metatool directory", func(t *testing.T) {
		tempDir := t.TempDir()
		testDir := filepath.Join(tempDir, "test-metatool")
		os.Setenv("MCP_METATOOL_DIR", testDir)

		configPath, err := GetConfigPath()
		if err != nil {
			t.Fatalf("GetConfigPath() error = %v", err)
		}

		metatoolDir, err := GetMetatoolDir()
		if err != nil {
			t.Fatalf("GetMetatoolDir() error = %v", err)
		}

		// Config path should be inside metatool directory
		if !strings.HasPrefix(configPath, metatoolDir) {
			t.Errorf("GetConfigPath() = %v should be inside %v", configPath, metatoolDir)
		}
	})
}

func TestPathsIntegration(t *testing.T) {
	// Save original env var and restore after test
	originalDir := os.Getenv("MCP_METATOOL_DIR")
	defer func() {
		if originalDir != "" {
			os.Setenv("MCP_METATOOL_DIR", originalDir)
		} else {
			os.Unsetenv("MCP_METATOOL_DIR")
		}
	}()

	t.Run("all paths use same base directory", func(t *testing.T) {
		tempDir := t.TempDir()
		testDir := filepath.Join(tempDir, "integration-test")
		os.Setenv("MCP_METATOOL_DIR", testDir)

		metatoolDir, err := GetMetatoolDir()
		if err != nil {
			t.Fatalf("GetMetatoolDir() error = %v", err)
		}

		toolsDir, err := GetToolsDir()
		if err != nil {
			t.Fatalf("GetToolsDir() error = %v", err)
		}

		configPath, err := GetConfigPath()
		if err != nil {
			t.Fatalf("GetConfigPath() error = %v", err)
		}

		// All paths should be under the same metatool directory
		if !strings.HasPrefix(toolsDir, metatoolDir) {
			t.Errorf("ToolsDir %v should be under MetatoolDir %v", toolsDir, metatoolDir)
		}

		if !strings.HasPrefix(configPath, metatoolDir) {
			t.Errorf("ConfigPath %v should be under MetatoolDir %v", configPath, metatoolDir)
		}

		// Verify structure
		expectedToolsDir := filepath.Join(metatoolDir, "tools")
		if toolsDir != expectedToolsDir {
			t.Errorf("ToolsDir = %v, want %v", toolsDir, expectedToolsDir)
		}

		expectedConfigPath := filepath.Join(metatoolDir, "servers.json")
		if configPath != expectedConfigPath {
			t.Errorf("ConfigPath = %v, want %v", configPath, expectedConfigPath)
		}
	})
}