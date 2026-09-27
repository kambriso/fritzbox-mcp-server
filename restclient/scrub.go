// SPDX-License-Identifier: LicenseRef-KSAL-1.0

package restclient

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

// secretKeys lists FRITZ!OS REST payload fields that carry credentials
// (Wi-Fi passphrases and their "_scnd" mirrors; every UIMod scalar keys a
// second-radio variant).
var secretKeys = []string{
	"pskvalue", "guest_pskvalue", "STA_pskvalue", "repeater_pskvalue",
	"pskvalue_scnd", "guest_pskvalue_scnd", "STA_pskvalue_scnd", "repeater_pskvalue_scnd",
}

// scrubSecrets replaces secret-bearing top-level fields in a JSON body with
// a redacted placeholder. Bodies that are not JSON objects (login_sid.lua
// XML, TR-064 SOAP envelopes, error pages) pass through unchanged.
func scrubSecrets(body []byte) []byte {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return body
	}
	scrubbed := false
	for _, key := range secretKeys {
		if _, ok := raw[key]; ok {
			raw[key] = json.RawMessage(`"[REDACTED]"`)
			scrubbed = true
		}
	}
	if !scrubbed {
		return body
	}
	out, err := json.Marshal(raw)
	if err != nil {
		return body
	}
	return out
}

// scrubTransport wraps the base transport so every response body is fully
// buffered and secret fields are redacted before any client code, error
// message, or log line can see them.
type scrubTransport struct {
	next http.RoundTripper
}

// newScrubTransport wraps next (http.DefaultTransport when nil).
func newScrubTransport(next http.RoundTripper) scrubTransport {
	if next == nil {
		next = http.DefaultTransport
	}
	return scrubTransport{next: next}
}

func (t scrubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.next.RoundTrip(req)
	if err != nil || resp.Body == nil {
		return resp, err
	}
	body, readErr := io.ReadAll(resp.Body)
	resp.Body.Close()
	if readErr != nil {
		return nil, readErr
	}
	resp.Body = io.NopCloser(bytes.NewReader(scrubSecrets(body)))
	return resp, nil
}
