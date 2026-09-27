// SPDX-License-Identifier: LicenseRef-KSAL-1.0

package restclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// secretPayload is the real wlan UIMod shape with a populated pskvalue.
const secretPayload = `{"APEnvStatus":"0","pskvalue":"hunter2-secret","guest_pskvalue":"","pskvalue_scnd":"","APEnv":[]}`

// fakeSecretBox serves login_sid.lua and a wlan payload carrying pskvalue.
func fakeSecretBox(t *testing.T) *httptest.Server {
	t.Helper()
	const challenge = "2$60000$9765b0565d3bff070f972ef2234d7f15$6000$9a009b1c4840044161ed4d802269a93b"
	mux := http.NewServeMux()
	mux.HandleFunc("/login_sid.lua", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Write([]byte("<?xml version=\"1.0\"?><SessionInfo><SID>0000000000000000</SID><Challenge>" + challenge + "</Challenge><BlockTime>0</BlockTime><Rights></Rights></SessionInfo>"))
			return
		}
		w.Write([]byte("<?xml version=\"1.0\"?><SessionInfo><SID>abcd1234</SID><Challenge>" + challenge + "</Challenge><BlockTime>0</BlockTime><Rights></Rights></SessionInfo>"))
	})
	mux.HandleFunc("/api/v0/generic/wlan", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(secretPayload))
	})
	return httptest.NewServer(mux)
}

func TestScrubTransportRedactsSecrets(t *testing.T) {
	srv := fakeSecretBox(t)
	defer srv.Close()

	// Plain fetch with the scrubbed client: body must be redacted.
	c := NewRestClient(srv.URL, "admin", "pw", false)
	resp, err := c.gen.GetApiV0GenericWlan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	buf := make([]byte, 1024)
	n, _ := resp.Body.Read(buf)
	body := string(buf[:n])
	if strings.Contains(body, "hunter2-secret") {
		t.Fatalf("pskvalue leaked into parsed body: %s", body)
	}
	if !strings.Contains(body, "[REDACTED]") {
		t.Fatalf("expected redaction marker in body: %s", body)
	}
}

func TestScrubSecretsCases(t *testing.T) {
	cases := []struct {
		in     string
		leak   bool
		marker bool
	}{
		{secretPayload, false, true},
		{`{"pskvalue":"x"}`, false, true},
		{`{"pskvalue_scnd":"x","pskvalue":"y"}`, false, true},
		{`{"APEnvStatus":"1"}`, false, false}, // untouched body preserved verbatim
		{`<xml>not json</xml>`, false, false}, // non-JSON passes through
		{`{"errors":[{"code":1,"message":"m"}]}`, false, false},
	}
	for i, tc := range cases {
		out := scrubSecrets([]byte(tc.in))
		if tc.leak && strings.Contains(string(out), "x") {
			t.Errorf("case %d: secret survived: %s", i, out)
		}
		if strings.Contains(string(out), "hunter2-secret") || strings.Contains(string(out), "y") {
			t.Errorf("case %d: secret survived: %s", i, out)
		}
		if tc.marker != strings.Contains(string(out), "[REDACTED]") {
			t.Errorf("case %d: marker mismatch: got %s", i, out)
		}
		// untouched inputs must round-trip byte-identically
		if !tc.marker && string(out) != tc.in {
			t.Errorf("case %d: untouched input changed: %q -> %q", i, tc.in, out)
		}
	}
}

// TestScanErrorDoesNotLeak asserts the error path (unexpected status) cannot
// embed an unscrubbed payload, because the transport redacts before the
// caller stringifies the body.
func TestScanErrorDoesNotLeak(t *testing.T) {
	// A box answering 500 with the secret payload in the error body.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login_sid.lua" {
			const challenge = "2$60000$9765b0565d3bff070f972ef2234d7f15$6000$9a009b1c4840044161ed4d802269a93b"
			if r.Method == "GET" {
				w.Write([]byte("<?xml version=\"1.0\"?><SessionInfo><SID>0000000000000000</SID><Challenge>" + challenge + "</Challenge><BlockTime>0</BlockTime></SessionInfo>"))
				return
			}
			w.Write([]byte("<?xml version=\"1.0\"?><SessionInfo><SID>abcd1234</SID><Challenge>" + challenge + "</Challenge><BlockTime>0</BlockTime></SessionInfo>"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(secretPayload))
	}))
	defer srv.Close()

	c := NewRestClient(srv.URL, "admin", "pw", false)
	_, _, err := c.GetChannelEnvironment(context.Background(), time.Second)
	if err == nil {
		t.Fatal("expected error from 500 response")
	}
	if strings.Contains(err.Error(), "hunter2-secret") {
		t.Fatalf("pskvalue leaked into error text: %s", err)
	}
}

// TestScrubArgNameList asserts the secret key list covers the observed
// payload variants.
func TestScrubArgNameList(t *testing.T) {
	for _, k := range secretKeys {
		body := []byte(`{"` + k + `":"x"}`)
		out := scrubSecrets(body)
		if !strings.Contains(string(out), "[REDACTED]") {
			t.Errorf("key not scrubbed: %s", k)
		}
	}
}
