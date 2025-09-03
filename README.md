# MCP Metatool

A Model Context Protocol (MCP) server implementation in Go that will eventually act as a proxy for other MCP servers and provide meta-functionality for tool composition.

## Current Status

This is the initial implementation with a simple "hello world" tool. The server will be expanded to include the full metatool functionality as described in the project specification.

## Installation

```bash
go build -o mcp-metatool .
```

## Usage

The server communicates over stdio using the MCP protocol. To test it manually:

```bash
./mcp-metatool
```

## Available Tools

### hello_world

A simple greeting tool.

**Parameters:**
- `name` (string): The name of the person to greet

**Example:**
```json
{
  "method": "tools/call",
  "params": {
    "name": "hello_world",
    "arguments": {
      "name": "World"
    }
  }
}
```

**Response:**
```json
{
  "content": [
    {
      "type": "text",
      "text": "Hello, World!"
    }
  ]
}
```

## Development

Built using the official MCP Go SDK: https://github.com/modelcontextprotocol/go-sdk

## Next Steps

This server will be expanded to include:
- Proxy functionality for connecting to other MCP servers
- JavaScript runtime for composite tool creation
- Tool management API (`save_tool`, `list_saved_tools`, etc.)
- File-based persistence for saved tools