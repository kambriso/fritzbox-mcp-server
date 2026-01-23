package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// argSpec represents a TR-064 action argument
type argSpec struct {
	Name            string   // e.g. NewModelName
	Direction       string   // "in" or "out"
	RelatedStateVar string   // reference to state variable
	AllowedValues   []string // from allowedValueList
	DataType        string   // string, ui4, boolean, etc.
}

// actionSpec represents a TR-064 action
type actionSpec struct {
	ServiceType string    // urn:dslforum-org:service:DeviceInfo:1
	Name        string    // GetInfo
	In          []argSpec // input arguments in order
	Out         []argSpec // output arguments in order
}

// serviceSpec represents a TR-064 service
type serviceSpec struct {
	ServiceType string                 // urn:dslforum-org:service:DeviceInfo:1
	ControlURL  string                 // /upnp/control/deviceinfo
	SCPDURL     string                 // /deviceinfoSCPD.xml
	Actions     map[string]*actionSpec // action name -> spec
}

// registry holds all discovered TR-064 services and actions
type registry struct {
	Services map[string]*serviceSpec // service type -> spec
}

// XML structures for parsing tr64desc.xml

type tr64Device struct {
	XMLName     xml.Name     `xml:"device"`
	ServiceList *serviceList `xml:"serviceList"`
	DeviceList  *deviceList  `xml:"deviceList"`
}

type deviceList struct {
	Devices []tr64Device `xml:"device"`
}

type serviceList struct {
	Services []service `xml:"service"`
}

type service struct {
	ServiceType string `xml:"serviceType"`
	ServiceID   string `xml:"serviceId"`
	ControlURL  string `xml:"controlURL"`
	EventSubURL string `xml:"eventSubURL"`
	SCPDURL     string `xml:"SCPDURL"`
}

type tr64Root struct {
	XMLName xml.Name   `xml:"root"`
	Device  tr64Device `xml:"device"`
}

// XML structures for parsing SCPD files

type scpdRoot struct {
	XMLName           xml.Name          `xml:"scpd"`
	ActionList        actionList        `xml:"actionList"`
	ServiceStateTable serviceStateTable `xml:"serviceStateTable"`
}

type actionList struct {
	Actions []scpdAction `xml:"action"`
}

type scpdAction struct {
	Name         string       `xml:"name"`
	ArgumentList argumentList `xml:"argumentList"`
}

type argumentList struct {
	Arguments []argument `xml:"argument"`
}

type argument struct {
	Name                 string `xml:"name"`
	Direction            string `xml:"direction"`
	RelatedStateVariable string `xml:"relatedStateVariable"`
}

type serviceStateTable struct {
	StateVariables []stateVariable `xml:"stateVariable"`
}

type stateVariable struct {
	Name          string        `xml:"name"`
	DataType      string        `xml:"dataType"`
	AllowedValues allowedValues `xml:"allowedValueList"`
}

type allowedValues struct {
	Values []string `xml:"allowedValue"`
}

// fetchAllXML downloads all XML descriptors from FRITZ!Box (no auth required)
// and saves them to the specified directory
func fetchAllXML(ctx context.Context, baseURL, outputDir string) error {
	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Step 1: Fetch tr64desc.xml
	descURL := baseURL + "/tr64desc.xml"
	descBody, err := fetchXML(ctx, descURL)
	if err != nil {
		return fmt.Errorf("fetching tr64desc.xml: %w", err)
	}

	// Save tr64desc.xml
	descPath := filepath.Join(outputDir, "tr64desc.xml")
	if err := os.WriteFile(descPath, descBody, 0644); err != nil {
		return fmt.Errorf("writing tr64desc.xml: %w", err)
	}

	// Parse to get SCPD URLs
	var root tr64Root
	if err := xml.Unmarshal(descBody, &root); err != nil {
		return fmt.Errorf("parsing tr64desc.xml: %w", err)
	}

	// Step 2: Recursively collect all services from root and nested devices
	allServices := collectServicesRecursive(&root.Device)

	// Step 3: Fetch all SCPD files
	for _, svc := range allServices {
		scpdURL := baseURL + svc.SCPDURL
		scpdBody, err := fetchXML(ctx, scpdURL)
		if err != nil {
			// Log but don't fail - some services might not be available
			fmt.Fprintf(os.Stderr, "Warning: failed to fetch %s: %v\n", scpdURL, err)
			continue
		}

		// Save SCPD file (sanitize filename)
		filename := sanitizeFilename(svc.SCPDURL)
		scpdPath := filepath.Join(outputDir, filename)
		if err := os.WriteFile(scpdPath, scpdBody, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", filename, err)
		}
	}

	return nil
}

// fetchXML fetches XML content from a URL (no authentication)
func fetchXML(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	return body, nil
}

