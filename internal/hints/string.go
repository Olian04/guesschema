package hints

import (
	"net"
	"regexp"
	"strings"
	"unicode"
)

// Standard names follow JSON Schema 2020-12 validation (email, uri, date-time, …).
// Telecom identifiers (msisdn, imsi, iccid) use common industry shapes (E.164, 3GPP, E.118).
var (
	reDateTime = regexp.MustCompile(
		`^\d{4}-\d{2}-\d{2}[Tt]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?$`,
	)
	reDate       = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	reTime       = regexp.MustCompile(`^\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?$`)
	reUUID       = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	reEmail      = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	reURI        = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*://\S+$`)
	reURIRef     = regexp.MustCompile(`^(?:\.{1,2}/|/|#)[^\s]*$`)
	reICCID      = regexp.MustCompile(`^(?:89)?[0-9]{18,20}$`)
	reIMSI       = regexp.MustCompile(`^[0-9]{14,15}$`)
	reMSISDN     = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)
	reCoords     = regexp.MustCompile(`^[-+]?\d{1,2}(?:\.\d+)?\s*,\s*[-+]?\d{1,3}(?:\.\d+)?$`)
	reHostname   = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+$`)
	rePhoneLoose = regexp.MustCompile(`^\+?[\d\s().-]{10,}$`)
)

func stringFor(propKey, s string) string {
	if h := stringValueFor(s); h != "" {
		return h
	}
	return keyFor(propKey)
}

func stringValueFor(s string) string {
	if s == "" {
		return ""
	}
	switch {
	case reUUID.MatchString(s):
		return "uuid"
	case parseIPv4(s):
		return "ipv4"
	case parseIPv6(s):
		return "ipv6"
	case reDateTime.MatchString(s):
		return "date-time"
	case reDate.MatchString(s):
		return "date"
	case reTime.MatchString(s):
		return "time"
	case reEmail.MatchString(s) && !hasNonASCII(s):
		return "email"
	case reEmail.MatchString(s) && hasNonASCII(s):
		return "idn-email"
	case reURI.MatchString(s):
		return "uri"
	case reURIRef.MatchString(s):
		return "uri-reference"
	case reICCID.MatchString(s):
		return "iccid"
	case reIMSI.MatchString(s):
		return "imsi"
	case reMSISDN.MatchString(s):
		return "msisdn"
	case looksLikePhone(s):
		return "phone"
	case reCoords.MatchString(s):
		return "geo-coordinates"
	case reHostname.MatchString(s) && !strings.Contains(s, "@") && !strings.Contains(s, "://"):
		return "hostname"
	default:
		return ""
	}
}

func parseIPv4(s string) bool {
	ip := net.ParseIP(s)
	return ip != nil && ip.To4() != nil && strings.Contains(s, ".")
}

func parseIPv6(s string) bool {
	if !strings.Contains(s, ":") {
		return false
	}
	ip := net.ParseIP(s)
	return ip != nil && ip.To4() == nil
}

func hasNonASCII(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return true
		}
	}
	return false
}

func looksLikePhone(s string) bool {
	if !rePhoneLoose.MatchString(s) {
		return false
	}
	digits := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			digits++
		}
	}
	return digits >= 8 && digits <= 15
}
