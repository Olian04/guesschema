package hints

func boolFor(propKey string) string {
	k := normalizeKey(propKey)
	switch {
	case keyIsOneOf(k, "active", "enabled", "is_active", "is_enabled"):
		return "flag"
	default:
		return ""
	}
}
