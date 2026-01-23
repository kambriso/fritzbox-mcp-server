package fritzbox

import (
	"fmt"
	"strings"
)

// Index provides documentation lookup for TR-064 services and actions
type Index struct {
	// In the future, this could load from parsed PDFs
	// For now, we provide basic stub documentation
}

// NewIndex creates a new documentation index
func NewIndex() *Index {
	return &Index{}
}

// Lookup returns documentation for a given service and action
func (idx *Index) Lookup(serviceType, actionName string) string {
	serviceName := extractServiceName(serviceType)
	return generateDocumentation(serviceName, actionName)
}

// LookupWithArgs returns documentation including argument details
func (idx *Index) LookupWithArgs(serviceType, actionName string, inArgs, outArgs []string) string {
	serviceName := extractServiceName(serviceType)
	return generateDetailedDocumentation(serviceName, actionName, inArgs, outArgs)
}

// generateDocumentation creates human-readable documentation from action names
func generateDocumentation(serviceName, actionName string) string {
	actionLower := strings.ToLower(actionName)
	serviceReadable := humanizeServiceName(serviceName)

	// Handle common action patterns
	if strings.HasPrefix(actionLower, "get") {
		subject := strings.TrimPrefix(actionName, "Get")
		subject = strings.TrimPrefix(subject, "get")
		if subject == "" || subject == "Info" {
			return fmt.Sprintf("Get %s information", serviceReadable)
		}
		return fmt.Sprintf("Get %s from %s", humanizeCamelCase(subject), serviceReadable)
	}

	if strings.HasPrefix(actionLower, "x_avm-de_get") || strings.HasPrefix(actionLower, "x_avm_de_get") {
		subject := strings.TrimPrefix(actionName, "X_AVM-DE_Get")
		subject = strings.TrimPrefix(subject, "X_AVM_DE_Get")
		return fmt.Sprintf("Get %s from %s (AVM extension)", humanizeCamelCase(subject), serviceReadable)
	}

	if strings.HasPrefix(actionLower, "set") {
		subject := strings.TrimPrefix(actionName, "Set")
		subject = strings.TrimPrefix(subject, "set")
		return fmt.Sprintf("Set %s on %s", humanizeCamelCase(subject), serviceReadable)
	}

	// Generic fallback
	return fmt.Sprintf("%s - %s action", serviceReadable, actionName)
}

// generateDetailedDocumentation creates documentation with argument details
func generateDetailedDocumentation(serviceName, actionName string, inArgs, outArgs []string) string {
	base := generateDocumentation(serviceName, actionName)

	var details []string
	if len(inArgs) > 0 {
		details = append(details, fmt.Sprintf("Input: %s", humanizeArgList(inArgs)))
	}
	if len(outArgs) > 0 {
		details = append(details, fmt.Sprintf("Returns: %s", humanizeArgList(outArgs)))
	}

	if len(details) > 0 {
		return fmt.Sprintf("%s. %s", base, strings.Join(details, ". "))
	}
	return base
}

// humanizeServiceName converts service names to readable format
func humanizeServiceName(serviceName string) string {
	// Handle common service name patterns
	replacements := map[string]string{
		"DeviceInfo":                 "device",
		"DeviceConfig":               "device configuration",
		"Hosts":                      "network hosts",
		"WLANConfiguration":          "WiFi",
		"WANIPConnection":            "WAN IP connection",
		"WANPPPConnection":           "WAN PPP connection",
		"WANCommonInterfaceConfig":   "WAN interface",
		"LANHostConfigManagement":    "LAN host configuration",
		"LANEthernetInterfaceConfig": "LAN ethernet interface",
		"Time":                       "system time",
		"UserInterface":              "user interface",
		"Layer3Forwarding":           "port forwarding",
		"X_AVM-DE_Storage":           "storage",
		"X_AVM-DE_Homeauto":          "smart home",
		"X_AVM-DE_OnTel":             "telephony",
		"X_AVM-DE_TAM":               "answering machine",
		"X_AVM-DE_Dect":              "DECT phone",
		"X_VoIP":                     "VoIP",
		"X_AVM-DE_Speedtest":         "speed test",
		"X_AVM-DE_MyFritz":           "MyFRITZ",
	}

	if readable, ok := replacements[serviceName]; ok {
		return readable
	}

	// Fallback: add spaces before capitals
	return humanizeCamelCase(serviceName)
}

// humanizeCamelCase converts CamelCase to "camel case"
func humanizeCamelCase(s string) string {
	if s == "" {
		return ""
	}

	// Handle special prefixes
	s = strings.ReplaceAll(s, "X_AVM-DE_", "")
	s = strings.ReplaceAll(s, "X_AVM_DE_", "")

	// Check for standalone acronyms (all caps)
	standAloneAcronyms := map[string]string{
		"SSID": "ssid",
		"IP":   "ip",
		"MAC":  "mac",
		"WAN":  "wan",
		"LAN":  "lan",
		"DNS":  "dns",
		"DHCP": "dhcp",
		"VPN":  "vpn",
		"USB":  "usb",
		"URL":  "url",
		"UUID": "uuid",
	}
	if replacement, ok := standAloneAcronyms[s]; ok {
		return replacement
	}

	// First, add spaces before capitals
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune(' ')
		}
		result.WriteRune(r)
	}

	spaced := result.String()

	// Then handle common acronyms - replace multi-letter sequences
	// Use patterns that match anywhere in the string
	replacements := []struct{ pattern, replacement string }{
		{" S S I D", " ssid"},
		{" I P ", " ip "},
		{" I P", " ip"}, // Handle end of string
		{"I P ", "ip "}, // Handle start of string
		{" M A C ", " mac "},
		{" M A C", " mac"},
		{"M A C ", "mac "},
		{" W A N", " wan"},
		{" L A N", " lan"},
		{" D N S", " dns"},
		{" D H C P", " dhcp"},
		{" V P N", " vpn"},
		{" U S B", " usb"},
		{" U R L", " url"},
		{" U U I D", " uuid"},
		{" U P n P", " upnp"},
	}

	for _, r := range replacements {
		spaced = strings.ReplaceAll(spaced, r.pattern, r.replacement)
	}

	return strings.ToLower(spaced)
}

// humanizeArgList converts argument names to readable format
func humanizeArgList(args []string) string {
	readable := make([]string, len(args))
	for i, arg := range args {
		// Remove "New" prefix common in TR-064
		clean := strings.TrimPrefix(arg, "New")
		readable[i] = humanizeCamelCase(clean)
	}
	return strings.Join(readable, ", ")
}

// extractServiceName extracts the service name from a service type URN
// e.g., "urn:dslforum-org:service:DeviceInfo:1" -> "DeviceInfo"
func extractServiceName(serviceType string) string {
	parts := strings.Split(serviceType, ":")
	if len(parts) >= 4 {
		return parts[3]
	}
	return serviceType
}
