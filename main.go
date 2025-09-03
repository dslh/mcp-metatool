package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-metatool",
		Version: "0.1.0",
	}, nil)

	type HelloWorldArgs struct {
		Name string `json:"name" jsonschema:"the name of the person to greet"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "hello_world",
		Description: "A simple hello world tool that greets a person by name",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args HelloWorldArgs) (*mcp.CallToolResult, any, error) {
		greeting := "Hello, " + args.Name + "!"
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: greeting},
			},
		}, nil, nil
	})

	log.Printf("Starting MCP metatool server...")
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}