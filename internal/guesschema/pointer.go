package guesschema

import (
	"strings"
)

func joinPointer(parent, child string) string {
	tok := encodeReferenceToken(child)
	if parent == "" {
		return "/" + tok
	}
	return parent + "/" + tok
}

func encodeReferenceToken(s string) string {
	s = strings.ReplaceAll(s, "~", "~0")
	s = strings.ReplaceAll(s, "/", "~1")
	return s
}

func decodeReferenceToken(s string) string {
	s = strings.ReplaceAll(s, "~1", "/")
	s = strings.ReplaceAll(s, "~0", "~")
	return s
}

func splitPointer(ptr string) []string {
	if ptr == "" {
		return nil
	}
	if !strings.HasPrefix(ptr, "/") {
		return nil
	}
	parts := strings.Split(ptr[1:], "/")
	for i := range parts {
		parts[i] = decodeReferenceToken(parts[i])
	}
	return parts
}
