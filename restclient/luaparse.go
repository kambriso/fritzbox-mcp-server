// SPDX-License-Identifier: LicenseRef-KSAL-1.0

package restclient

import (
	"fmt"
	"strconv"
	"strings"
)

// apChannel is one channel entry from a radio in APEnv/AP entries.
type apChannel struct {
	Frequency int `json:"frequency"`
	StartCH   int `json:"startch"`
	EndCH     int `json:"endch"`
	UsedCH    int `json:"usedch"`
}

// apEntry is one radio/network entry in the APEnv list. The Box returns a
// Lua table serialization string ("{ ssid = \"...\", rssi = -66, ... }")
// rather than JSON; parsed via parseLuaStruct.
type apEntry struct {
	RadioType    string
	SSID         string
	MAC          string
	RSSI         int
	Channels     []apChannel
	ChannelWidth int
	RadioBand    int
	Caps         int
}

// parseLuaStruct converts the Box's Lua-table string into apEntry.
// Format: "{ key = value, key2 = { nested = 1 }, key3 = "str" }".
func parseLuaStruct(raw string) (*apEntry, error) {
	if len(raw) < 4 {
		return nil, fmt.Errorf("lua struct too short")
	}
	inner := strings.TrimSpace(raw)
	inner = strings.TrimPrefix(inner, "{")
	inner = strings.TrimSuffix(inner, "}")
	e := &apEntry{}
	flat := parseLuaFields(inner)
	e.RadioType = flat["radiotype"]
	e.SSID = flat["ssid"]
	e.MAC = flat["mac"]
	e.RSSI = atoi(flat["rssi"])
	e.ChannelWidth = atoi(flat["channel_width"])
	e.RadioBand = atoi(flat["radioband"])
	e.Caps = atoi(flat["caps"])
	if rawCh, ok := flat["channels"]; ok {
		// channels = {{ ... },{ ... }}
		inner := strings.TrimPrefix(rawCh, "{")
		inner = strings.TrimSuffix(inner, "}")
		for _, c := range strings.Split(inner, "},{") {
			fields := parseLuaFields(strings.Trim(c, "{} "))
			e.Channels = append(e.Channels, apChannel{
				Frequency: atoi(fields["frequency"]),
				StartCH:   atoi(fields["startch"]),
				EndCH:     atoi(fields["endch"]),
				UsedCH:    atoi(fields["usedch"]),
			})
		}
	}
	return e, nil
}

// atoi parses s as an int, returning 0 on failure (Lua numeric fields).
func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// parseLuaFields splits a flat Lua table body ("k = v, k2 = { ... }") into
// key -> trimmed-string-value pairs. Nested tables stay as raw text; the
// caller re-parses them if needed. Tolerates the Box's inconsistently quoted
// and spaced serialization.
func parseLuaFields(s string) map[string]string {
	out := map[string]string{}
	split := func(body string) []string {
		var parts []string
		depth := 0
		var cur strings.Builder
		for _, ch := range []byte(body) {
			switch ch {
			case '{':
				depth++
				cur.WriteByte(ch)
			case '}':
				depth--
				cur.WriteByte(ch)
			case ',':
				if depth == 0 {
					parts = append(parts, cur.String())
					cur.Reset()
				} else {
					cur.WriteByte(ch)
				}
			default:
				cur.WriteByte(ch)
			}
		}
		if cur.Len() > 0 {
			parts = append(parts, cur.String())
		}
		return parts
	}
	for _, p := range split(s) {
		kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
		if len(kv) == 2 {
			out[strings.TrimSpace(kv[0])] = strings.Trim(strings.TrimSpace(kv[1]), `" `)
		}
	}
	return out
}
