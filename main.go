// SPDX-License-Identifier: LicenseRef-KSAL-1.0

package main

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

//go:embed USAGE.txt
var usageText string

const (
	serverName = "fritzbox-mcp-server"
)

// Build metadata (set via ldflags)
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

var (
	debug       = flag.Bool("debug", false, "Enable verbose debug logging")
	showVersion = flag.Bool("version", false, "Show version information and exit")
	// CLI execution mode
	executeMode  = flag.Bool("execute", false, "Execute a single action and exit (CLI mode)")
	serviceType  = flag.String("service", "", "Service type for CLI execution (e.g., urn:dslforum-org:service:DeviceInfo:1)")
	actionName   = flag.String("action", "", "Action name for CLI execution (e.g., GetInfo)")
	actionArgs   = flag.String("args", "{}", "Action arguments as JSON object for CLI execution")
	executeQuiet = flag.Bool("quiet", false, "Suppress log output in execute mode (only print result)")
	setupMode    = flag.Bool("setup", false, "Interactive setup to discover FRITZ!Box and create .env file")
)

func main() {
	// Set custom usage function
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s\n", usageText)
	}

	flag.Parse()

	if *showVersion {
		fmt.Printf("%s version %s (commit %s, built %s)\n", serverName, version, commit, date)
		os.Exit(0)
	}

	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run() error {
	// Set up logging
	log.SetOutput(os.Stderr) // MCP uses stdout for protocol, stderr for logs

	// If --setup flag is set, run interactive setup
	if *setupMode {
		return runSetupMode()
	}

	ctx := context.Background()

	// If --execute flag is set, execute action and exit (CLI mode)
	if *executeMode {
		return runExecuteMode(ctx)
	}

	// Step 1: load full configuration (requires auth credentials)
	var configErr error
	cfg, err := load()
	if err != nil {
		configErr = fmt.Errorf("No configuration found. Please run `%s --setup` in your terminal to discover and configure your device.", os.Args[0])
		log.Printf("Warning: %v", configErr)
	}

	// Step 2-4: Disover services (only if config is valid)
	var tr064Client *client
	var registry *registry

	if configErr == nil {
		log.Printf("Configured for FRITZ!Box at %s", cfg.baseURL())

		// Create TR-064 client
		tr064Client = newClient(cfg.baseURL(), cfg.Username, cfg.Password, *debug)

		// Discover services from FRITZ!Box
		log.Println("Discovering TR-064 services from FRITZ!Box...")
		registry, err = loadFromFritz(ctx, cfg.baseURL())
		if err != nil {
			configErr = fmt.Errorf("failed to load TR-064 services: %w", err)
			log.Printf("Warning: %v", configErr)
		} else {
			log.Printf("Discovered %d services", len(registry.Services))

			// Optional - call DeviceInfo:GetInfo to log device details
			log.Println("Fetching device information...")
			if err := logDeviceInfo(ctx, tr064Client, registry); err != nil {
				log.Printf("Warning: could not get device info: %v", err)
			}
		}
	}

	// Step 5: Create documentation index
	log.Println("Loading documentation index...")
	docsIndex := newIndex()

	// Step 6: Create and start MCP server
	log.Printf("Starting MCP server v%s...\n", version)
	mcpSrv := newServer(serverName, version, tr064Client, registry, docsIndex, configErr)

	log.Println("MCP server ready")
	return server.ServeStdio(mcpSrv.getMCPServer())
}

