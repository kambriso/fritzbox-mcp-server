// SPDX-License-Identifier: LicenseRef-KSAL-1.0

package main

import (
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// TestMCPUnconfiguredState verifies that tools return an instructional error
// when the server is started without a valid configuration.
func TestMCPUnconfiguredState(t *testing.T) {
	configErr := fmt.Errorf("No configuration found. Please run setup")

	// Create a server in unconfigured state
	// Parameters: name, version, tr064Client, registry, docsIndex, configErr
	srv := newServer("test-server", "1.0.0", nil, nil, nil, configErr)

	// Test cases for different tool handlers
	tests := []struct {
		name        string
		toolName    string
		expectError bool
	}{
		{"ListServices", "list_services", true},
		{"ListActions", "list_actions", true},
		{"CallAction", "call_action", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result *mcp.CallToolResult
			var err error

			// We need to call the handlers directly
			// The handlers take map[string]interface{} as arguments
			args := map[string]interface{}{}

			switch tt.toolName {
			case "list_services":
				result, err = srv.handleListServices(args)
			case "list_actions":
				args["service_type"] = "any"
				result, err = srv.handleListActions(args)
			case "call_action":
				args["service_type"] = "any"
				args["action"] = "any"
				result, err = srv.handleCallAction(args)
			}

			if err != nil {
				t.Fatalf("Handler returned unexpected error: %v", err)
			}

			if !result.IsError {
				t.Errorf("Expected IsError=true for unconfigured server, got false")
			}

			// Verify error message content
			found := false
			for _, msg := range result.Content {
				if text, ok := msg.(mcp.TextContent); ok {
					if text.Text != "" && (fmt.Sprintf("Action failed: %v", configErr) == text.Text) {
						found = true
						break
					}
				}
			}

			if !found {
				t.Errorf("Instructional error message not found in result content")
			}
		})
	}
}

// TestUnconfiguredNoArgumentTools verifies auto-registered "no-argument" tools
func TestUnconfiguredNoArgumentTools(t *testing.T) {
	configErr := fmt.Errorf("No configuration found")

	// Mock a registry with one service and one read-only action
	reg := &registry{
		Services: map[string]*serviceSpec{
			"urn:test:service:1": {
				ServiceType: "urn:test:service:1",
				Actions: map[string]*actionSpec{
					"GetStatus": {
						Name: "GetStatus",
						In:   []argSpec{},
					},
				},
			},
		},
	}

	srv := newServer("test-server", "1.0.0", nil, reg, newIndex(), configErr)

	// The tool handler should be created even if config is bad
	handler := srv.createActionHandler("urn:test:service:1", "GetStatus")

	result, err := handler(nil)
	if err != nil {
		t.Fatalf("Handler failed: %v", err)
	}

	if !result.IsError {
		t.Error("Expected error result for no-argument tool")
	}
}
