package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestRun_ListCommand(t *testing.T) {
	// Test that Run recognizes "list" command
	exitCode := Run([]string{"list"})
	if exitCode < 0 {
		t.Error("Run should recognize 'list' command")
	}
}

func TestRun_NonListCommand(t *testing.T) {
	// Test that Run returns -1 for non-list commands
	exitCode := Run([]string{"other"})
	if exitCode != -1 {
		t.Errorf("Run should return -1 for non-list commands, got %d", exitCode)
	}
}

func TestRun_NoArgs(t *testing.T) {
	// Test that Run returns -1 when no args
	exitCode := Run([]string{})
	if exitCode != -1 {
		t.Errorf("Run should return -1 for no args, got %d", exitCode)
	}
}

func TestListTools_SavedToolsSection(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run list command
	err := ListTools()

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should not fail
	if err != nil {
		t.Errorf("ListTools should not fail: %v", err)
	}

	// Should contain expected sections
	if !strings.Contains(output, "Saved Tools:") {
		t.Error("Output should contain 'Saved Tools:' section")
	}

	if !strings.Contains(output, "Built-in Tools:") {
		t.Error("Output should contain 'Built-in Tools:' section")
	}

	// Should list built-in tools
	expectedTools := []string{
		"eval_starlark",
		"save_tool",
		"list_saved_tools",
		"show_saved_tool",
		"delete_saved_tool",
	}

	for _, tool := range expectedTools {
		if !strings.Contains(output, tool) {
			t.Errorf("Output should contain built-in tool: %s", tool)
		}
	}
}

func TestListTools_OutputFormat(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run list command
	ListTools()

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Check that output uses bullet points for tools
	if !strings.Contains(output, "• eval_starlark") {
		t.Error("Tools should be formatted with bullet points")
	}

	// Check that descriptions are included
	if !strings.Contains(output, "Execute Starlark code") {
		t.Error("Tool descriptions should be included")
	}
}