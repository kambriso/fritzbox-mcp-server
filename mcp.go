package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server wraps the MCP server with TR-064 integration
type Server struct {
	mcpServer *server.MCPServer
	tr064     *Client
	registry  *Registry
	docsIndex *Index
}

// NewServer creates a new MCP server with TR-064 integration
func NewServer(name, version string, tr064Client *Client, registry *Registry, docsIndex *Index) *Server {
	s := &Server{
		tr064:     tr064Client,
		registry:  registry,
		docsIndex: docsIndex,
	}

	// Create MCP server
	s.mcpServer = server.NewMCPServer(name, version)

	// Register introspection tools
	s.registerIntrospectionTools()

	// Register generic execution tool
	s.registerExecutionTools()

	// Auto-register tools for all services
	s.registerAllServiceTools()

	return s
}

// GetMCPServer returns the underlying MCP server instance
func (s *Server) GetMCPServer() *server.MCPServer {
	return s.mcpServer
}

// registerExecutionTools registers the generic action execution tool
func (s *Server) registerExecutionTools() {
	// call_action - generic tool to execute any TR-064 action
	s.mcpServer.AddTool(mcp.Tool{
		Name:        "call_action",
		Description: "Execute any TR-064 action on the FRITZ!Box. Use list_services and describe_action to discover available actions and their parameters.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"service_type": map[string]interface{}{
					"type":        "string",
					"description": "The service type identifier (e.g., urn:dslforum-org:service:DeviceInfo:1)",
				},
				"action": map[string]interface{}{
					"type":        "string",
					"description": "The action name (e.g., GetInfo)",
				},
				"arguments": map[string]interface{}{
					"type":        "object",
					"description": "Input arguments as key-value pairs (optional, use describe_action to see required arguments)",
					"additionalProperties": map[string]interface{}{
						"type": "string",
					},
				},
			},
			Required: []string{"service_type", "action"},
		},
	}, s.handleCallAction)
}

// registerIntrospectionTools registers tools for discovering FRITZ!Box capabilities
func (s *Server) registerIntrospectionTools() {
	// list_services
	s.mcpServer.AddTool(mcp.Tool{
		Name:        "list_services",
		Description: "List all services available on the FRITZ!Box",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}, s.handleListServices)

	// list_actions
	s.mcpServer.AddTool(mcp.Tool{
		Name:        "list_actions",
		Description: "List all actions for a specific service",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"service_type": map[string]interface{}{
					"type":        "string",
					"description": "The service type identifier (e.g., urn:dslforum-org:service:DeviceInfo:1)",
				},
			},
			Required: []string{"service_type"},
		},
	}, s.handleListActions)

	// describe_action
	s.mcpServer.AddTool(mcp.Tool{
		Name:        "describe_action",
		Description: "Get detailed information about a specific action including documentation",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"service_type": map[string]interface{}{
					"type":        "string",
					"description": "The service type identifier",
				},
				"action": map[string]interface{}{
					"type":        "string",
					"description": "The action name",
				},
			},
			Required: []string{"service_type", "action"},
		},
	}, s.handleDescribeAction)
}

// registerAllServiceTools auto-registers tools for all services
// Registers read-only GET actions with no input parameters as convenience tools
func (s *Server) registerAllServiceTools() {
	for serviceType, serviceSpec := range s.registry.Services {
		// Extract service name from URN (e.g., "DeviceInfo" from "urn:dslforum-org:service:DeviceInfo:1")
		serviceName := extractServiceName(serviceType)

		for actionName, actionSpec := range serviceSpec.Actions {
			// Only auto-register GET actions with no input arguments
			if !isReadOnlyAction(actionName) || len(actionSpec.In) > 0 {
				continue
			}

			// Create tool name: servicename_actionname (e.g., deviceinfo_getinfo)
			toolName := strings.ToLower(serviceName + "_" + actionName)

			// Get documentation
			doc := s.docsIndex.Lookup(serviceType, actionName)
			if doc == "" {
				doc = fmt.Sprintf("Execute %s action on %s service", actionName, serviceName)
			}

			// Register the tool
			s.mcpServer.AddTool(mcp.Tool{
				Name:        toolName,
				Description: doc,
				InputSchema: mcp.ToolInputSchema{
					Type:       "object",
					Properties: map[string]interface{}{},
				},
			}, s.createActionHandler(serviceType, actionName))
		}
	}
}

// isReadOnlyAction checks if an action is read-only based on naming convention
func isReadOnlyAction(actionName string) bool {
	actionLower := strings.ToLower(actionName)

	// Allow Get* actions
	if strings.HasPrefix(actionLower, "get") {
		return true
	}

	// Allow X_AVM-DE_Get* actions
	if strings.HasPrefix(actionLower, "x_avm-de_get") {
		return true
	}

	// Disallow Set*, Delete*, Add*, Remove*, etc.
	mutatingPrefixes := []string{"set", "delete", "add", "remove", "update", "create"}
	for _, prefix := range mutatingPrefixes {
		if strings.HasPrefix(actionLower, prefix) {
			return false
		}
	}

	return false
}

