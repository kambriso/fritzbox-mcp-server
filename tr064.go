package fritzbox

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// Client is a TR-064 SOAP client with authentication support
type Client struct {
	BaseURL    string
	Username   string
	Password   string
	httpClient *http.Client
	Debug      bool

	// Auth state
	realm     string
	nonce     string
	qop       string
	opaque    string
	algorithm string
}

// NewClient creates a new TR-064 client
func NewClient(baseURL, username, password string, debug bool) *Client {
	return &Client{
		BaseURL:  baseURL,
		Username: username,
		Password: password,
		Debug:    debug,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// debugf logs a debug message if debug mode is enabled
func (c *Client) debugf(format string, args ...interface{}) {
	if c.Debug {
		log.Printf("DEBUG: "+format, args...)
	}
}

// Call executes a TR-064 action with the given arguments
func (c *Client) Call(ctx context.Context, svc *ServiceSpec, action *ActionSpec, in map[string]string) (map[string]string, error) {
	return c.callWithRetry(ctx, svc, action, in, false)
}

// callWithRetry is the internal implementation that tracks auth retries
func (c *Client) callWithRetry(ctx context.Context, svc *ServiceSpec, action *ActionSpec, in map[string]string, isRetry bool) (map[string]string, error) {
	// Check context deadline
	if deadline, ok := ctx.Deadline(); ok {
		c.debugf("Context deadline: %v (in %v)", deadline, time.Until(deadline))
	} else {
		c.debugf("Context has no deadline")
	}

	// Build SOAP request
	soapReq := c.buildSOAPRequest(action, in)

	// Prepare HTTP request
	url := c.BaseURL + svc.ControlURL
	c.debugf("Calling %s#%s at %s", svc.ServiceType, action.Name, url)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader([]byte(soapReq)))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "text/xml; charset=\"utf-8\"")
	req.Header.Set("SOAPAction", fmt.Sprintf("%s#%s", svc.ServiceType, action.Name))

	// Add auth header if we have auth state
	if c.nonce != "" {
		authHeader := c.buildAuthHeader("POST", svc.ControlURL)
		req.Header.Set("Authorization", authHeader)
		c.debugf("Using auth header")
	} else {
		c.debugf("No auth header (expecting 401)")
	}

	// Execute request
	c.debugf("Sending HTTP request...")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.debugf("HTTP request failed: %v", err)
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()
	c.debugf("Received response with status %d", resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	// Handle auth challenge
	if resp.StatusCode == http.StatusUnauthorized {
		// If we already retried with auth and still got 401, credentials are wrong
		if isRetry {
			return nil, fmt.Errorf("authentication failed: got 401 after retry for user %q at %s (response body: %s)", c.Username, url, string(body))
		}

		c.debugf("Got 401, handling auth challenge...")
		if err := c.handleAuthChallenge(resp); err != nil {
			return nil, fmt.Errorf("handling auth challenge: %w", err)
		}
		c.debugf("Auth prepared, retrying request...")

		// Retry with auth (only once)
		return c.callWithRetry(ctx, svc, action, in, true)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse SOAP response
	out, err := c.parseSOAPResponse(action, body)
	if err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return out, nil
}

// buildSOAPRequest constructs a SOAP XML request
func (c *Client) buildSOAPRequest(action *ActionSpec, in map[string]string) string {
	var buf strings.Builder

	buf.WriteString(`<?xml version="1.0" encoding="utf-8"?>`)
	buf.WriteString(`<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">`)
	buf.WriteString(`<s:Body>`)

	// Extract service name from service type (e.g., DeviceInfo from urn:dslforum-org:service:DeviceInfo:1)
	serviceName := extractServiceName(action.ServiceType)

	buf.WriteString(fmt.Sprintf(`<u:%s xmlns:u="%s">`, action.Name, action.ServiceType))

	// Add arguments in order (critical for FRITZ!Box)
	for _, arg := range action.In {
		value := in[arg.Name]
		buf.WriteString(fmt.Sprintf("<%s>%s</%s>", arg.Name, xmlEscape(value), arg.Name))
	}

	buf.WriteString(fmt.Sprintf(`</u:%s>`, action.Name))
	buf.WriteString(`</s:Body>`)
	buf.WriteString(`</s:Envelope>`)

	_ = serviceName // used for debugging if needed

	return buf.String()
}

// parseSOAPResponse extracts output arguments from SOAP XML response
func (c *Client) parseSOAPResponse(action *ActionSpec, body []byte) (map[string]string, error) {
	// Simple XML parsing - look for response element
	responseName := action.Name + "Response"

	// Parse the XML structure
	type envelope struct {
		XMLName xml.Name `xml:"Envelope"`
		Body    struct {
			XMLName  xml.Name
			Response []byte `xml:",innerxml"`
		} `xml:"Body"`
	}

	var env envelope
	if err := xml.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("unmarshaling envelope: %w", err)
	}

	// Now parse the inner response to extract all output arguments
	// We need to dynamically extract all child elements
	result := make(map[string]string)

	// Find the response element
	decoder := xml.NewDecoder(bytes.NewReader(env.Body.Response))
	inResponse := false

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parsing response XML: %w", err)
		}

		switch t := token.(type) {
		case xml.StartElement:
			if strings.HasSuffix(t.Name.Local, "Response") {
				inResponse = true
			} else if inResponse {
				// This is an output argument
				argName := t.Name.Local
				var value string
				if err := decoder.DecodeElement(&value, &t); err != nil {
					return nil, fmt.Errorf("decoding element %s: %w", argName, err)
				}
				result[argName] = value
			}
		case xml.EndElement:
			if strings.HasSuffix(t.Name.Local, "Response") {
				inResponse = false
			}
		}
	}

	_ = responseName // used for validation if needed

	return result, nil
}

