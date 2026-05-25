package hints

import "math"

func numberFor(propKey string, n float64) string {
	if math.IsInf(n, 0) || math.IsNaN(n) {
		return ""
	}
	k := normalizeKey(propKey)
	switch {
	case keyIsOneOf(k, "latitude", "lat"):
		if n >= -90 && n <= 90 {
			return "latitude"
		}
	case keyIsOneOf(k, "longitude", "lon", "lng"):
		if n >= -180 && n <= 180 {
			return "longitude"
		}
	case keyHasAnySuffix(k, "_at", "_time", "_timestamp") || keyIsOneOf(k, "timestamp", "created", "updated"):
		if looksUnixSeconds(n) || looksUnixMillis(n) {
			return "unix-time"
		}
	}
	if h := keyFor(propKey); h != "" && h != "date-time" {
		return h
	}
	return ""
}

func looksUnixSeconds(n float64) bool {
	if n != math.Trunc(n) {
		return false
	}
	return n >= 978_307_200 && n <= 9_999_999_999
}

func looksUnixMillis(n float64) bool {
	if n != math.Trunc(n) {
		return false
	}
	return n >= 978_307_200_000 && n <= 9_999_999_999_999
}
