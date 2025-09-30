package main

import (
	"context"
	"log"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dslh/mcp-metatool/internal/cmd"
	"github.com/dslh/mcp-metatool/internal/config"
	"github.com/dslh/mcp-metatool/internal/proxy"
	"github.com/dslh/mcp-metatool/internal/tools"
)

func main() {
	// Check for subcommands
	if exitCode := cmd.Run(os.Args[1:]); exitCode >= 0 {
		os.Exit(exitCode)
	}

	// No subcommand matched, proceed with normal MCP server startup
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-metatool",
		Version: "0.1.0",
	}, nil)

	// Initialize proxy manager if config exists
	var proxyManager *proxy.Manager
	cfg, err := config.LoadDefaultConfig()
	if err != nil {
		// Check if it's just a missing file
		if _, ok := err.(*os.PathError); ok {
			log.Printf("No MCP server configuration found - running without proxied servers")
		} else {
			log.Printf("Warning: failed to load config: %v", err)
		}
	} else if err := cfg.Validate(); err != nil {
		log.Printf("Warning: invalid config: %v", err)
	} else {
		proxyManager = proxy.NewManager(cfg)
		if err := proxyManager.Start(); err != nil {
			log.Printf("Warning: failed to start proxy manager: %v", err)
			proxyManager = nil
		} else {
			log.Printf("Proxy manager started with %d servers", len(proxyManager.GetConnectedServers()))
			
			// Register proxied tools with the MCP server
			if err := tools.RegisterProxiedTools(server, proxyManager, cfg); err != nil {
				log.Printf("Warning: failed to register proxied tools: %v", err)
			}
		}
	}

	// Ensure proxy manager is cleaned up on exit
	if proxyManager != nil {
		defer proxyManager.Stop()
	}

	// Register built-in tools
	tools.RegisterEvalStarlark(server, proxyManager)
	tools.RegisterSaveTool(server)
	tools.RegisterListSavedTools(server)
	tools.RegisterShowSavedTool(server)
	tools.RegisterDeleteSavedTool(server)

	// Load and register saved tools
	if err := tools.RegisterSavedTools(server, proxyManager); err != nil {
		log.Printf("Warning: failed to load saved tools: %v", err)
	}

	log.Printf("Starting MCP metatool server...")
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

