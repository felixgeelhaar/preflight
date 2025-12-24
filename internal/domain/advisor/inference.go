package advisor

import (
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

// Common personal email domains.
var personalEmailDomains = map[string]bool{
	"gmail.com":      true,
	"yahoo.com":      true,
	"hotmail.com":    true,
	"outlook.com":    true,
	"icloud.com":     true,
	"me.com":         true,
	"mac.com":        true,
	"protonmail.com": true,
	"proton.me":      true,
	"fastmail.com":   true,
	"tutanota.com":   true,
	"aol.com":        true,
	"live.com":       true,
	"msn.com":        true,
	"mail.com":       true,
	"zoho.com":       true,
}

// InferredContext contains the results of context inference.
type InferredContext struct {
	WorkContext    WorkContext
	DeviceType     DeviceType
	EmailDomains   []string
	WorkDomains    []string
	PersonalEmails []string
	Confidence     float64  // 0.0-1.0 confidence in the inference
	Signals        []string // Human-readable signals that led to the inference
}

// InferWorkContext analyzes email addresses to determine work/personal context.
func InferWorkContext(emails []string) InferredContext {
	ctx := InferredContext{
		WorkContext:  WorkContextUnknown,
		EmailDomains: make([]string, 0),
		WorkDomains:  make([]string, 0),
		Signals:      make([]string, 0),
	}

	if len(emails) == 0 {
		return ctx
	}

	var workEmails, personalEmails int

	for _, email := range emails {
		domain := extractDomain(email)
		if domain == "" {
			continue
		}

		ctx.EmailDomains = append(ctx.EmailDomains, domain)

		if personalEmailDomains[domain] {
			personalEmails++
			ctx.PersonalEmails = append(ctx.PersonalEmails, email)
			ctx.Signals = append(ctx.Signals, "Personal email domain: "+domain)
		} else {
			workEmails++
			ctx.WorkDomains = append(ctx.WorkDomains, domain)
			ctx.Signals = append(ctx.Signals, "Corporate email domain: "+domain)
		}
	}

	// Determine work context based on email distribution
	total := workEmails + personalEmails
	if total == 0 {
		return ctx
	}

	workRatio := float64(workEmails) / float64(total)

	switch {
	case workRatio == 1.0:
		ctx.WorkContext = WorkContextWork
		ctx.Confidence = 0.9
	case workRatio == 0.0:
		ctx.WorkContext = WorkContextPersonal
		ctx.Confidence = 0.9
	case workRatio >= 0.5:
		ctx.WorkContext = WorkContextMixed
		ctx.Confidence = 0.7
	default:
		ctx.WorkContext = WorkContextMixed
		ctx.Confidence = 0.6
	}

	return ctx
}

// InferDeviceType detects whether the machine is a laptop or desktop.
func InferDeviceType() DeviceType {
	switch runtime.GOOS {
	case "darwin":
		return inferDeviceTypeMacOS()
	case "linux":
		return inferDeviceTypeLinux()
	default:
		return DeviceTypeUnknown
	}
}

// inferDeviceTypeMacOS uses sysctl to detect device type on macOS.
func inferDeviceTypeMacOS() DeviceType {
	// Check hardware model
	out, err := exec.Command("sysctl", "-n", "hw.model").Output()
	if err != nil {
		return DeviceTypeUnknown
	}

	model := strings.ToLower(strings.TrimSpace(string(out)))

	// MacBook models are laptops
	if strings.Contains(model, "macbook") {
		return DeviceTypeLaptop
	}

	// Mac Mini, Mac Pro, iMac, Mac Studio are desktops
	if strings.Contains(model, "macmini") ||
		strings.Contains(model, "macpro") ||
		strings.Contains(model, "imac") ||
		strings.Contains(model, "mac14") || // Mac Studio
		strings.Contains(model, "mac15") {
		return DeviceTypeDesktop
	}

	// Check for battery as fallback (laptops have batteries)
	batteryOut, err := exec.Command("pmset", "-g", "batt").Output()
	if err == nil && strings.Contains(string(batteryOut), "Battery") {
		return DeviceTypeLaptop
	}

	return DeviceTypeUnknown
}

// inferDeviceTypeLinux checks for battery presence on Linux.
func inferDeviceTypeLinux() DeviceType {
	// Check if /sys/class/power_supply/BAT* exists
	out, err := exec.Command("ls", "/sys/class/power_supply/").Output()
	if err != nil {
		return DeviceTypeUnknown
	}

	if strings.Contains(string(out), "BAT") {
		return DeviceTypeLaptop
	}

	return DeviceTypeDesktop
}

// InferWorkContextFromSSHKeys analyzes SSH key names for work/personal patterns.
func InferWorkContextFromSSHKeys(keyNames []string) WorkContext {
	var workKeys, personalKeys int

	workPatterns := []string{"work", "corp", "company", "office", "enterprise", "business"}
	personalPatterns := []string{"personal", "home", "private", "github", "gitlab"}

	for _, name := range keyNames {
		lower := strings.ToLower(name)

		for _, pattern := range workPatterns {
			if strings.Contains(lower, pattern) {
				workKeys++
				break
			}
		}

		for _, pattern := range personalPatterns {
			if strings.Contains(lower, pattern) {
				personalKeys++
				break
			}
		}
	}

	if workKeys > 0 && personalKeys > 0 {
		return WorkContextMixed
	}
	if workKeys > 0 {
		return WorkContextWork
	}
	if personalKeys > 0 {
		return WorkContextPersonal
	}

	return WorkContextUnknown
}

// InferWorkContextFromTools analyzes installed tools for work indicators.
func InferWorkContextFromTools(tools []string) (WorkContext, []string) {
	signals := make([]string, 0)

	workTools := map[string]string{
		"slack":            "Team communication tool",
		"teams":            "Microsoft Teams",
		"zoom":             "Video conferencing",
		"webex":            "Cisco WebEx",
		"okta":             "Enterprise SSO",
		"1password":        "Password manager (often corporate)",
		"jamf":             "MDM tool",
		"kandji":           "MDM tool",
		"openvpn":          "VPN client",
		"tunnelblick":      "VPN client",
		"cisco-anyconnect": "Corporate VPN",
		"zscaler":          "Enterprise security",
	}

	personalTools := map[string]string{
		"steam":   "Gaming platform",
		"discord": "Gaming/community chat",
		"spotify": "Music streaming",
		"vlc":     "Media player",
	}

	var workCount, personalCount int

	for _, tool := range tools {
		lower := strings.ToLower(tool)

		if desc, ok := workTools[lower]; ok {
			workCount++
			signals = append(signals, "Work tool detected: "+desc)
		}

		if desc, ok := personalTools[lower]; ok {
			personalCount++
			signals = append(signals, "Personal tool detected: "+desc)
		}
	}

	if workCount > 0 && personalCount > 0 {
		return WorkContextMixed, signals
	}
	if workCount > 0 {
		return WorkContextWork, signals
	}
	if personalCount > 0 {
		return WorkContextPersonal, signals
	}

	return WorkContextUnknown, signals
}

// SuggestLayers returns suggested layer names based on inferred context.
func SuggestLayers(ctx InferredContext, deviceType DeviceType) []string {
	layers := []string{"base"}

	// Add identity layer based on work context
	switch ctx.WorkContext {
	case WorkContextWork:
		layers = append(layers, "identity.work")
	case WorkContextPersonal:
		layers = append(layers, "identity.personal")
	case WorkContextMixed:
		layers = append(layers, "identity.work", "identity.personal")
	case WorkContextUnknown:
		// No identity layer suggested when context is unknown
	}

	// Add device layer based on device type
	switch deviceType {
	case DeviceTypeLaptop:
		layers = append(layers, "device.laptop")
	case DeviceTypeDesktop:
		layers = append(layers, "device.desktop")
	case DeviceTypeUnknown:
		// No device layer suggested when type is unknown
	}

	return layers
}

// extractDomain extracts the domain from an email address.
func extractDomain(email string) string {
	// Simple regex to extract domain
	re := regexp.MustCompile(`@([a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`)
	matches := re.FindStringSubmatch(email)
	if len(matches) >= 2 {
		return strings.ToLower(matches[1])
	}
	return ""
}
