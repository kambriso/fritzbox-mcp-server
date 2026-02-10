// SPDX-License-Identifier: LicenseRef-KSAL-1.0

package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// discoveryResult represents a discovered FRITZ!Box
type discoveryResult struct {
	IP         string
	Port       string
	Model      string
	Location   string
	IsMaster   bool
	IsLikelyGW bool
}

const (
	ssdpIPv4Multicast = "239.255.255.250:1900"
	ssdpIPv6Multicast = "[ff02::c]:1900"
)

// discoverFritzBox searches for FRITZ!Box devices on the network using SSDP
func discoverFritzBox(ctx context.Context, timeout time.Duration) ([]discoveryResult, error) {
	results := make(map[string]*discoveryResult)

	// Listen on all interfaces
	pc4, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		return nil, fmt.Errorf("listening on UDP4: %w", err)
	}
	defer pc4.Close()

	pc6, err := net.ListenPacket("udp6", ":0")
	if err != nil {
		// Log but continue - IPv6 might not be available
	} else {
		defer pc6.Close()
	}

	// M-SEARCH request
	msearch := "M-SEARCH * HTTP/1.1\r\n" +
		"HOST: 239.255.255.250:1900\r\n" +
		"MAN: \"ssdp:discover\"\r\n" +
		"MX: 2\r\n" +
		"ST: ssdp:all\r\n" +
		"\r\n"

	// Send IPv4 multicast
	addr4, _ := net.ResolveUDPAddr("udp4", ssdpIPv4Multicast)
	_, _ = pc4.WriteTo([]byte(msearch), addr4)

	// Send IPv6 multicast
	if pc6 != nil {
		addr6, _ := net.ResolveUDPAddr("udp6", ssdpIPv6Multicast)
		_, _ = pc6.WriteTo([]byte(msearch), addr6)
	}

	// Collect responses
	done := make(chan bool)
	go func() {
		buf := make([]byte, 2048)
		for {
			_ = pc4.SetReadDeadline(time.Now().Add(timeout))
			n, addr, err := pc4.ReadFrom(buf)
			if err != nil {
				break
			}
			processSSDPResponse(string(buf[:n]), addr.String(), results)
		}
		done <- true
	}()

	if pc6 != nil {
		go func() {
			buf := make([]byte, 2048)
			for {
				_ = pc6.SetReadDeadline(time.Now().Add(timeout))
				n, addr, err := pc6.ReadFrom(buf)
				if err != nil {
					break
				}
				processSSDPResponse(string(buf[:n]), addr.String(), results)
			}
			done <- true
		}()
	}

	// Wait for timeout or context cancellation
	select {
	case <-time.After(timeout):
	case <-ctx.Done():
	}

	// Convert map to slice
	finalResults := []discoveryResult{}
	for _, res := range results {
		// Final scoring/hints
		if strings.HasSuffix(res.IP, ".1") {
			res.IsLikelyGW = true
		}
		finalResults = append(finalResults, *res)
	}

	return finalResults, nil
}

func processSSDPResponse(raw, from string, results map[string]*discoveryResult) {
	resp, err := http.ReadResponse(bufio.NewReader(strings.NewReader(raw)), nil)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	server := resp.Header.Get("Server")
	if !strings.Contains(server, "FRITZ!Box") {
		return
	}

	location := resp.Header.Get("Location")
	if location == "" {
		return
	}

	u, err := url.Parse(location)
	if err != nil {
		return
	}

	host, port, _ := net.SplitHostPort(u.Host)
	if host == "" {
		host = u.Host
	}

	// Create or update result
	res, ok := results[host]
	if !ok {
		res = &discoveryResult{
			IP:       host,
			Port:     port,
			Location: location,
		}
		results[host] = res
	}

	// Extract model from Server header if available
	// e.g. "FRITZ!Box 7590 UPnP/1.0 AVM FRITZ!Box 7590 154.08.20"
	if res.Model == "" {
		parts := strings.Split(server, " ")
		for i, p := range parts {
			if p == "FRITZ!Box" && i+1 < len(parts) {
				res.Model = parts[i+1]
				break
			}
		}
	}

	// Detect Mesh Master
	if strings.Contains(location, "l2tpv3.xml") {
		res.IsMaster = true
	}
}
