// SPDX-License-Identifier: LicenseRef-KSAL-1.0

// Package-level REST client for the Box's api/v0 interfaces (web-UI
// port), with AVM-SID auth from BoxAuth (login_sid.lua). Separate from
// TR-064 digest auth. Lua-table parsing lives in luaparse.go.
// The oapi-codegen generated client lives in client.gen.go.
package restclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type RestClient struct {
	auth   *BoxAuth
	gen    *Client
	genErr error
	debug  bool
}

func NewRestClient(baseURL, username, password string, debug bool) *RestClient {
	auth := NewBoxAuth(baseURL, username, password, debug)
	c := &RestClient{auth: auth, debug: debug}
	c.gen, c.genErr = NewClient(baseURL,
		WithHTTPClient(auth.httpClient),
		WithRequestEditorFn(auth.Editor()))
	return c
}

// WithReauth runs op once, then, if op reported a 401/403 (SID invalidated),
// again with a fresh login; a second failure is returned as-is.
func (c *RestClient) WithReauth(ctx context.Context, op func(context.Context) error) error {
	if c.genErr != nil {
		return fmt.Errorf("building rest client: %w", c.genErr)
	}
	if err := op(ctx); err == nil || !isUnauthorized(err) {
		return err
	}
	c.debugf("retrying REST call after re-authentication")
	return op(ctx)
}

func (c *RestClient) debugf(format string, args ...interface{}) {
	if c.debug {
		log.Printf("DEBUG(rest): "+format, args...)
	}
}

// GetChannelEnvironment performs an on-demand neighbor scan via the REST
// api and returns the parsed environment entries. Status codes observed:
// 0=done, 1/3=busy, 2=error.
func (c *RestClient) GetChannelEnvironment(ctx context.Context, timeout time.Duration) ([]apEntry, string, error) {
	// Trigger scan (idempotent if already running).
	if err := c.triggerScan(ctx); err != nil {
		return nil, "", err
	}
	deadline := time.Now().Add(timeout)
	for {
		status, raw, err := c.fetchAPEnv(ctx)
		if err != nil {
			return nil, "", err
		}
		if status != "1" && status != "3" {
			return raw, status, nil
		}
		if time.Now().After(deadline) {
			return raw, status, fmt.Errorf("scan did not complete within timeout (status=%s)", status)
		}
		select {
		case <-ctx.Done():
			return nil, "", ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

// triggerScan issues PUT {"scan_apenv":"1"} via the oapi-codegen client,
// retrying once with a fresh login on 401/403.
func (c *RestClient) triggerScan(ctx context.Context) error {
	return c.WithReauth(ctx, func(ctx context.Context) error {
		resp, err := c.gen.PutApiV0GenericWlan(ctx, PutApiV0GenericWlanJSONRequestBody{
			ScanApenv: "1",
		})
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		switch resp.StatusCode {
		case http.StatusOK:
			return nil
		case http.StatusForbidden, http.StatusUnauthorized:
			c.auth.Invalidate()
			return errUnauthorized{op: "scan trigger"}
		default:
			return fmt.Errorf("scan trigger failed (%d): %s", resp.StatusCode, string(body))
		}
	})
}

// fetchAPEnv issues GET via the oapi-codegen client and parses APEnv entries,
// retrying once with a fresh login on 401/403.
func (c *RestClient) fetchAPEnv(ctx context.Context) (string, []apEntry, error) {
	var status string
	var entries []apEntry
	err := c.WithReauth(ctx, func(ctx context.Context) error {
		resp, err := c.gen.GetApiV0GenericWlan(ctx)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
			c.auth.Invalidate()
			return errUnauthorized{op: "APEnv read"}
		}
		var payload WlanUIMod
		if uerr := json.Unmarshal(body, &payload); uerr != nil {
			var env ErrorEnvelope
			if eerr := json.Unmarshal(body, &env); eerr == nil && len(env.Errors) > 0 {
				return fmt.Errorf("REST error: %v", env.Errors[0].Message)
			}
			return fmt.Errorf("parsing APEnv REST response (%d): %s", resp.StatusCode, string(body))
		}
		for _, item := range payload.APEnv {
			if item.Params == "" {
				continue
			}
			if entry, lerr := parseLuaStruct(item.Params); lerr == nil {
				entries = append(entries, *entry)
			}
		}
		status = payload.APEnvStatus
		return nil
	})
	return status, entries, err
}
