// SPDX-License-Identifier: LicenseRef-KSAL-1.0

package restclient

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// BoxAuth manages a FRITZ!Box web session (login_sid.lua / AVM-SID):
// it mints SIDs via challenge-response login and injects them into
// outgoing requests, suitable as a request-editor fn for the generated
// REST client.
type BoxAuth struct {
	baseURL    string
	Username   string
	Password   string
	httpClient *http.Client

	mu  sync.Mutex
	sid string
}

// NewBoxAuth builds a session manager for the box at baseURL
// (scheme://host). debug enables request logging; the shared HTTP
// client carries the scrubbed transport.
func NewBoxAuth(baseURL, username, password string, debug bool) *BoxAuth {
	return &BoxAuth{
		baseURL:  baseURL,
		Username: username,
		Password: password,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: newScrubTransport(http.DefaultTransport),
		},
	}
}

// Editor returns a request-editor fn (WithRequestEditorFn)
// that sets the Authorization header from the cached SID.
func (a *BoxAuth) Editor() func(context.Context, *http.Request) error {
	return a.editor
}

// editor sets the Authorization header from the cached SID.
func (a *BoxAuth) editor(ctx context.Context, req *http.Request) error {
	sid, err := a.SID(ctx)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "AVM-SID "+sid)
	return nil
}

// Invalidate clears a stale session so the next call re-authenticates.
func (a *BoxAuth) Invalidate() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sid = ""
}

// SID returns a valid session ID, logging in if necessary.
func (a *BoxAuth) SID(ctx context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.sid != "" {
		return a.sid, nil
	}
	sess, err := a.fetchSession(ctx, http.MethodGet, nil)
	if err != nil {
		return "", err
	}
	if sess.BlockTime > 0 {
		return "", fmt.Errorf("login blocked for %d seconds", sess.BlockTime)
	}
	respStr, err := buildLoginResponse(sess.Challenge, a.Password)
	if err != nil {
		return "", fmt.Errorf("building login response: %w", err)
	}
	form := url.Values{}
	form.Set("username", a.Username)
	form.Set("response", respStr)
	sess, err = a.fetchSession(ctx, http.MethodPost, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	if sess.SID == "" || sess.SID == "0000000000000000" {
		return "", fmt.Errorf("login failed (check username/password)")
	}
	a.sid = sess.SID
	return a.sid, nil
}

// fetchSession performs one login_sid.lua round-trip and parses the XML
// SessionInfo result.
func (a *BoxAuth) fetchSession(ctx context.Context, method string, body io.Reader) (sessionInfo, error) {
	req, err := http.NewRequestWithContext(ctx, method,
		a.baseURL+"/login_sid.lua?version=2", body)
	if err != nil {
		return sessionInfo{}, fmt.Errorf("creating login request: %w", err)
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return sessionInfo{}, fmt.Errorf("login request: %w", err)
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return sessionInfo{}, fmt.Errorf("reading login response: %w", err)
	}
	var sess sessionInfo
	if err := xml.Unmarshal(payload, &sess); err != nil {
		return sessionInfo{}, fmt.Errorf("parsing login response: %w", err)
	}
	return sess, nil
}
