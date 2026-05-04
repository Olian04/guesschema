package schemaguess

import (
	"strings"
)

// JoinPointer returns RFC 6901 reference-token path: parent + "/" + encoded child key.
// parent is "" for document root; child is an object property name (not yet encoded).
func JoinPointer(parent, child string) string {
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

// DecodeReferenceToken reverses encodeReferenceToken.
func DecodeReferenceToken(s string) string {
	s = strings.ReplaceAll(s, "~1", "/")
	s = strings.ReplaceAll(s, "~0", "~")
	return s
}

// SplitPointer splits an RFC 6901 pointer into reference tokens (no leading empty).
// "" -> nil; "/a/b" -> ["a","b"].
func SplitPointer(ptr string) []string {
	if ptr == "" {
		return nil
	}
	if !strings.HasPrefix(ptr, "/") {
		return nil
	}
	parts := strings.Split(ptr[1:], "/")
	for i := range parts {
		parts[i] = DecodeReferenceToken(parts[i])
	}
	return parts
}
