package guesschema_test

import (
	"testing"
	"time"

	"github.com/Olian04/guesschema/pkg/guesschema"
)

func TestBuildSchema_stringFormatHints(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		line   string
		field  string
		format string
	}{
		{name: "date-time", line: `{"t":"2020-01-02T03:04:05Z"}`, field: "t", format: "date-time"},
		{name: "date", line: `{"d":"2020-01-02"}`, field: "d", format: "date"},
		{name: "email", line: `{"e":"user@example.com"}`, field: "e", format: "email"},
		{name: "uri", line: `{"u":"https://example.com/path"}`, field: "u", format: "uri"},
		{name: "uri-reference", line: `{"r":"/foo/bar"}`, field: "r", format: "uri-reference"},
		{name: "uuid", line: `{"id":"f81d4fae-7dec-11d0-a765-00a0c91e6bf6"}`, field: "id", format: "uuid"},
		{name: "ipv4", line: `{"ip":"192.168.0.1"}`, field: "ip", format: "ipv4"},
		{name: "ipv6", line: `{"ip":"2001:db8::1"}`, field: "ip", format: "ipv6"},
		{name: "msisdn", line: `{"m":"+14155552671"}`, field: "m", format: "msisdn"},
		{name: "imsi", line: `{"i":"310150123456789"}`, field: "i", format: "imsi"},
		{name: "iccid", line: `{"c":"89014103211118510720"}`, field: "c", format: "iccid"},
		{name: "phone", line: `{"p":"+1 (415) 555-2671"}`, field: "p", format: "phone"},
		{name: "geo-coordinates", line: `{"g":"37.7749, -122.4194"}`, field: "g", format: "geo-coordinates"},
		{name: "hostname", line: `{"h":"api.example.com"}`, field: "h", format: "hostname"},
		{name: "key-email", line: `{"email":"not-an-email"}`, field: "email", format: "email"},
		{name: "unix-time", line: `{"created_at":1609459200}`, field: "created_at", format: "unix-time"},
		{name: "latitude", line: `{"lat":37.77}`, field: "lat", format: "latitude"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := guesschema.NewAccumulator()
			if err := a.ObserveLine([]byte(tc.line)); err != nil {
				t.Fatal(err)
			}
			s := guesschema.BuildSchema(a, 0.1, time.Unix(0, 0).UTC())
			props := s["properties"].(map[string]any)
			f := props[tc.field].(map[string]any)
			if f["format"] != tc.format {
				t.Fatalf("format: got %#v want %q", f["format"], tc.format)
			}
		})
	}
}