// sanitizeFilename converts a URL path to a safe filename
func sanitizeFilename(path string) string {
	// Remove leading slash
	name := strings.TrimPrefix(path, "/")
	// Replace remaining slashes with underscores
	name = strings.ReplaceAll(name, "/", "_")
	// Ensure .xml extension
	if !strings.HasSuffix(name, ".xml") {
		name += ".xml"
	}
	return name
}

// loadFromFritz discovers all TR-064 services from the FRITZ!Box
func loadFromFritz(ctx context.Context, baseURL string) (*registry, error) {
	registry := &registry{
		Services: make(map[string]*serviceSpec),
	}

	// Step 1: Fetch and parse tr64desc.xml
	descURL := baseURL + "/tr64desc.xml"
	req, err := http.NewRequestWithContext(ctx, "GET", descURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching tr64desc.xml: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tr64desc.xml returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading tr64desc.xml: %w", err)
	}

	var root tr64Root
	if err := xml.Unmarshal(body, &root); err != nil {
		return nil, fmt.Errorf("parsing tr64desc.xml: %w", err)
	}

	// Step 2: Recursively collect all services from root device and nested devices
	allServices := collectServicesRecursive(&root.Device)

	// Step 3: For each service, fetch and parse its SCPD
	for _, svc := range allServices {
		serviceSpec := &serviceSpec{
			ServiceType: svc.ServiceType,
			ControlURL:  svc.ControlURL,
			SCPDURL:     svc.SCPDURL,
			Actions:     make(map[string]*actionSpec),
		}

		// Fetch SCPD
		scpdURL := baseURL + svc.SCPDURL
		if err := loadSCPD(ctx, scpdURL, serviceSpec); err != nil {
			// Log but don't fail - some services might not be available
			fmt.Fprintf(io.Discard, "Warning: failed to load SCPD for %s: %v\n", svc.ServiceType, err)
			continue
		}

		registry.Services[svc.ServiceType] = serviceSpec
	}

	return registry, nil
}

// collectServicesRecursive recursively collects all services from a device and its nested devices
func collectServicesRecursive(device *tr64Device) []service {
	var services []service

	// Collect services from this device
	if device.ServiceList != nil {
		services = append(services, device.ServiceList.Services...)
	}

	// Recursively collect services from nested devices
	if device.DeviceList != nil {
		for i := range device.DeviceList.Devices {
			nestedServices := collectServicesRecursive(&device.DeviceList.Devices[i])
			services = append(services, nestedServices...)
		}
	}

	return services
}

// loadSCPD fetches and parses a service's SCPD XML
func loadSCPD(ctx context.Context, scpdURL string, serviceSpec *serviceSpec) error {
	req, err := http.NewRequestWithContext(ctx, "GET", scpdURL, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching SCPD: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SCPD returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading SCPD: %w", err)
	}

	var scpd scpdRoot
	if err := xml.Unmarshal(body, &scpd); err != nil {
		return fmt.Errorf("parsing SCPD: %w", err)
	}

	// Build state variable lookup for data types and allowed values
	stateVars := make(map[string]*stateVariable)
	for i := range scpd.ServiceStateTable.StateVariables {
		sv := &scpd.ServiceStateTable.StateVariables[i]
		stateVars[sv.Name] = sv
	}

	// Process each action
	for _, action := range scpd.ActionList.Actions {
		actionSpec := &actionSpec{
			ServiceType: serviceSpec.ServiceType,
			Name:        action.Name,
			In:          []argSpec{},
			Out:         []argSpec{},
		}

		// Process arguments in order (important!)
		for _, arg := range action.ArgumentList.Arguments {
			argSpec := argSpec{
				Name:            arg.Name,
				Direction:       arg.Direction,
				RelatedStateVar: arg.RelatedStateVariable,
			}

			// Look up data type and allowed values from state variable
			if sv, ok := stateVars[arg.RelatedStateVariable]; ok {
				argSpec.DataType = sv.DataType
				argSpec.AllowedValues = sv.AllowedValues.Values
			}

			// Add to in or out list (preserving order)
			if strings.ToLower(arg.Direction) == "in" {
				actionSpec.In = append(actionSpec.In, argSpec)
			} else {
				actionSpec.Out = append(actionSpec.Out, argSpec)
			}
		}

		serviceSpec.Actions[action.Name] = actionSpec
	}

	return nil
}

// getAction looks up an action by service type and action name
func (r *registry) getAction(serviceType, actionName string) (*actionSpec, error) {
	svc, ok := r.Services[serviceType]
	if !ok {
		return nil, fmt.Errorf("service not found: %s", serviceType)
	}

	action, ok := svc.Actions[actionName]
	if !ok {
		return nil, fmt.Errorf("action not found: %s in service %s", actionName, serviceType)
	}

	return action, nil
}