// handleAuthChallenge processes the WWW-Authenticate challenge from FRITZ!Box
func (c *Client) handleAuthChallenge(resp *http.Response) error {
	authHeader := resp.Header.Get("WWW-Authenticate")
	if authHeader == "" {
		return fmt.Errorf("no WWW-Authenticate header")
	}
	c.debugf("WWW-Authenticate: %s", authHeader)

	// Parse Digest auth challenge
	// Format: Digest realm="...", nonce="..."
	// Use SplitN to only split on the first space (realm might contain spaces like "HTTPS Access")
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) < 2 || parts[0] != "Digest" {
		return fmt.Errorf("unsupported auth type: %s", parts[0])
	}

	// Parse key-value pairs
	params := make(map[string]string)
	for _, part := range strings.Split(parts[1], ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) == 2 {
			key := kv[0]
			value := strings.Trim(kv[1], `"`)
			params[key] = value
			c.debugf("Parsed auth param: %s = %s", key, value)
		}
	}

	// Store auth parameters
	c.realm = params["realm"]
	c.nonce = params["nonce"]
	c.qop = params["qop"]
	c.opaque = params["opaque"]
	c.algorithm = params["algorithm"]

	c.debugf("Realm: '%s', Nonce: '%s', QOP: '%s'", c.realm, c.nonce, c.qop)

	if c.nonce == "" {
		return fmt.Errorf("no nonce in WWW-Authenticate header")
	}

	return nil
}

// buildAuthHeader constructs the Authorization header for Digest auth
func (c *Client) buildAuthHeader(method, uri string) string {
	// Generate client nonce
	cnonce := generateCNonce()
	nc := "00000001"

	// Compute HA1 = MD5(username:realm:password)
	ha1Input := fmt.Sprintf("%s:%s:%s", c.Username, c.realm, c.Password)
	ha1 := fmt.Sprintf("%x", md5.Sum([]byte(ha1Input)))
	c.debugf("HA1 input: %s:%s:[REDACTED] -> %s", c.Username, c.realm, ha1)

	// Compute HA2 = MD5(method:uri)
	ha2Input := fmt.Sprintf("%s:%s", method, uri)
	ha2 := fmt.Sprintf("%x", md5.Sum([]byte(ha2Input)))
	c.debugf("HA2 input: %s -> %s", ha2Input, ha2)

	// Compute response hash
	var response string
	var responseInput string
	if c.qop == "auth" || c.qop == "auth-int" {
		// RFC 2617: response = MD5(HA1:nonce:nc:cnonce:qop:HA2)
		responseInput = fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, c.nonce, nc, cnonce, c.qop, ha2)
		response = fmt.Sprintf("%x", md5.Sum([]byte(responseInput)))
		c.debugf("Response (with qop=%s): MD5(%s) = %s", c.qop, responseInput, response)
	} else {
		// Simple digest: response = MD5(HA1:nonce:HA2)
		responseInput = fmt.Sprintf("%s:%s:%s", ha1, c.nonce, ha2)
		response = fmt.Sprintf("%x", md5.Sum([]byte(responseInput)))
		c.debugf("Response (no qop): MD5(%s) = %s", responseInput, response)
	}

	// Build Authorization header
	var auth strings.Builder
	auth.WriteString(`Digest username="`)
	auth.WriteString(c.Username)
	auth.WriteString(`", realm="`)
	auth.WriteString(c.realm)
	auth.WriteString(`", nonce="`)
	auth.WriteString(c.nonce)
	auth.WriteString(`", uri="`)
	auth.WriteString(uri)
	auth.WriteString(`", response="`)
	auth.WriteString(response)
	auth.WriteString(`"`)

	if c.qop != "" {
		auth.WriteString(`, qop=`)
		auth.WriteString(c.qop)
		auth.WriteString(`, nc=`)
		auth.WriteString(nc)
		auth.WriteString(`, cnonce="`)
		auth.WriteString(cnonce)
		auth.WriteString(`"`)
	}

	if c.opaque != "" {
		auth.WriteString(`, opaque="`)
		auth.WriteString(c.opaque)
		auth.WriteString(`"`)
	}

	authHeader := auth.String()
	c.debugf("Authorization header: %s", authHeader)
	return authHeader
}

// generateCNonce generates a random client nonce
func generateCNonce() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err) // crypto/rand.Read should never fail
	}
	return hex.EncodeToString(b)
}

// xmlEscape escapes special characters for XML
func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, `'`, "&#39;")
	return s
}
