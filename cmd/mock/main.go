package main

import (
	"bytes"
	"context"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"time"
)

// Minimal descriptor types just for capture; you can replace with internal/spec later.
type rootDevice struct {
	Device device `xml:"device"`
}

type device struct {
	ServiceList []service `xml:"serviceList>service"`
}

type service struct {
	ServiceType string `xml:"serviceType"`
	ControlURL  string `xml:"controlURL"`
	SCPDURL     string `xml:"SCPDURL"`
}

func main() {
	var (
		host   = flag.String("host", "fritz.box", "FRITZ!Box host")
		port   = flag.Int("port", 49000, "FRITZ!Box TR-064 port")
		outDir = flag.String("out", "testdata/tr064", "output directory for captures")
		user   = flag.String("user", "", "username (optional, for later auth)")
		pass   = flag.String("pass", "", "password (optional, for later auth)")
	)
	flag.Parse()

	baseURL := fmt.Sprintf("http://%s:%d", *host, *port)
	log.Printf("capturing from %s into %s", baseURL, *outDir)

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatalf("mkdir out: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// plain http client for descriptor + SCPD (usually no auth required)
	httpClient := &http.Client{Timeout: 10 * time.Second}

	// 1) capture tr64desc.xml
	descPath := filepath.Join(*outDir, "tr64desc.xml")
	descXML, services, err := fetchDescriptor(ctx, httpClient, baseURL)
	if err != nil {
		log.Fatalf("fetch descriptor: %v", err)
	}
	if err := os.WriteFile(descPath, descXML, 0o644); err != nil {
		log.Fatalf("write tr64desc: %v", err)
	}
	log.Printf("wrote %s", descPath)

	// 2) capture SCPD XMLs for each service
	for _, svc := range services {
		url := joinURL(baseURL, svc.SCPDURL)
		name := sanitizeServiceType(svc.ServiceType) + ".scpd.xml"
		path := filepath.Join(*outDir, name)

		data, err := fetch(ctx, httpClient, url)
		if err != nil {
			log.Printf("warn: fetch SCPD %s (%s): %v", svc.ServiceType, url, err)
			continue
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			log.Printf("warn: write SCPD %s: %v", path, err)
			continue
		}
		log.Printf("wrote %s", path)
	}

	// 3) capture one real SOAP request/response for DeviceInfo:GetInfo
	// this uses a capturing transport so you see exactly what is sent/returned.
	if *user == "" || *pass == "" {
		log.Printf("no user/pass given; skipping DeviceInfo.GetInfo capture")
		return
	}
	if err := captureDeviceInfoGetInfo(ctx, baseURL, *user, *pass, *outDir); err != nil {
		log.Printf("warn: capture DeviceInfo.GetInfo failed: %v", err)
	} else {
		log.Printf("captured DeviceInfo.GetInfo SOAP exchange")
	}
}

// fetchDescriptor downloads tr64desc.xml and returns raw XML plus parsed services.
func fetchDescriptor(ctx context.Context, client *http.Client, baseURL string) ([]byte, []service, error) {
	url := baseURL + "/tr64desc.xml"
	data, err := fetch(ctx, client, url)
	if err != nil {
		return nil, nil, err
	}
	var root rootDevice
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, nil, fmt.Errorf("unmarshal tr64desc: %w", err)
	}
	return data, root.Device.ServiceList, nil
}

func fetch(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return nil, fmt.Errorf("GET %s: %s: %s", url, res.Status, string(body))
	}
	return io.ReadAll(res.Body)
}

// joinURL handles relative SCPDURL/controlURL like "/upnp/control/deviceinfo".
func joinURL(base, path string) string {
	if len(path) == 0 {
		return base
	}
	if path[0] == '/' {
		return base + path
	}
	return base + "/" + path
}

func sanitizeServiceType(st string) string {
	// urn:dslforum-org:service:DeviceInfo:1 -> deviceinfo_1
	last := st
	if i := len(st) - 1; i >= 0 {
		// quick and dirty: split by ':' from right
		parts := bytes.Split([]byte(st), []byte(":"))
		if len(parts) >= 2 {
			last = string(parts[len(parts)-2]) + "_" + string(parts[len(parts)-1])
		}
	}
	return lowerAscii(last)
}

func lowerAscii(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c = c + ('a' - 'A')
		}
		if c == '-' || c == '.' {
			c = '_'
		}
		b[i] = c
	}
	return string(b)
}

// captureDeviceInfoGetInfo does one authenticated POST to the DeviceInfo control URL
// and dumps raw HTTP request and response to files.
func captureDeviceInfoGetInfo(ctx context.Context, baseURL, user, pass, outDir string) error {
	// This is intentionally simple and not fully "spec-correct" auth-wise.
	// You can later swap this to use your internal/tr064.Client.
	controlURL := baseURL + "/upnp/control/deviceinfo" // standard on FRITZ!Box

	soapBody := []byte(`<?xml version="1.0" encoding="utf-8"?>
<s:Envelope
  xmlns:s="http://schemas.xmlsoap.org/soap/envelope/"
  s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
  <s:Body>
    <u:GetInfo xmlns:u="urn:dslforum-org:service:DeviceInfo:1">
    </u:GetInfo>
  </s:Body>
</s:Envelope>`)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, controlURL, bytes.NewReader(soapBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/xml; charset=\"utf-8\"")
	req.Header.Set("SOAPAction", `"urn:dslforum-org:service:DeviceInfo:1#GetInfo"`)

	// If you already implemented real auth in internal/tr064, replace this
	// with that client instead of basic auth:
	req.SetBasicAuth(user, pass)

	// capturing transport
	var rt captureRT
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: &rt,
	}

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	_, err = io.ReadAll(res.Body) // force captureRT to see full body
	if err != nil {
		return err
	}

	// write captured request/response
	if err := os.WriteFile(filepath.Join(outDir, "deviceinfo_getinfo_request.http"), rt.reqDump, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "deviceinfo_getinfo_response.http"), rt.resDump, 0o644); err != nil {
		return err
	}
	return nil
}

type captureRT struct {
	reqDump []byte
	resDump []byte
}

func (c *captureRT) RoundTrip(req *http.Request) (*http.Response, error) {
	rd, err := httputil.DumpRequestOut(req, true)
	if err == nil {
		c.reqDump = rd
	}

	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	_ = res.Body.Close()
	res.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	// Fake a dumped response (status line + headers + body)
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "HTTP/%d.%d %d %s\r\n", res.ProtoMajor, res.ProtoMinor, res.StatusCode, http.StatusText(res.StatusCode))
	if err := res.Header.Write(&buf); err == nil {
		buf.WriteString("\r\n")
	}
	buf.Write(bodyBytes)
	c.resDump = buf.Bytes()

	return res, nil
}
