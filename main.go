package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

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
	fetchXMLOnly = flag.Bool("fetch-xml", false, "Fetch TR-064 XML files from network-attached FRITZ!Box and exit")
	xmlDir       = flag.String("xml-dir", "", "Directory to store fetched XML files (default: ~/.cache/fritzbox-mcp-server)")
	debug        = flag.Bool("debug", false, "Enable verbose debug logging")
	showVersion  = flag.Bool("version", false, "Show version information and exit")
	// CLI execution mode
	executeMode  = flag.Bool("execute", false, "Execute a single action and exit (CLI mode)")
	serviceType  = flag.String("service", "", "Service type for CLI execution (e.g., urn:dslforum-org:service:DeviceInfo:1)")
	actionName   = flag.String("action", "", "Action name for CLI execution (e.g., GetInfo)")
	actionArgs   = flag.String("args", "{}", "Action arguments as JSON object for CLI execution")
	executeQuiet = flag.Bool("quiet", false, "Suppress log output in execute mode (only print result)")
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
	if *executeQuiet {
		log.SetOutput(os.Stderr)
	} else {
		log.SetOutput(os.Stderr) // MCP uses stdout for protocol, stderr for logs
	}

	// Set default XML directory if not specified
	if *xmlDir == "" {
		*xmlDir = getCacheDir()
	}

	ctx := context.Background()

	// If --execute flag is set, execute action and exit (CLI mode)
	if *executeMode {
		return runExecuteMode(ctx)
	}

	// If --fetch-xml flag is set, only fetch XML and exit (no auth required)
	if *fetchXMLOnly {
		// Try to load minimal config (only FRITZ_HOST needed)
		cfg, _ := load()
		var baseURL string
		if cfg == nil {
			// load failed, try environment directly
			host := os.Getenv("FRITZ_HOST")
			if host == "" {
				return fmt.Errorf("FRITZ_HOST is required (set in .env or environment)")
			}
			port := os.Getenv("FRITZ_PORT")
			if port == "" {
				port = "49000"
			}
			baseURL = fmt.Sprintf("http://%s:%s", host, port)
		} else {
			baseURL = cfg.baseURL()
		}

		log.Printf("Fetching TR-064 XML files from %s to %s/", baseURL, *xmlDir)
		if err := fetchAllXML(ctx, baseURL, *xmlDir); err != nil {
			return fmt.Errorf("failed to fetch XML: %w", err)
		}
		log.Println("✓ XML files fetched successfully")

		// List downloaded files
		entries, err := os.ReadDir(*xmlDir)
		if err != nil {
			return fmt.Errorf("failed to read XML directory: %w", err)
		}

		log.Printf("Downloaded %d files:", len(entries))
		for _, entry := range entries {
			info, _ := entry.Info()
			if info != nil {
				log.Printf("  - %s (%d bytes)", entry.Name(), info.Size())
			}
		}
		return nil
	}

	// Step 1: load full configuration (requires auth credentials)
	log.Println("Loading configuration...")
	cfg, err := load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	log.Printf("Configured for FRITZ!Box at %s", cfg.baseURL())

	// Step 2: Create TR-064 client
	log.Println("Creating TR-064 client...")
	tr064Client := newClient(cfg.baseURL(), cfg.Username, cfg.Password, *debug)

	// Step 3: Discover services from FRITZ!Box
	log.Println("Discovering TR-064 services from FRITZ!Box...")
	registry, err := loadFromFritz(ctx, cfg.baseURL())
	if err != nil {
		return fmt.Errorf("failed to load TR-064 services: %w", err)
	}
	log.Printf("Discovered %d services", len(registry.Services))

	// Step 4: Optional - call DeviceInfo:GetInfo to log device details
	log.Println("Fetching device information...")
	if err := logDeviceInfo(ctx, tr064Client, registry); err != nil {
		log.Printf("Warning: could not get device info: %v", err)
	}

	// Step 5: Create documentation index
	log.Println("Loading documentation index...")
	docsIndex := newIndex()

	// Step 6: Create and start MCP server
	log.Printf("Starting MCP server v%s...\n", version)
	mcpSrv := newServer(serverName, version, tr064Client, registry, docsIndex)

	log.Println("MCP server ready")
	return server.ServeStdio(mcpSrv.getMCPServer())
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
