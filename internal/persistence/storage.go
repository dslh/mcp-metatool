package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dslh/mcp-metatool/internal/paths"
)

// SavedToolDefinition represents a saved tool
type SavedToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	Code        string                 `json:"code"`
}

// GetToolsDirectory returns the directory where tools are stored
// Deprecated: Use paths.GetToolsDir() instead
func GetToolsDirectory() (string, error) {
	return paths.GetToolsDir()
}

// SaveTool saves a tool definition to disk
func SaveTool(tool *SavedToolDefinition) error {
	toolsDir, err := GetToolsDirectory()
	if err != nil {
		return err
	}
	
	// Validate tool name
	if err := validateToolName(tool.Name); err != nil {
		return err
	}
	
	// Write to file
	filename := filepath.Join(toolsDir, tool.Name+".json")
	data, err := json.MarshalIndent(tool, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tool: %w", err)
	}
	
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write tool file: %w", err)
	}
	
	return nil
}

// LoadTool loads a tool definition from disk
func LoadTool(name string) (*SavedToolDefinition, error) {
	toolsDir, err := GetToolsDirectory()
	if err != nil {
		return nil, err
	}
	
	filename := filepath.Join(toolsDir, name+".json")
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read tool file: %w", err)
	}
	
	var tool SavedToolDefinition
	if err := json.Unmarshal(data, &tool); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tool: %w", err)
	}
	
	return &tool, nil
}

// ListTools returns all saved tool definitions
func ListTools() ([]*SavedToolDefinition, error) {
	toolsDir, err := GetToolsDirectory()
	if err != nil {
		return nil, err
	}
	
	entries, err := os.ReadDir(toolsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*SavedToolDefinition{}, nil
		}
		return nil, fmt.Errorf("failed to read tools directory: %w", err)
	}
	
	var tools []*SavedToolDefinition
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		
		toolName := strings.TrimSuffix(entry.Name(), ".json")
		tool, err := LoadTool(toolName)
		if err != nil {
			// Skip malformed tools but continue with others
			continue
		}
		tools = append(tools, tool)
	}
	
	return tools, nil
}

// DeleteTool removes a tool definition from disk
func DeleteTool(name string) error {
	if err := validateToolName(name); err != nil {
		return err
	}
	
	toolsDir, err := GetToolsDirectory()
	if err != nil {
		return err
	}
	
	filename := filepath.Join(toolsDir, name+".json")
	if err := os.Remove(filename); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("tool '%s' does not exist", name)
		}
		return fmt.Errorf("failed to delete tool: %w", err)
	}
	
	return nil
}

// validateToolName ensures the tool name is safe for filesystem use
func validateToolName(name string) error {
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}
	
	if len(name) > 100 {
		return fmt.Errorf("tool name too long (max 100 characters)")
	}
	
	// Check for filesystem-unsafe characters
	unsafe := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", "..", " "}
	for _, char := range unsafe {
		if strings.Contains(name, char) {
			return fmt.Errorf("tool name contains invalid character: %s", char)
		}
	}
	
	return nil
}