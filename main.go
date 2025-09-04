package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dslh/mcp-metatool/internal/tools"
)

func main() {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-metatool",
		Version: "0.1.0",
	}, nil)

	// Register built-in tools
	tools.RegisterEvalStarlark(server)
	tools.RegisterSaveTool(server)

	// Load and register saved tools
	if err := tools.RegisterSavedTools(server); err != nil {
		log.Printf("Warning: failed to load saved tools: %v", err)
	}

	log.Printf("Starting MCP metatool server...")
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}