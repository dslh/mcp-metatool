MCP Metatool Design Specification
=================================

Executive Summary
-----------------

The MCP Metatool is a proxy server that enables agents to create, manage, and execute composite tools from existing MCP server capabilities. It exposes proxied tools from connected MCP servers while adding meta-functionality for tool composition through a Starlark runtime.

Core Architecture
-----------------

### System Components

```
Agent <--> MCP Metatool Server <--> Connected MCP Servers
                |                         |
                |                    (GitHub, ZenHub,
            Starlark Runtime         Slack, etc.)
                |
           Saved Tools (disk)
```

### MCP Server Configuration

-   **Transport**: stdio (standard input/output)
-   **Persistence**: Local filesystem for saved tools
-   **Runtime**: Starlark execution via go.starlark.net for sandboxing
-   **Connection Mode**: Acts as MCP client to upstream servers, MCP server to agents

Tool Management API
-------------------

### Core Tools

#### 1\. `save_tool`

Creates or updates a composite tool definition.

**Parameters:**

starlark

```
{
  "name": "string",           // Tool identifier
  "description": "string",    // Human-readable description
  "inputSchema": {            // JSON Schema for parameters
    "type": "object",
    "properties": {...},
    "required": [...]
  },
  "code": "string"           // Starlark implementation
}
```

**Starlark Environment:**

-   All proxied tools available as synchronous functions
-   Standard Starlark features (try/except, loops, conditionals)
-   Deterministic execution model
-   Minimal utility functions provided

**Example Tool Definition:**

starlark

```
# Access proxied tools as functions
issue = github.searchIssues({
  "query": params.issueTitle
})[0]

zenhubData = zenhub.getIssueDetails({
  "issueId": issue.id,
  "repoId": issue.repository.id
})

return {
  "githubUrl": issue.html_url,
  "zenhubEstimate": zenhubData.estimate,
  "pipeline": zenhubData.pipeline.name,
  "blockedBy": zenhubData.dependencies
}
```

#### 2\. `list_saved_tools`

Returns all saved tool definitions with metadata.

**Response:**

json

```
[
  {
    "name": "string",
    "description": "string",
    "created": "ISO8601",
    "modified": "ISO8601",
    "inputSchema": {...}
  }
]
```

#### 3\. `show_saved_tool`

Retrieves complete definition of a specific tool.

**Parameters:**

json

```
{
  "name": "string"  // Tool name
}
```

#### 4\. `delete_saved_tool`

Removes a saved tool definition.

**Parameters:**

json

```
{
  "name": "string"  // Tool name
}
```

Starlark Execution Environment
------------------------------

### Sandboxing

-   Use `go.starlark.net/starlark` for secure execution
-   No filesystem access
-   No network access (except via proxied tools)
-   Deterministic execution
-   Memory limits enforced
-   Execution timeout (configurable, default 30s)

### Available APIs

**Proxied Tools:** All tools from connected MCP servers exposed as functions:

starlark

```
# Pattern: serverName.toolName(params)
github.createIssue({...})
slack.postMessage({...})
database.query({...})
```

**Built-in Objects:**

-   Standard Starlark primitives (int, float, string, bool, list, dict, set)
-   JSON parsing/stringification
-   Math, string, list, dict methods
-   Print function (captured in execution context)

**Explicitly Excluded:**

-   File system operations
-   Network operations (except via proxied tools)
-   System/process access
-   Non-deterministic operations
-   Import statements (except built-ins)

Input/Output Handling
---------------------

### Parameter Validation

Before executing saved tools:

1.  Validate against defined `inputSchema`
2.  Return structured error if validation fails
3.  Type coercion where appropriate

### Response Format

json

```
{
  "result": any,           // Tool execution result
  "logs": string[],        // Captured print output
  "executionTime": number, // Milliseconds
  "toolCalls": [{          // Audit trail
    "tool": "string",
    "params": {...},
    "result": any
  }]
}
```

