# MCP Metatool

A Model Context Protocol (MCP) server implementation in Go that acts as a proxy for other MCP servers and provides meta-functionality for tool composition using Starlark scripts.

## Current Status

The server now includes:
- ✅ **Starlark Runtime**: Execute arbitrary Starlark code with parameter passing
- ✅ **Flexible Result Handling**: Support both explicit `result` variables and automatic globals capture
- ✅ **Clean Architecture**: Modular codebase ready for extension
- 🚧 **Tool Composition**: Coming soon - save and execute composite tools

## Installation

```bash
go build -o mcp-metatool .
```

## Usage

The server communicates over stdio using the MCP protocol. Add it to your Claude Code configuration:

```json
{
  "mcpServers": {
    "mcp-metatool": {
      "type": "stdio",
      "command": "/path/to/mcp-metatool"
    }
  }
}
```

## Available Tools

### eval_starlark

Execute Starlark code and return structured results.

**Parameters:**
- `code` (string): The Starlark code to execute
- `params` (object, optional): Parameters available as `params` dict in the code

**Examples:**

Simple expression:
```python
2 + 3  # Returns: 5
```

With parameters:
```python
"Hello, " + params["name"]  # With params: {"name": "World"} → "Hello, World"
```

Complex data processing:
```python
data = [1, 2, 3, 4, 5]
processed = [x * 2 for x in data]
result = {
    "original": data,
    "processed": processed,
    "count": len(processed)
}
# Returns: {"original": [1,2,3,4,5], "processed": [2,4,6,8,10], "count": 5}
```

Multiple variables (no explicit result):
```python
name = "Alice"
age = 30
scores = [95, 87, 92]
# Returns: {"name": "Alice", "age": 30, "scores": [95, 87, 92]}
```

## Project Structure

```
├── main.go                 # Server setup and tool registration
├── internal/
│   ├── starlark/
│   │   ├── executor.go     # Starlark execution engine
│   │   └── convert.go      # Go<->Starlark value conversion
│   └── tools/
│       └── eval.go         # eval_starlark tool definition
└── spec.md                 # Full project specification
```

## Development

Built using:
- [Official MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk)
- [Starlark in Go](https://pkg.go.dev/go.starlark.net/starlark)

## Roadmap

**Next Phase**:
- `save_tool`: Create and persist composite tools written in Starlark
- `list_saved_tools`, `show_saved_tool`, `delete_saved_tool`: Tool management API
- File-based persistence for saved tools

**Future**:
- MCP server proxying: Connect to upstream MCP servers and expose their tools in Starlark
- Advanced tool composition patterns and examples