package main

import (
	"reflect"
	"testing"
)

// Issue #888: the CORS settings Compose forwards must have the documented
// effect. Compose forwards an unset host value as the empty string, so empty
// must behave exactly like unset: deny all cross-origin requests.
func TestCORSConfigFromEnv(t *testing.T) {
	cases := []struct {
		name     string
		origins  string
		wildcard string
		want     []string
		unsafe   bool
	}{
		{name: "empty values keep deny-all", origins: "", wildcard: ""},
		{name: "comma separated origins are trimmed", origins: " https://a.example , https://b.example ,", want: []string{"https://a.example", "https://b.example"}},
		{name: "wildcard needs exact true", origins: "https://a.example", wildcard: "TRUE", want: []string{"https://a.example"}},
		{name: "wildcard true", wildcard: "true", unsafe: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("NRCC_CORS_ORIGINS", tc.origins)
			t.Setenv("NRCC_CORS_UNSAFE_WILDCARD", tc.wildcard)
			got := corsConfigFromEnv()
			if !reflect.DeepEqual(got.AllowedOrigins, tc.want) {
				t.Errorf("AllowedOrigins = %#v, want %#v", got.AllowedOrigins, tc.want)
			}
			if got.AllowUnsafeWildcard != tc.unsafe {
				t.Errorf("AllowUnsafeWildcard = %v, want %v", got.AllowUnsafeWildcard, tc.unsafe)
			}
		})
	}
}
