package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	fritzbox "github.com/jhinrichsen/fritzbox-mcp-server"
)

// Test program to verify FRITZ!Box connection and device info retrieval
func main() {
	log.SetOutput(os.Stderr)

	fmt.Println("=== FRITZ!Box MCP Server - Device Info Test ===")

	// Step 1: Load configuration
	fmt.Println("1. Loading configuration from .env...")
	cfg, err := fritzbox.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	fmt.Printf("   ✓ Configured for: %s\n\n", cfg.BaseURL())

	// Step 2: Create TR-064 client
	fmt.Println("2. Creating TR-064 client...")
	tr064Client := fritzbox.NewClient(cfg.BaseURL(), cfg.Username, cfg.Password, false)
	fmt.Println("   ✓ Client created")

	// Step 3: Discover services
	fmt.Println("3. Discovering services from FRITZ!Box...")
	ctx := context.Background()
	registry, err := fritzbox.LoadFromFritz(ctx, cfg.BaseURL())
	if err != nil {
		log.Fatalf("Failed to load services: %v", err)
	}
	fmt.Printf("   ✓ Discovered %d services\n\n", len(registry.Services))

	// Step 4: List all services
	fmt.Println("4. Available services:")
	for serviceType := range registry.Services {
		fmt.Printf("   - %s\n", serviceType)
	}
	fmt.Println()

	// Step 5: Get DeviceInfo
	fmt.Println("5. Retrieving device information...")
	deviceInfo, err := getDeviceInfo(ctx, tr064Client, registry)
	if err != nil {
		log.Fatalf("Failed to get device info: %v", err)
	}

	// Step 6: Display results
	fmt.Println("\n=== Device Information ===")

	displayField("Manufacturer", deviceInfo["NewManufacturerName"])
	displayField("Model", deviceInfo["NewModelName"])
	displayField("Serial Number", deviceInfo["NewSerialNumber"])
	displayField("Software Version", deviceInfo["NewSoftwareVersion"])
	displayField("Hardware Version", deviceInfo["NewHardwareVersion"])
	displayField("Spec Version", deviceInfo["NewSpecVersion"])
	displayField("Uptime (seconds)", deviceInfo["NewUpTime"])

	fmt.Println("\n=== Full Response (JSON) ===")
	prettyJSON, err := json.MarshalIndent(deviceInfo, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}
	fmt.Println(string(prettyJSON))

	fmt.Println("\n✓ Test completed successfully!")
}

func getDeviceInfo(ctx context.Context, client *fritzbox.Client, registry *fritzbox.Registry) (map[string]string, error) {
	// Find DeviceInfo service
	var deviceInfoType string
	for serviceType := range registry.Services {
		if serviceType == "urn:dslforum-org:service:DeviceInfo:1" {
			deviceInfoType = serviceType
			break
		}
	}

	if deviceInfoType == "" {
		return nil, fmt.Errorf("DeviceInfo service not found")
	}

	// Get action spec
	actionSpec, err := registry.GetAction(deviceInfoType, "GetInfo")
	if err != nil {
		return nil, fmt.Errorf("GetInfo action not found: %w", err)
	}

	serviceSpec := registry.Services[deviceInfoType]

	// Call action
	result, err := client.Call(ctx, serviceSpec, actionSpec, map[string]string{})
	if err != nil {
		return nil, fmt.Errorf("API call failed: %w", err)
	}

	return result, nil
}

func displayField(label, value string) {
	if value != "" {
		fmt.Printf("%-20s: %s\n", label, value)
	}
}