// createActionHandler creates a handler for a no-argument action
func (s *Server) createActionHandler(serviceType, actionName string) func(map[string]interface{}) (*mcp.CallToolResult, error) {
	return func(args map[string]interface{}) (*mcp.CallToolResult, error) {
		// Get service and action specs
		actionSpec, err := s.registry.GetAction(serviceType, actionName)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		serviceSpec := s.registry.Services[serviceType]

		// Call FRITZ!Box API
		ctx := context.Background()
		result, err := s.tr064.Call(ctx, serviceSpec, actionSpec, map[string]string{})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("API call failed: %v", err)), nil
		}

		// Format result
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}

// handleListServices implements list_services
func (s *Server) handleListServices(args map[string]interface{}) (*mcp.CallToolResult, error) {
	services := []map[string]string{}

	for serviceType, serviceSpec := range s.registry.Services {
		services = append(services, map[string]string{
			"service_type": serviceType,
			"control_url":  serviceSpec.ControlURL,
		})
	}

	data, err := json.MarshalIndent(services, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal services: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

// handleListActions implements list_actions
func (s *Server) handleListActions(args map[string]interface{}) (*mcp.CallToolResult, error) {
	serviceType, ok := args["service_type"].(string)
	if !ok {
		return mcp.NewToolResultError("service_type is required"), nil
	}

	serviceSpec, ok := s.registry.Services[serviceType]
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf("Service not found: %s", serviceType)), nil
	}

	actions := []map[string]interface{}{}

	for actionName, actionSpec := range serviceSpec.Actions {
		inArgs := []string{}
		for _, arg := range actionSpec.In {
			inArgs = append(inArgs, arg.Name)
		}

		outArgs := []string{}
		for _, arg := range actionSpec.Out {
			outArgs = append(outArgs, arg.Name)
		}

		actions = append(actions, map[string]interface{}{
			"action":   actionName,
			"in_args":  inArgs,
			"out_args": outArgs,
		})
	}

	data, err := json.MarshalIndent(actions, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal actions: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

// handleDescribeAction implements describe_action
func (s *Server) handleDescribeAction(args map[string]interface{}) (*mcp.CallToolResult, error) {
	serviceType, ok := args["service_type"].(string)
	if !ok {
		return mcp.NewToolResultError("service_type is required"), nil
	}

	actionName, ok := args["action"].(string)
	if !ok {
		return mcp.NewToolResultError("action is required"), nil
	}

	actionSpec, err := s.registry.GetAction(serviceType, actionName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Build detailed description with arguments
	inArgNames := []string{}
	inArgs := []map[string]interface{}{}
	for _, arg := range actionSpec.In {
		inArgNames = append(inArgNames, arg.Name)
		argInfo := map[string]interface{}{
			"name":      arg.Name,
			"data_type": arg.DataType,
		}
		if len(arg.AllowedValues) > 0 {
			argInfo["allowed_values"] = arg.AllowedValues
		}
		inArgs = append(inArgs, argInfo)
	}

	outArgNames := []string{}
	outArgs := []map[string]interface{}{}
	for _, arg := range actionSpec.Out {
		outArgNames = append(outArgNames, arg.Name)
		argInfo := map[string]interface{}{
			"name":      arg.Name,
			"data_type": arg.DataType,
		}
		outArgs = append(outArgs, argInfo)
	}

	// Get detailed documentation with argument context
	doc := s.docsIndex.LookupWithArgs(serviceType, actionName, inArgNames, outArgNames)

	result := map[string]interface{}{
		"service_type": serviceType,
		"action":       actionName,
		"doc":          doc,
		"in_args":      inArgs,
		"out_args":     outArgs,
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal description: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

// handleCallAction implements the generic call_action tool
func (s *Server) handleCallAction(args map[string]interface{}) (*mcp.CallToolResult, error) {
	serviceType, ok := args["service_type"].(string)
	if !ok {
		return mcp.NewToolResultError("service_type is required"), nil
	}

	actionName, ok := args["action"].(string)
	if !ok {
		return mcp.NewToolResultError("action is required"), nil
	}

	// Get service and action specs
	serviceSpec, ok := s.registry.Services[serviceType]
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf("Service not found: %s", serviceType)), nil
	}

	actionSpec, err := s.registry.GetAction(serviceType, actionName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Parse arguments
	inputArgs := make(map[string]string)
	if argsMap, ok := args["arguments"].(map[string]interface{}); ok {
		for key, val := range argsMap {
			if strVal, ok := val.(string); ok {
				inputArgs[key] = strVal
			} else {
				inputArgs[key] = fmt.Sprintf("%v", val)
			}
		}
	}

	// Call FRITZ!Box API
	ctx := context.Background()
	result, err := s.tr064.Call(ctx, serviceSpec, actionSpec, inputArgs)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("API call failed: %v", err)), nil
	}

	// Format result
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}
