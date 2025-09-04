package types

// SaveToolArgs defines the arguments for the save_tool MCP tool
type SaveToolArgs struct {
	Name        string                 `json:"name" jsonschema:"Tool identifier"`
	Description string                 `json:"description" jsonschema:"Human-readable description of what the tool does"`
	InputSchema map[string]interface{} `json:"inputSchema" jsonschema:"JSON Schema for tool parameters"`
	Code        string                 `json:"code" jsonschema:"Starlark implementation of the tool"`
}

// SavedToolParams provides a flexible parameter structure for saved tools
// This struct accepts any JSON object and allows the MCP framework to validate
// against the dynamic schemas from saved tool definitions
type SavedToolParams map[string]interface{}