package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	configContent := `{
  "mcpServers": {
    "github": {
      "command": "mcp-server-github",
      "args": ["--token", "${GITHUB_TOKEN}"],
      "env": {
        "DEBUG": "true"
      }
    },
    "slack": {
      "command": "mcp-server-slack",
      "args": [],
      "env": {
        "SLACK_TOKEN": "${SLACK_TOKEN}"
      }
    }
  }
}`

	// Set up test environment variables
	os.Setenv("GITHUB_TOKEN", "test-github-token")
	os.Setenv("SLACK_TOKEN", "test-slack-token")
	defer os.Unsetenv("GITHUB_TOKEN")
	defer os.Unsetenv("SLACK_TOKEN")

	// Create temporary file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.json")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load config
	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Validate structure
	if len(config.MCPServers) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(config.MCPServers))
	}

	// Check github server
	github, ok := config.MCPServers["github"]
	if !ok {
		t.Fatal("Github server not found")
	}
	if github.Command != "mcp-server-github" {
		t.Errorf("Expected command 'mcp-server-github', got '%s'", github.Command)
	}
	if len(github.Args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(github.Args))
	}
	if github.Args[1] != "test-github-token" {
		t.Errorf("Expected expanded token 'test-github-token', got '%s'", github.Args[1])
	}

	// Check slack server
	slack, ok := config.MCPServers["slack"]
	if !ok {
		t.Fatal("Slack server not found")
	}
	if slack.Env["SLACK_TOKEN"] != "test-slack-token" {
		t.Errorf("Expected expanded token 'test-slack-token', got '%s'", slack.Env["SLACK_TOKEN"])
	}
}

func TestLoadDefaultConfig(t *testing.T) {
	// Create a temporary config with custom MCP_METATOOL_DIR
	configContent := `{
  "mcpServers": {
    "test": {
      "command": "echo",
      "args": ["hello"]
    }
  }
}`

	// Set up test environment
	tmpDir := t.TempDir()
	os.Setenv("MCP_METATOOL_DIR", tmpDir)
	defer os.Unsetenv("MCP_METATOOL_DIR")

	// Create config file
	configPath := filepath.Join(tmpDir, "servers.json")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load default config
	config, err := LoadDefaultConfig()
	if err != nil {
		t.Fatalf("LoadDefaultConfig failed: %v", err)
	}

	// Validate structure
	if len(config.MCPServers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(config.MCPServers))
	}

	// Check test server
	test, ok := config.MCPServers["test"]
	if !ok {
		t.Fatal("Test server not found")
	}
	if test.Command != "echo" {
		t.Errorf("Expected command 'echo', got '%s'", test.Command)
	}
}

func TestGetMetatoolDirectory(t *testing.T) {
	// Test with custom directory
	tmpDir := t.TempDir()
	os.Setenv("MCP_METATOOL_DIR", tmpDir)
	defer os.Unsetenv("MCP_METATOOL_DIR")

	dir, err := GetMetatoolDirectory()
	if err != nil {
		t.Fatalf("GetMetatoolDirectory failed: %v", err)
	}

	if dir != tmpDir {
		t.Errorf("Expected directory %s, got %s", tmpDir, dir)
	}

	// Verify directory was created
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("Directory was not created")
	}
}

func TestExpandString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		envVars  map[string]string
		expected string
	}{
		{
			name:     "simple expansion",
			input:    "${HOME}/config",
			envVars:  map[string]string{"HOME": "/home/user"},
			expected: "/home/user/config",
		},
		{
			name:     "multiple expansions",
			input:    "${USER}@${HOST}",
			envVars:  map[string]string{"USER": "alice", "HOST": "example.com"},
			expected: "alice@example.com",
		},
		{
			name:     "no expansion needed",
			input:    "plain-string",
			envVars:  map[string]string{},
			expected: "plain-string",
		},
		{
			name:     "missing variable",
			input:    "${MISSING_VAR}",
			envVars:  map[string]string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			for key, value := range tt.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			result, err := expandString(tt.input)
			if err != nil {
				t.Errorf("expandString failed: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				MCPServers: map[string]MCPServerConfig{
					"test": {Command: "test-command"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with hidden server",
			config: Config{
				MCPServers: map[string]MCPServerConfig{
					"test": {Command: "test-command", Hidden: true},
				},
			},
			wantErr: false,
		},
		{
			name: "no servers",
			config: Config{
				MCPServers: map[string]MCPServerConfig{},
			},
			wantErr: true,
		},
		{
			name: "empty command",
			config: Config{
				MCPServers: map[string]MCPServerConfig{
					"test": {Command: ""},
				},
			},
			wantErr: true,
		},
		{
			name: "whitespace command",
			config: Config{
				MCPServers: map[string]MCPServerConfig{
					"test": {Command: "   "},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldHideProxiedTools(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected bool
	}{
		{
			name:     "environment variable not set",
			envValue: "",
			expected: false,
		},
		{
			name:     "environment variable set to true",
			envValue: "true",
			expected: true,
		},
		{
			name:     "environment variable set to false",
			envValue: "false",
			expected: true, // Any non-empty value should hide tools
		},
		{
			name:     "environment variable set to 1",
			envValue: "1",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up env var first
			os.Unsetenv("MCP_METATOOL_HIDE_PROXIED_TOOLS")
			
			// Set env var if needed
			if tt.envValue != "" {
				os.Setenv("MCP_METATOOL_HIDE_PROXIED_TOOLS", tt.envValue)
				defer os.Unsetenv("MCP_METATOOL_HIDE_PROXIED_TOOLS")
			}

			result := ShouldHideProxiedTools()
			if result != tt.expected {
				t.Errorf("ShouldHideProxiedTools() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestLoadConfigWithHiddenField(t *testing.T) {
	configContent := `{
  "mcpServers": {
    "visible": {
      "command": "test-visible",
      "args": []
    },
    "hidden": {
      "command": "test-hidden",
      "args": [],
      "hidden": true
    }
  }
}`

	// Create temporary file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.json")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load config
	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Check visible server
	visible, ok := config.MCPServers["visible"]
	if !ok {
		t.Fatal("Visible server not found")
	}
	if visible.Hidden {
		t.Error("Visible server should not be hidden")
	}

	// Check hidden server
	hidden, ok := config.MCPServers["hidden"]
	if !ok {
		t.Fatal("Hidden server not found")
	}
	if !hidden.Hidden {
		t.Error("Hidden server should be hidden")
	}
}