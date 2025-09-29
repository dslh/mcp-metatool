package cmd

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"golang.org/x/term"

	"github.com/dslh/mcp-metatool/internal/config"
	"github.com/dslh/mcp-metatool/internal/persistence"
	"github.com/dslh/mcp-metatool/internal/proxy"
)

// ANSI color codes
const (
	colorReset     = "\x1b[0m"
	colorBoldWhite = "\x1b[1;97m"
	colorCyan      = "\x1b[36m"
)

// toolInfo represents a tool with its name and description
type toolInfo struct {
	name        string
	description string
}

// isTerminal checks if stdout is connected to a terminal
func isTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// colorize returns the text with ANSI color codes if terminal supports it
func colorize(text, colorCode string) string {
	if isTerminal() {
		return colorCode + text + colorReset
	}
	return text
}

// truncateDescription removes everything after the first newline
func truncateDescription(desc string) string {
	if idx := strings.Index(desc, "\n"); idx >= 0 {
		return strings.TrimSpace(desc[:idx])
	}
	return desc
}

// printToolGroup prints a group of tools with aligned columns
func printToolGroup(tools []toolInfo) {
	if len(tools) == 0 {
		return
	}

	// Find the longest tool name for alignment
	maxNameLen := 0
	for _, tool := range tools {
		if len(tool.name) > maxNameLen {
			maxNameLen = len(tool.name)
		}
	}

	// Print each tool with aligned descriptions
	for _, tool := range tools {
		toolName := colorize(tool.name, colorBoldWhite)
		desc := truncateDescription(tool.description)
		// Note: We use the uncolored name length for padding calculation
		padding := maxNameLen - len(tool.name)
		fmt.Printf("  • %s%s - %s\n", toolName, strings.Repeat(" ", padding), desc)
	}
}

// ListTools displays all tools exposed by mcp-metatool
func ListTools() error {
	// 1. Load and display saved tools
	fmt.Println(colorize("Saved Tools:", colorCyan))
	savedTools, err := persistence.ListTools()
	if err != nil {
		log.Printf("Warning: failed to load saved tools: %v", err)
	} else if len(savedTools) == 0 {
		fmt.Println("  (none)")
	} else {
		sort.Slice(savedTools, func(i, j int) bool {
			return savedTools[i].Name < savedTools[j].Name
		})
		tools := make([]toolInfo, len(savedTools))
		for i, tool := range savedTools {
			tools[i] = toolInfo{
				name:        tool.Name,
				description: tool.Description,
			}
		}
		printToolGroup(tools)
	}
	fmt.Println()

	// 2. Display built-in tools
	fmt.Println(colorize("Built-in Tools:", colorCyan))
	builtinTools := []toolInfo{
		{"eval_starlark", "Execute Starlark code with access to proxied MCP tools"},
		{"save_tool", "Create or update a composite tool definition"},
		{"list_saved_tools", "List all saved composite tool definitions"},
		{"show_saved_tool", "Show the complete definition of a saved tool"},
		{"delete_saved_tool", "Delete a saved tool definition from storage"},
	}
	printToolGroup(builtinTools)
	fmt.Println()

	// 3. Load MCP server configuration and display proxied tools
	cfg, err := config.LoadDefaultConfig()
	if err != nil {
		// Check if it's just a missing file
		if _, ok := err.(*os.PathError); ok {
			fmt.Println("Proxied Tools:")
			fmt.Println("  (no MCP server configuration found)")
			return nil
		}
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Check if proxied tools are globally hidden
	if config.ShouldHideProxiedTools() {
		fmt.Println("Proxied Tools:")
		fmt.Println("  (hidden via MCP_METATOOL_HIDE_PROXIED_TOOLS environment variable)")
		return nil
	}

	// 4. Connect to MCP servers and fetch tools
	proxyManager := proxy.NewManager(cfg, proxy.WithQuietMode())
	if err := proxyManager.Start(); err != nil {
		return fmt.Errorf("failed to start proxy manager: %w", err)
	}
	defer proxyManager.Stop()

	allTools := proxyManager.GetAllTools()
	if len(allTools) == 0 {
		fmt.Println("Proxied Tools:")
		fmt.Println("  (no tools discovered from MCP servers)")
		return nil
	}

	// Sort server names for consistent output
	serverNames := make([]string, 0, len(allTools))
	for serverName := range allTools {
		serverNames = append(serverNames, serverName)
	}
	sort.Strings(serverNames)

	// Display tools grouped by server
	for _, serverName := range serverNames {
		tools := allTools[serverName]

		// Get server configuration for filtering
		serverConfig, exists := cfg.MCPServers[serverName]
		if !exists {
			log.Printf("Warning: No configuration found for server %s, skipping", serverName)
			continue
		}

		// Skip hidden servers
		if serverConfig.Hidden {
			continue
		}

		// Filter tools based on server configuration and convert to toolInfo
		visibleTools := make([]toolInfo, 0)
		for _, tool := range tools {
			if serverConfig.ShouldIncludeTool(tool.Name) {
				visibleTools = append(visibleTools, toolInfo{
					name:        tool.Name,
					description: tool.Description,
				})
			}
		}

		// Only print section if there are visible tools
		if len(visibleTools) > 0 {
			// Create header with cyan "Proxied Tools from" and bold white server name
			headerPrefix := colorize("Proxied Tools from ", colorCyan)
			serverNameColored := colorize("'"+serverName+"'", colorBoldWhite)
			headerSuffix := colorize(":", colorCyan)
			fmt.Println(headerPrefix + serverNameColored + headerSuffix)
			printToolGroup(visibleTools)
			fmt.Println()
		}
	}

	return nil
}

// Run is the entry point for the list command
func Run(args []string) int {
	if len(args) > 0 && args[0] == "list" {
		if err := ListTools(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	}
	return -1 // Not a list command
}