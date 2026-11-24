package requests

import (
	"strings"
	"unicode/utf8"
)

type ReadinessClass uint8

const (
	Required ReadinessClass = iota
	BoundedMedia
	Background
)

// ClassifyReadiness is shared by render activity and request readiness. Unknown
// resource types are required so a new Chrome type does not silently disappear.
func ClassifyReadiness(resourceType string) ReadinessClass {
	switch resourceType {
	case "Media":
		return BoundedMedia
	case "Ping", "EventSource", "WebSocket", "CSPViolationReport", "Preflight",
		"XHR", "Fetch", "Manifest", "SignedExchange", "FedCM", "Other", "Prefetch":
		return Background
	default:
		return Required
	}
}

// BoundedURLForLog truncates diagnostic URLs and returns their original byte length.
func BoundedURLForLog(rawURL string) (string, int) {
	originalLength := len(rawURL)
	if strings.EqualFold(rawURL[:min(5, len(rawURL))], "data:") {
		return "data:[omitted]", originalLength
	}
	if originalLength <= 512 {
		return rawURL, originalLength
	}
	const ellipsis = "…"
	end := 512 - len(ellipsis)
	for end > 0 && !utf8.RuneStart(rawURL[end]) {
		end--
	}
	return rawURL[:end] + ellipsis, originalLength
}
