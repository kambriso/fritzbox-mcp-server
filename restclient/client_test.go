// SPDX-License-Identifier: LicenseRef-KSAL-1.0

package restclient

import (
	"context"
	"crypto/pbkdf2"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBuildLoginResponseLegacyMD5(t *testing.T) {
	// Pre-7.24 challenge: no $ separators
	resp, err := buildLoginResponse("1234567", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(resp, "1234567-") {
		t.Fatalf("expected challenge-prefixed response, got %q", resp)
	}
	if len(resp)-len("1234567-") != 32 {
		t.Fatalf("expected 32-hex md5, got %q", resp)
	}
}

func TestBuildLoginResponsePBKDF2(t *testing.T) {
	// Canonical AVM 7.24+ format: 2$<iter1>$<salt1>$<iter2>$<salt2>
	challenge := "2$60000$9765b0565d3bff070f972ef2234d7f15$6000$9a009b1c4840044161ed4d802269a93b"
	resp, err := buildLoginResponse(challenge, "secret")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(resp, "$")
	if len(parts) != 2 {
		t.Fatalf("expected 2 fields, got %d: %q", len(parts), resp)
	}
	if parts[0] != "9a009b1c4840044161ed4d802269a93b" {
		t.Fatalf("response must start with salt2, got %q", resp)
	}
	salt1, _ := hex.DecodeString("9765b0565d3bff070f972ef2234d7f15")
	salt2, _ := hex.DecodeString("9a009b1c4840044161ed4d802269a93b")
	wantstatic, _ := pbkdf2.Key(sha256.New, "secret", salt1, 60000, 32)
	want, _ := pbkdf2.Key(sha256.New, string(wantstatic), salt2, 6000, 32)
	if got, _ := hex.DecodeString(parts[1]); string(got) != string(want) {
		t.Fatalf("dynamic hash mismatch\nwant %x\ngot  %x", want, got)
	}
}

func makeFakeRESTBox(t *testing.T) *httptest.Server {
	t.Helper()
	const challenge = "2$60000$9765b0565d3bff070f972ef2234d7f15$6000$9a009b1c4840044161ed4d802269a93b"
	status := "1"
	scanCalls := 0
	apenv := `{ "params": "{ radiotype = 1,channels = {{ frequency = 2462,endch = 13,startch = 5,usedch = 11 }},channel_width = 40,radioband = 1 }" },
{ "params": "{ radiotype = 2,ssid = \"TESTNEIGHBOR-WIFI\",channels = {{ frequency = 2412,endch = 3,startch = 1,usedch = 1 }},channel_width = 20,mac = \"02:11:22:33:44:55\",radioband = 1,rssi = -75,caps = 2229250 }" }`
	mux := http.NewServeMux()
	mux.HandleFunc("/login_sid.lua", func(w http.ResponseWriter, r *http.Request) {
		body := func(sid string) string {
			return "<?xml version=\"1.0\"?><SessionInfo><SID>" + sid + "</SID><Challenge>" + challenge + "</Challenge><BlockTime>0</BlockTime></SessionInfo>"
		}
		if r.Method == "GET" {
			w.Write([]byte(body("0000000000000000")))
			return
		}
		w.Write([]byte(body("abcd1234")))
	})
	mux.HandleFunc("/api/v0/generic/wlan", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "PUT" {
			scanCalls++
			w.Write([]byte(`{}`))
			return
		}
		if scanCalls > 0 {
			status = "0"
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"APEnvStatus": status,
			"APEnv":       json.RawMessage("[" + apenv + "]"),
		})
	})
	return httptest.NewServer(mux)
}

func TestGetChannelEnvironmentParses(t *testing.T) {
	srv := makeFakeRESTBox(t)
	defer srv.Close()
	c := NewRestClient(srv.URL, "admin", "pw", false)
	entries, status, err := c.GetChannelEnvironment(context.Background(), 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if status != "0" {
		t.Fatalf("expected done status 0, got %q", status)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 APEnv entries, got %d", len(entries))
	}
	own, env := entries[0], entries[1]
	if own.RadioType != "1" || own.ChannelWidth != 40 || len(own.Channels) != 1 || own.Channels[0].UsedCH != 11 {
		t.Fatalf("bad own radio entry: %+v", own)
	}
	if env.SSID != "TESTNEIGHBOR-WIFI" || env.RSSI != -75 || env.MAC != "02:11:22:33:44:55" {
		t.Fatalf("bad neighbor entry: %+v", env)
	}
	if env.Channels[0].UsedCH != 1 || env.ChannelWidth != 20 {
		t.Fatalf("bad neighbor channel: %+v", env.Channels)
	}
}

func TestGetChannelEnvironmentReauthenticates(t *testing.T) {
	srv := makeFakeExpiringRESTBox(t)
	defer srv.Close()
	c := NewRestClient(srv.URL, "admin", "pw", false)
	entries, status, err := c.GetChannelEnvironment(context.Background(), 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if status != "0" || len(entries) != 1 {
		t.Fatalf("expected done status 0 with 1 entry, got status=%q entries=%d", status, len(entries))
	}
	c.auth.mu.Lock()
	got := c.auth.sid
	c.auth.mu.Unlock()
	if got == "expired" {
	}
}

// makeFakeExpiringRESTBox serves one valid SID, rejects further calls with 403,
// then issues a fresh SID on re-login; exercises WithReauth.

func makeFakeExpiringRESTBox(t *testing.T) *httptest.Server {
	t.Helper()
	const challenge = "2$60000$9765b0565d3bff070f972ef2234d7f15$6000$9a009b1c4840044161ed4d802269a93b"
	logins := 0
	validSID := "" // set after first login
	mux := http.NewServeMux()
	mux.HandleFunc("/login_sid.lua", func(w http.ResponseWriter, r *http.Request) {
		body := func(sid string) string {
			return "<?xml version=\"1.0\"?><SessionInfo><SID>" + sid + "</SID><Challenge>" + challenge + "</Challenge><BlockTime>0</BlockTime></SessionInfo>"
		}
		if r.Method == "GET" {
			w.Write([]byte(body("0000000000000000")))
			return
		}
		logins++
		if logins == 1 {
			validSID = "expired"
		} else {
			validSID = "fresh" + fmt.Sprint(logins)
		}
		w.Write([]byte(body(validSID)))
	})
	mux.HandleFunc("/api/v0/generic/wlan", func(w http.ResponseWriter, r *http.Request) {
		sid := strings.TrimPrefix(r.Header.Get("Authorization"), "AVM-SID ")
		if sid == "" || sid == "expired" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "PUT" {
			w.Write([]byte(`{}`))
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"APEnvStatus": "0",
			"APEnv":       json.RawMessage(`[{ "params": "{ radiotype = 1,channels = {{ frequency = 2412,endch = 3,startch = 1,usedch = 1 }},channel_width = 20,radioband = 1 }" }]`),
		})
	})
	return httptest.NewServer(mux)
}