Error Handling
--------------

### Error Types

1.  **Validation Errors**: Input schema mismatches
2.  **Runtime Errors**: Starlark execution failures
3.  **Tool Errors**: Proxied tool failures
4.  **Timeout Errors**: Execution time exceeded
5.  **Resource Errors**: Memory/recursion limits

### Error Response Format

json

```
{
  "error": {
    "type": "string",
    "message": "string",
    "details": {...},      // Optional context
    "stack": "string"      // If debug mode enabled
  }
}
```

Security Considerations
-----------------------

### Tool Execution

-   Each tool runs in isolated context
-   No state persistence between executions
-   Resource limits strictly enforced
-   No ability to modify metatool server state

### Authentication

-   Inherit authentication from connected MCP servers
-   No credential storage in saved tools
-   Authentication flows handled by metatool proxy

### Audit & Logging

-   Log all tool executions with timestamps
-   Track which proxied tools were called
-   Optional detailed execution traces

Persistence
-----------

### Storage Format

json

```
// ~/.mcp-metatool/tools/{toolName}.json
{
  "version": "1.0",
  "name": "string",
  "description": "string",
  "inputSchema": {...},
  "code": "string",
  "metadata": {
    "created": "ISO8601",
    "modified": "ISO8601",
    "executionCount": number,
    "lastExecuted": "ISO8601"
  }
}
```

### File Organization

```
~/.mcp-metatool/
├── config.json          # Server configuration
├── tools/               # Saved tool definitions
│   ├── tool1.json
│   └── tool2.json
└── logs/                # Execution logs (optional)
```

Configuration
-------------

### Server Configuration (stdio)

json

```
// .mcp.json
{
  "mcpServers": {
    "metatool": {
      "command": "mcp-metatool",
      "args": ["--config", "path/to/config.json"],
      "env": {
        "MCP_METATOOL_TIMEOUT": "30000",
        "MCP_METATOOL_DEBUG": "false"
      }
    }
  }
}
```

### Metatool Configuration

json

```
// config.json
{
  "upstreamServers": [
    {
      "name": "github",
      "command": "mcp-server-github",
      "args": ["--token", "${GITHUB_TOKEN}"]
    },
    {
      "name": "zenhub",
      "command": "mcp-server-zenhub",
      "args": []
    }
  ],
  "execution": {
    "timeout": 30000,
    "maxMemory": "128MB",
    "enableDebug": false
  }
}
```

Implementation Phases
---------------------

### Phase 1: Core Functionality

-   [ ]  Basic proxy functionality
-   [ ]  Starlark runtime with go.starlark.net
-   [ ]  save_tool, list_saved_tools implementations
-   [ ]  Simple file-based persistence

### Phase 2: Enhanced Features

-   [ ]  Input schema validation
-   [ ]  Execution timeouts and resource limits
-   [ ]  show_saved_tool, delete_saved_tool
-   [ ]  Error handling improvements

### Phase 3: Production Readiness

-   [ ]  Comprehensive logging
-   [ ]  Performance optimizations
-   [ ]  Security hardening
-   [ ]  Documentation and examples

Future Considerations
---------------------

### Potential Enhancements

-   **Testing Framework**: Mock tool responses for dry-run testing
-   **Composition Patterns**: Library of common patterns
-   **Tool Discovery**: Better introspection of available proxied tools
-   **Metrics**: Execution statistics and performance monitoring
-   **Sharing**: Export/import tool definitions

### Deferred Features

-   Version control integration (git)
-   Remote tool storage
-   Async execution model
-   Visual tool builder

Success Criteria
----------------

1.  Agents can easily create composite tools without understanding implementation details
2.  Saved tools appear seamlessly alongside native tools in tool listings
3.  Execution is secure and resource-bounded
4.  Error messages are clear and actionable
5.  Performance overhead is minimal (<100ms per composite tool call)