// runSetupMode runs the interactive setup flow
func runSetupMode() error {
	fmt.Println("=== FRITZ!Box MCP Server Setup ===")

	// Check if already configured
	if path := configFile(); path != "" {
		fmt.Printf("Existing configuration found at %s.\n", path)
		fmt.Print("Do you want to overwrite it? [y/N]: ")
		var resp string
		fmt.Scanln(&resp)
		if strings.ToLower(resp) != "y" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	ctx := context.Background()

	// 1. Discovery
	fmt.Print("Searching for FRITZ!Box devices (2s)... ")
	results, err := discoverFritzBox(ctx, 2*time.Second)
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}

	if len(results) == 0 {
		fmt.Print("Not found. Retrying with longer timeout (10s)... ")
		results, err = discoverFritzBox(ctx, 10*time.Second)
		if err != nil {
			return fmt.Errorf("discovery retry failed: %w", err)
		}
	}

	if len(results) == 0 {
		fmt.Println("\nNo FRITZ!Box devices found. Please enter details manually.")
	}

	var selected *discoveryResult
	if len(results) == 1 {
		selected = &results[0]
		fmt.Printf("\nFound %s at %s", selected.Model, selected.IP)
		if selected.IsMaster {
			fmt.Print(" [Mesh Master]")
		}
		fmt.Println()
	} else if len(results) > 1 {
		fmt.Println("\nMultiple FRITZ!Box devices found:")
		for i, res := range results {
			masterHint := ""
			if res.IsMaster {
				masterHint = " [Detected Mesh Master]"
			}
			fmt.Printf("%d. %s at %s%s\n", i+1, res.Model, res.IP, masterHint)
		}
		fmt.Print("Select device [1]: ")
		var idx int
		fmt.Scanln(&idx)
		if idx < 1 || idx > len(results) {
			idx = 1
		}
		selected = &results[idx-1]
	}

	// 2. Credentials
	reader := bufio.NewReader(os.Stdin)
	cfg := &config{
		Host: "192.168.178.1",
		Port: 49000,
	}

	if selected != nil {
		cfg.Host = selected.IP
	} else {
		fmt.Printf("FRITZ!Box IP [%s]: ", cfg.Host)
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line != "" {
			cfg.Host = line
		}
	}

	fmt.Printf("Username [mcp]: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)
	if username == "" {
		username = "mcp"
	}
	cfg.Username = username

	fmt.Printf("Password: ")
	password, _ := reader.ReadString('\n')
	cfg.Password = strings.TrimSpace(password)

	fmt.Printf("Use TLS (HTTPS) [n/Y]: ")
	tlsResp, _ := reader.ReadString('\n')
	tlsResp = strings.TrimSpace(strings.ToLower(tlsResp))
	if tlsResp == "" || tlsResp == "y" {
		cfg.TLS = true
		cfg.Port = 4433
	}

	// 3. Save
	targetPath := ".env"
	if configDir := getConfigDir(); configDir != "" {
		targetPath = filepath.Join(configDir, ".env")
		// Ask if user wants it in global or local dir
		fmt.Printf("Save configuration to %s? [Y/n]: ", targetPath)
		saveResp, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(saveResp)) == "n" {
			targetPath = ".env"
		}
	}

	if err := cfg.save(targetPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("\n✓ Configuration saved to %s (permissions 0600)\n", targetPath)
	fmt.Println("Setup complete. You can now start the MCP server.")
	return nil
}

// runExecuteMode executes a single action in CLI mode and exits
func runExecuteMode(ctx context.Context) error {
	// Validate required flags
	if *serviceType == "" {
		return fmt.Errorf("--service is required in execute mode")
	}
	if *actionName == "" {
		return fmt.Errorf("--action is required in execute mode")
	}

	if !*executeQuiet {
		log.Println("CLI execution mode")
	}

	// load configuration
	if !*executeQuiet {
		log.Println("Loading configuration...")
	}
	cfg, err := load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if !*executeQuiet {
		log.Printf("Configured for FRITZ!Box at %s", cfg.baseURL())
	}

	// Create TR-064 client
	if !*executeQuiet {
		log.Println("Creating TR-064 client...")
	}
	tr064Client := newClient(cfg.baseURL(), cfg.Username, cfg.Password, *debug)

	// Discover services
	if !*executeQuiet {
		log.Println("Discovering TR-064 services...")
	}
	registry, err := loadFromFritz(ctx, cfg.baseURL())
	if err != nil {
		return fmt.Errorf("failed to load TR-064 services: %w", err)
	}
	if !*executeQuiet {
		log.Printf("Discovered %d services", len(registry.Services))
	}

	// Get service and action specs
	serviceSpec, ok := registry.Services[*serviceType]
	if !ok {
		return fmt.Errorf("service not found: %s", *serviceType)
	}

	actionSpec, err := registry.getAction(*serviceType, *actionName)
	if err != nil {
		return fmt.Errorf("action not found: %s", err)
	}

	// Parse arguments
	inputArgs := make(map[string]string)
	if *actionArgs != "" && *actionArgs != "{}" {
		var argsMap map[string]interface{}
		if err := json.Unmarshal([]byte(*actionArgs), &argsMap); err != nil {
			return fmt.Errorf("failed to parse arguments JSON: %w", err)
		}
		for key, val := range argsMap {
			inputArgs[key] = fmt.Sprintf("%v", val)
		}
	}

	// Execute action
	if !*executeQuiet {
		log.Printf("Executing %s:%s...", *serviceType, *actionName)
	}
	result, err := tr064Client.call(ctx, serviceSpec, actionSpec, inputArgs)
	if err != nil {
		return fmt.Errorf("action execution failed: %w", err)
	}

	// Print result to stdout as JSON
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}
	fmt.Println(string(data))

	return nil
}

// logDeviceInfo calls DeviceInfo:GetInfo to log device details at startup
func logDeviceInfo(ctx context.Context, client *client, registry *registry) error {
	// Find DeviceInfo service
	var deviceInfoType string
	for serviceType := range registry.Services {
		if serviceType == "urn:dslforum-org:service:DeviceInfo:1" {
			deviceInfoType = serviceType
			break
		}
	}

	if deviceInfoType == "" {
		return fmt.Errorf("DeviceInfo service not found")
	}

	// Get action spec
	actionSpec, err := registry.getAction(deviceInfoType, "GetInfo")
	if err != nil {
		return err
	}

	serviceSpec := registry.Services[deviceInfoType]

	// call action
	result, err := client.call(ctx, serviceSpec, actionSpec, map[string]string{})
	if err != nil {
		return err
	}

	// Log relevant info
	model := result["NewModelName"]
	version := result["NewSoftwareVersion"]
	serial := result["NewSerialNumber"]

	log.Printf("Connected to: %s (S/N: %s, Firmware: %s)", model, serial, version)

	return nil
}
