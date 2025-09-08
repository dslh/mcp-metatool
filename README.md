# MCP Metatool

A Model Context Protocol (MCP) server implementation in Go that enables tool composition using Starlark scripts. Create, save, and execute custom composite tools that combine logic and data processing capabilities.

## Current Status

The server now includes:
- ✅ **Starlark Runtime**: Execute arbitrary Starlark code with parameter passing and flexible result handling
- ✅ **Tool Composition**: Save and execute custom composite tools written in Starlark
- ✅ **Dynamic Tool Loading**: Saved tools are automatically loaded and registered at startup
- ✅ **Tool Management API**: List, view, and delete saved tools with dedicated management commands
- ✅ **File-based Persistence**: Tools stored as JSON files with configurable directory
- ✅ **Enhanced Type Support**: Full support for Starlark tuples and complex data structures
- ✅ **Clean Architecture**: Modular, well-tested codebase ready for extension

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

### Environment Variables

- `MCP_METATOOL_DIR`: Override the default storage directory (`~/.mcp-metatool`)

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

### save_tool

Create or update a composite tool definition that can be executed later.

**Parameters:**
- `name` (string): Tool identifier
- `description` (string): Human-readable description of what the tool does
- `inputSchema` (object): JSON Schema for tool parameters
- `code` (string): Starlark implementation of the tool

**Example:**
```javascript
// Create a greeting tool
{
  "name": "greet_user",
  "description": "A simple greeting tool that takes a name parameter",
  "inputSchema": {
    "type": "object",
    "properties": {
      "name": {
        "type": "string",
        "description": "The name to greet"
      }
    },
    "required": ["name"]
  },
  "code": "name = params.get('name', 'World')\nresult = 'Hello, ' + name + '!'"
}
```

### list_saved_tools

List all saved composite tool definitions.

**Parameters:** None

**Returns:** A list of saved tools with their names and descriptions.

**Example:**
```javascript
list_saved_tools()  // Returns: {"tools": [{"name": "greet_user", "description": "A simple greeting tool"}]}
```

### show_saved_tool

Show the complete definition of a saved tool including its code, schema, and metadata.

**Parameters:**
- `name` (string): The name of the tool to display

**Example:**
```javascript
show_saved_tool({"name": "greet_user"})  // Returns complete tool definition
```

### delete_saved_tool

Delete a saved tool definition from storage.

**Parameters:**
- `name` (string): The name of the tool to delete

**Example:**
```javascript
delete_saved_tool({"name": "greet_user"})  // Removes the tool (restart server to unregister)
```

### Dynamic Saved Tools

Once saved with `save_tool`, custom tools become available as regular MCP tools. For example, the `greet_user` tool above becomes callable with:

```javascript
// Call the saved tool
greet_user({"name": "Alice"})  // Returns: "Hello, Alice!"
```

## Project Structure

```
├── main.go                 # Server setup and initialization
├── internal/
│   ├── persistence/
│   │   └── storage.go      # File-based tool persistence
│   ├── starlark/
│   │   ├── executor.go     # Starlark execution engine
│   │   └── convert.go      # Go<->Starlark value conversion
│   ├── tools/
│   │   ├── eval.go         # eval_starlark tool
│   │   ├── save.go         # save_tool tool
│   │   ├── manage.go       # Tool management API (list/show/delete)
│   │   └── saved.go        # Dynamic saved tool registration
│   └── types/
│       └── tool.go         # Type definitions
└── spec.md                 # Full project specification
```

## Development

Built using:
- [Official MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk)
- [Starlark in Go](https://pkg.go.dev/go.starlark.net/starlark)

Run tests:
```bash
go test ./...
```

## Storage

Saved tools are stored as JSON files in `~/.mcp-metatool/tools/` (or `$MCP_METATOOL_DIR/tools/`). Each tool is saved as `{toolname}.json` with the complete tool definition including metadata.

## Roadmap

**Next Phase**:
- Input schema validation for saved tools
- Enhanced error handling and validation messages

**Future**:
- MCP server proxying: Connect to upstream MCP servers and expose their tools in Starlark
- Advanced tool composition patterns and examples
- Tool versioning and migration support