// SPDX-License-Identifier: LicenseRef-KSAL-1.0

package main

import (
	"context"
	"testing"
)

// BenchmarkDiscovery measures the time to perform a full dynamic discovery
// of all services and actions from the FRITZ!Box.
// Run with: go test -v -bench=BenchmarkDiscovery .
func BenchmarkDiscovery(b *testing.B) {
	cfg, err := load()
	if err != nil {
		b.Skip("Skipping benchmark: no FRITZ!Box configuration found (.env)")
	}

	ctx := context.Background()
	baseURL := cfg.baseURL()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry, err := loadFromFritz(ctx, baseURL)
		if err != nil {
			b.Fatalf("Discovery failed: %v", err)
		}
		if len(registry.Services) == 0 {
			b.Fatal("Discovered 0 services")
		}
	}
}
