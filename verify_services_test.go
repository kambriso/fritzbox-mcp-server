package main

import (
	"encoding/xml"
	"os"
	"testing"
)

// TestVerifyAllServicesDiscovered ensures we capture all services from tr64desc.xml
func TestVerifyAllServicesDiscovered(t *testing.T) {
	// load tr64desc.xml from testdata
	data, err := os.ReadFile("testdata/xml/tr64desc.xml")
	if err != nil {
		t.Fatalf("Failed to read tr64desc.xml: %v", err)
	}

	var root tr64Root
	if err := xml.Unmarshal(data, &root); err != nil {
		t.Fatalf("Failed to parse tr64desc.xml: %v", err)
	}

	// Use our recursive collection function
	services := collectServicesRecursive(&root.Device)

	// Expected services based on tr64desc.xml structure:
	// Root InternetGatewayDevice: 24 services
	// LANDevice: 6 services (3x WLANConfiguration, Hosts, LANEthernetInterfaceConfig, LANHostConfigManagement)
	// WANDevice: 2 services (WANCommonInterfaceConfig, WANDSLInterfaceConfig)
	// WANConnectionDevice: 5 services
	expectedCount := 37

	if len(services) != expectedCount {
		t.Errorf("Expected %d services, got %d", expectedCount, len(services))

		// Log all discovered services for debugging
		t.Logf("Discovered %d services:", len(services))
		for i, svc := range services {
			t.Logf("  %d. %s -> %s", i+1, svc.ServiceType, svc.ControlURL)
		}
	}

	// Verify critical services are present
	criticalServices := []string{
		"urn:dslforum-org:service:DeviceInfo:1",
		"urn:dslforum-org:service:Hosts:1",
		"urn:dslforum-org:service:WLANConfiguration:1",
		"urn:dslforum-org:service:WLANConfiguration:2",
		"urn:dslforum-org:service:WLANConfiguration:3",
		"urn:dslforum-org:service:WANIPConnection:1",
		"urn:dslforum-org:service:WANPPPConnection:1",
		"urn:dslforum-org:service:LANHostConfigManagement:1",
		"urn:dslforum-org:service:X_AVM-DE_Homeauto:1",
	}

	serviceMap := make(map[string]bool)
	for _, svc := range services {
		serviceMap[svc.ServiceType] = true
	}

	for _, criticalSvc := range criticalServices {
		if !serviceMap[criticalSvc] {
			t.Errorf("Critical service not found: %s", criticalSvc)
		}
	}
}
