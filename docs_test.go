package main

import (
	"strings"
	"testing"
)

func TestGenerateDocumentation(t *testing.T) {
	tests := []struct {
		service  string
		action   string
		expected string
	}{
		{"DeviceInfo", "GetInfo", "Get device information"},
		{"Hosts", "GetHostNumberOfEntries", "Get host number of entries from network hosts"},
		{"WLANConfiguration", "GetInfo", "Get WiFi information"},
		{"WLANConfiguration", "GetSSID", "Get ssid from WiFi"},
		{"X_AVM-DE_Homeauto", "GetGenericDeviceInfos", "Get generic device infos from smart home"},
		{"WANIPConnection", "GetStatusInfo", "Get status info from WAN IP connection"},
		{"Layer3Forwarding", "GetDefaultConnectionService", "Get default connection service from port forwarding"},
	}

	for _, tt := range tests {
		t.Run(tt.service+"."+tt.action, func(t *testing.T) {
			result := generateDocumentation(tt.service, tt.action)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("generateDocumentation(%q, %q) = %q, want to contain %q",
					tt.service, tt.action, result, tt.expected)
			}
		})
	}
}

func TestGenerateDetailedDocumentation(t *testing.T) {
	tests := []struct {
		service  string
		action   string
		inArgs   []string
		outArgs  []string
		contains []string
	}{
		{
			service:  "DeviceInfo",
			action:   "GetInfo",
			inArgs:   []string{},
			outArgs:  []string{"NewManufacturerName", "NewModelName", "NewSerialNumber"},
			contains: []string{"device information", "manufacturer name", "model name", "serial number"},
		},
		{
			service:  "Hosts",
			action:   "GetGenericHostEntry",
			inArgs:   []string{"NewIndex"},
			outArgs:  []string{"NewIPAddress", "NewMACAddress", "NewActive"},
			contains: []string{"network hosts", "Input: index", "ip address", "mac address"},
		},
		{
			service:  "WLANConfiguration",
			action:   "SetEnable",
			inArgs:   []string{"NewEnable"},
			outArgs:  []string{},
			contains: []string{"WiFi", "Input: enable"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.service+"."+tt.action, func(t *testing.T) {
			result := generateDetailedDocumentation(tt.service, tt.action, tt.inArgs, tt.outArgs)
			for _, expected := range tt.contains {
				if !strings.Contains(strings.ToLower(result), strings.ToLower(expected)) {
					t.Errorf("generateDetailedDocumentation() = %q, want to contain %q", result, expected)
				}
			}
		})
	}
}

func TestHumanizeServiceName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"DeviceInfo", "device"},
		{"Hosts", "network hosts"},
		{"WLANConfiguration", "WiFi"},
		{"X_AVM-DE_Homeauto", "smart home"},
		{"X_AVM-DE_OnTel", "telephony"},
		{"LANHostConfigManagement", "LAN host configuration"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := humanizeServiceName(tt.input)
			if result != tt.expected {
				t.Errorf("humanizeServiceName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestHumanizeCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"GetInfo", "get info"},
		{"HostNumberOfEntries", "host number of entries"},
		{"NewIPAddress", "new ip address"},
		{"X_AVM-DE_GetInfo", "get info"},
		{"SSID", "ssid"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := humanizeCamelCase(tt.input)
			if result != tt.expected {
				t.Errorf("humanizeCamelCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDocsIndex(t *testing.T) {
	idx := newIndex()

	// Test basic lookup
	doc := idx.lookup("urn:dslforum-org:service:DeviceInfo:1", "GetInfo")
	if !strings.Contains(doc, "device") {
		t.Errorf("Expected documentation to contain 'device', got: %s", doc)
	}

	// Test lookup with args
	detailedDoc := idx.lookupWithArgs(
		"urn:dslforum-org:service:Hosts:1",
		"GetGenericHostEntry",
		[]string{"NewIndex"},
		[]string{"NewIPAddress", "NewMACAddress"},
	)
	if !strings.Contains(strings.ToLower(detailedDoc), "input") {
		t.Errorf("Expected detailed documentation to contain 'input', got: %s", detailedDoc)
	}
	if !strings.Contains(strings.ToLower(detailedDoc), "returns") {
		t.Errorf("Expected detailed documentation to contain 'returns', got: %s", detailedDoc)
	}
}
