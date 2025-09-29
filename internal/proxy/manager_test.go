package proxy

import (
	"testing"

	"github.com/dslh/mcp-metatool/internal/config"
)

func TestNewManager(t *testing.T) {
	cfg := &config.Config{
		MCPServers: map[string]config.MCPServerConfig{
			"test": {
				Command: "echo",
				Args:    []string{"hello"},
			},
		},
	}

	manager := NewManager(cfg)
	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	if manager.config != cfg {
		t.Error("Manager config not set correctly")
	}

	if len(manager.clients) != 0 {
		t.Error("Manager should start with no clients")
	}

	if len(manager.sessions) != 0 {
		t.Error("Manager should start with no sessions")
	}

	if len(manager.tools) != 0 {
		t.Error("Manager should start with no tools")
	}

	// Clean up
	manager.Stop()
}

func TestManagerLifecycle(t *testing.T) {
	cfg := &config.Config{
		MCPServers: map[string]config.MCPServerConfig{
			// Use a simple command that will fail but won't hang
			"test": {
				Command: "false", // Command that exits with error
			},
		},
	}

	manager := NewManager(cfg)
	defer manager.Stop()

	// Start should not fail even if connections fail
	err := manager.Start()
	if err != nil {
		t.Errorf("Start() should not fail even with bad configs: %v", err)
	}

	// Should be able to get connected servers (will be empty due to failed connection)
	servers := manager.GetConnectedServers()
	if len(servers) != 0 {
		t.Errorf("Expected no connected servers, got %d", len(servers))
	}

	// Should be able to get tools (will be empty)
	tools := manager.GetAllTools()
	if len(tools) != 0 {
		t.Errorf("Expected no tools, got %d", len(tools))
	}

	// Stop should not panic
	manager.Stop()
}

func TestGetConnectedServers(t *testing.T) {
	cfg := &config.Config{
		MCPServers: map[string]config.MCPServerConfig{},
	}

	manager := NewManager(cfg)
	defer manager.Stop()

	// Should return empty slice when no servers
	servers := manager.GetConnectedServers()
	if len(servers) != 0 {
		t.Errorf("Expected empty slice, got %v", servers)
	}
}

func TestGetAllTools(t *testing.T) {
	cfg := &config.Config{
		MCPServers: map[string]config.MCPServerConfig{},
	}

	manager := NewManager(cfg)
	defer manager.Stop()

	// Should return empty map when no servers
	tools := manager.GetAllTools()
	if len(tools) != 0 {
		t.Errorf("Expected empty map, got %v", tools)
	}
}

func TestCallToolNonexistentServer(t *testing.T) {
	cfg := &config.Config{
		MCPServers: map[string]config.MCPServerConfig{},
	}

	manager := NewManager(cfg)
	defer manager.Stop()

	// Should return error for nonexistent server
	_, err := manager.CallTool("nonexistent", "test_tool", map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for nonexistent server")
	}

	if err.Error() != "server nonexistent not connected" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestWithQuietMode(t *testing.T) {
	cfg := &config.Config{
		MCPServers: map[string]config.MCPServerConfig{
			"test": {
				Command: "echo",
				Args:    []string{"hello"},
			},
		},
	}

	// Test default (verbose mode)
	manager := NewManager(cfg)
	if manager.quiet {
		t.Error("Manager should default to verbose mode (quiet=false)")
	}
	manager.Stop()

	// Test with quiet mode enabled
	managerQuiet := NewManager(cfg, WithQuietMode())
	if !managerQuiet.quiet {
		t.Error("Manager should be in quiet mode when WithQuietMode() is used")
	}
	managerQuiet.Stop()
}

func TestMultipleOptions(t *testing.T) {
	cfg := &config.Config{
		MCPServers: map[string]config.MCPServerConfig{},
	}

	// Test that multiple options can be applied
	// (Currently we only have one option, but this tests the pattern)
	manager := NewManager(cfg, WithQuietMode())
	if !manager.quiet {
		t.Error("Options not applied correctly")
	}
	manager.Stop()
}