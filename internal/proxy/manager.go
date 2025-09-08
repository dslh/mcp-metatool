package proxy

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dslh/mcp-metatool/internal/config"
)

// Manager manages connections to upstream MCP servers
type Manager struct {
	config    *config.Config
	clients   map[string]*mcp.Client
	sessions  map[string]*mcp.ClientSession
	tools     map[string][]*mcp.Tool // server name -> tools
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewManager creates a new proxy manager
func NewManager(cfg *config.Config) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		config:   cfg,
		clients:  make(map[string]*mcp.Client),
		sessions: make(map[string]*mcp.ClientSession),
		tools:    make(map[string][]*mcp.Tool),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start initializes connections to all configured upstream servers
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for serverName, serverConfig := range m.config.MCPServers {
		if err := m.connectServer(serverName, serverConfig); err != nil {
			log.Printf("Warning: Failed to connect to server %s: %v", serverName, err)
			// Continue with other servers instead of failing completely
			continue
		}
	}

	return nil
}

// Stop closes all connections and cleans up resources
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Cancel the context to signal shutdown
	m.cancel()

	// Close all sessions
	for serverName, session := range m.sessions {
		if err := session.Close(); err != nil {
			log.Printf("Error closing session for server %s: %v", serverName, err)
		}
	}

	// Clear all state
	m.clients = make(map[string]*mcp.Client)
	m.sessions = make(map[string]*mcp.ClientSession)
	m.tools = make(map[string][]*mcp.Tool)
}

// connectServer establishes a connection to a single upstream server
func (m *Manager) connectServer(serverName string, serverConfig config.MCPServerConfig) error {
	// Create the command
	cmd := exec.CommandContext(m.ctx, serverConfig.Command, serverConfig.Args...)
	
	// Set environment variables
	if len(serverConfig.Env) > 0 {
		env := cmd.Environ()
		for key, value := range serverConfig.Env {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
		cmd.Env = env
	}

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "mcp-metatool",
		Version: "0.1.0",
	}, nil)

	// Create transport and connect
	transport := mcp.NewCommandTransport(cmd)
	session, err := client.Connect(m.ctx, transport, &mcp.ClientSessionOptions{})
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	// Store client and session
	m.clients[serverName] = client
	m.sessions[serverName] = session

	// Discover tools
	if err := m.discoverTools(serverName, session); err != nil {
		log.Printf("Warning: Failed to discover tools for server %s: %v", serverName, err)
		// Don't fail the connection for tool discovery issues
	}

	log.Printf("Successfully connected to MCP server: %s", serverName)
	return nil
}

// discoverTools queries a server for its available tools
func (m *Manager) discoverTools(serverName string, session *mcp.ClientSession) error {
	// List tools from the upstream server
	result, err := session.ListTools(m.ctx, &mcp.ListToolsParams{})
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	// Store the tools
	m.tools[serverName] = result.Tools
	
	log.Printf("Discovered %d tools from server %s", len(result.Tools), serverName)
	for _, tool := range result.Tools {
		log.Printf("  - %s: %s", tool.Name, tool.Description)
	}

	return nil
}

// GetAllTools returns all discovered tools from all servers
func (m *Manager) GetAllTools() map[string][]*mcp.Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create a copy to avoid race conditions
	result := make(map[string][]*mcp.Tool)
	for serverName, tools := range m.tools {
		result[serverName] = make([]*mcp.Tool, len(tools))
		copy(result[serverName], tools)
	}

	return result
}

// CallTool calls a tool on the specified upstream server
func (m *Manager) CallTool(serverName, toolName string, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	m.mu.RLock()
	session, exists := m.sessions[serverName]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("server %s not connected", serverName)
	}

	// Call the tool
	result, err := session.CallTool(m.ctx, &mcp.CallToolParams{
		Name:      toolName,
		Arguments: arguments,
	})
	if err != nil {
		return nil, fmt.Errorf("tool call failed: %w", err)
	}

	return result, nil
}

// GetConnectedServers returns the names of all connected servers
func (m *Manager) GetConnectedServers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	servers := make([]string, 0, len(m.sessions))
	for serverName := range m.sessions {
		servers = append(servers, serverName)
	}

	return servers
}