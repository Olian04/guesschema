package hints

import "strings"

func keyFor(propKey string) string {
	k := normalizeKey(propKey)
	if k == "" {
		return ""
	}
	switch {
	case keyIsOneOf(k, "email", "e_mail", "mail", "user_email", "email_address"):
		return "email"
	case keyIsOneOf(k, "url", "uri", "href", "link", "website", "homepage"):
		return "uri"
	case keyIsOneOf(k, "uuid", "guid"):
		return "uuid"
	case keyIsOneOf(k, "hostname", "host", "domain"):
		return "hostname"
	case keyIsOneOf(k, "ipv4", "ip4"):
		return "ipv4"
	case keyIsOneOf(k, "ipv6", "ip6"):
		return "ipv6"
	case keyIsOneOf(k, "ip", "ip_address"):
		return "ipv4"
	case keyIsOneOf(k, "msisdn", "mobile", "mobile_number"):
		return "msisdn"
	case keyIsOneOf(k, "imsi"):
		return "imsi"
	case keyIsOneOf(k, "iccid"):
		return "iccid"
	case keyIsOneOf(k, "phone", "telephone", "tel", "phone_number"):
		return "phone"
	case keyIsOneOf(k, "latitude", "lat"):
		return "latitude"
	case keyIsOneOf(k, "longitude", "lon", "lng"):
		return "longitude"
	case keyIsOneOf(k, "coordinates", "coords", "geo", "location"):
		return "geo-coordinates"
	case keyHasAnySuffix(k, "_at", "_time", "_timestamp") || keyIsOneOf(k, "timestamp", "created", "updated", "datetime", "date_time"):
		return "date-time"
	case keyIsOneOf(k, "date", "birthdate", "birth_date", "dob"):
		return "date"
	default:
		return ""
	}
}

func normalizeKey(propKey string) string {
	return strings.ToLower(strings.TrimSpace(propKey))
}

func keyIsOneOf(k string, names ...string) bool {
	for _, n := range names {
		if k == n {
			return true
		}
	}
	return false
}

func keyHasAnySuffix(k string, suffixes ...string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(k, suffix) {
			return true
		}
	}
	return false
}
