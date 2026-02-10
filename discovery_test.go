// SPDX-License-Identifier: LicenseRef-KSAL-1.0

package main

import (
	"testing"
)

// TestProcessSSDPResponse verifies the parsing and filtering logic
// using the sample data provided in testdata/ssdp-response.txt
func TestProcessSSDPResponse(t *testing.T) {
	results := make(map[string]*discoveryResult)

	// The sample data has multiple HTTP responses. We need to split and process them.
	// In the real code, these would arrive as individual UDP packets.
	rawResponses := []string{
		"HTTP/1.1 200 OK\r\n" +
			"Cache-Control: max-age=1800\r\n" +
			"Location: http://192.168.178.1:49000/fboxdesc.xml\r\n" +
			"Server: FRITZ!Box 7590 UPnP/1.0 AVM FRITZ!Box 7590 154.08.20\r\n" +
			"Ext: \r\n" +
			"ST: upnp:rootdevice\r\n" +
			"USN: uuid:123402409-bccb-40e7-8e6c-DC396F421B1E::upnp:rootdevice\r\n\r\n",

		"HTTP/1.1 200 OK\r\n" +
			"Cache-Control: max-age=1800\r\n" +
			"Location: http://192.168.178.1:49000/l2tpv3.xml\r\n" +
			"Server: FRITZ!Box 7590 UPnP/1.0 AVM FRITZ!Box 7590 154.08.20\r\n" +
			"Ext: \r\n" +
			"ST: upnp:rootdevice\r\n" +
			"USN: uuid:95802409-bccb-40e7-8e6c-DC396F421B1E::upnp:rootdevice\r\n\r\n",

		"HTTP/1.1 200 OK\r\n" +
			"Cache-Control: max-age=1800\r\n" +
			"Location: http://192.168.178.24:49000/igddesc.xml\r\n" +
			"Server: FRITZ!Box 7490 (UI) UPnP/1.0 AVM FRITZ!Box 7490 (UI) 113.07.60\r\n" +
			"Ext: \r\n" +
			"ST: upnp:rootdevice\r\n" +
			"USN: uuid:75802409-bccb-40e7-8e6c-C02506F15BFC::upnp:rootdevice\r\n\r\n",
	}

	for _, raw := range rawResponses {
		processSSDPResponse(raw, "127.0.0.1:1900", results)
	}

	// Verify we found 2 unique devices
	if len(results) != 2 {
		t.Errorf("Expected 2 devices, got %d", len(results))
	}

	// Verify 192.168.178.1 is the Master
	master, ok := results["192.168.178.1"]
	if !ok {
		t.Fatal("192.168.178.1 not found in results")
	}
	if !master.IsMaster {
		t.Error("Expected 192.168.178.1 to be detected as Mesh Master due to l2tpv3.xml")
	}
	if master.Model != "7590" {
		t.Errorf("Expected model 7590, got %s", master.Model)
	}

	// Verify 192.168.178.24 is NOT the Master
	repeater, ok := results["192.168.178.24"]
	if !ok {
		t.Fatal("192.168.178.24 not found in results")
	}
	if repeater.IsMaster {
		t.Error("Expected 192.168.178.24 NOT to be detected as Mesh Master")
	}
	if repeater.Model != "7490" {
		t.Errorf("Expected model 7490, got %s", repeater.Model)
	}
}

// TestFritzBoxFilter verifies that non-FritzBox devices are ignored
func TestFritzBoxFilter(t *testing.T) {
	results := make(map[string]*discoveryResult)

	badResponse := `HTTP/1.1 200 OK
Location: http://192.168.178.50:80/description.xml
Server: Some Generic Smart Bulb UPnP/1.0`

	processSSDPResponse(badResponse, "192.168.178.50:1900", results)

	if len(results) != 0 {
		t.Errorf("Expected 0 devices (filtered bulb), got %d", len(results))
	}
}
