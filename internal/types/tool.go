package types

import "encoding/json"

// SaveToolArgs defines the arguments for the save_tool MCP tool
type SaveToolArgs struct {
	Name        string                 `json:"name" jsonschema:"Tool identifier"`
	Description string                 `json:"description" jsonschema:"Human-readable description of what the tool does"`
	InputSchema map[string]interface{} `json:"inputSchema" jsonschema:"JSON Schema for tool parameters"`
	Code        string                 `json:"code" jsonschema:"Starlark implementation of the tool"`
}

// SavedToolParams provides a flexible parameter structure for saved tools
// This allows the MCP framework to properly validate parameters while still
// allowing dynamic parameter schemas from saved tool definitions
type SavedToolParams struct {
	// Use json.RawMessage to preserve the original JSON structure
	// This allows us to handle any parameter schema dynamically
	Parameters json.RawMessage `json:"parameters,omitempty"`
}