package module

import (
	"testing"

	"github.com/MachuraHarry/pipe/pkg/object"
)

func TestVersionLTE(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"1.2.3", "1.2.3", true},
		{"1.2.3", "1.2.4", true},
		{"1.2.3", "1.3.0", true},
		{"1.2.3", "2.0.0", true},
		{"2.0.0", "1.2.3", false},
		{"1.2.3", "1.2.2", false},
		{"1.2", "1.2.0", true},
		{"1", "1.0.0", true},
		{"v1.2.3", "1.2.4", true},
		{"1.2.3-alpha", "1.2.3", true},
		{"1.2.3", "1.2.3-alpha", false},
		{"1.2.3-alpha.1", "1.2.3-alpha.2", true},
		{"1.2.3-1", "1.2.3-alpha", true},
		{"1.2.3+meta", "1.2.3", true},
	}
	for _, tc := range cases {
		if got := versionLTE(tc.a, tc.b); got != tc.want {
			t.Errorf("versionLTE(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestResolveVersionCaret(t *testing.T) {
	mod := &object.ModuleEntry{
		Latest: "2.0.0",
		Versions: map[string]string{
			"1.0.0": "https://x/1.0.0",
			"1.2.0": "https://x/1.2.0",
			"1.9.9": "https://x/1.9.9",
			"2.0.0": "https://x/2.0.0",
		},
	}
	cases := []struct {
		constraint string
		want       string
	}{
		{"^1.0.0", "1.9.9"},
		{"^1.2.0", "1.9.9"},
		{"^1.9.9", "1.9.9"},
		{"^2.0.0", "2.0.0"},
		{"^3.0.0", "2.0.0"},
		{"1.2.0", "1.2.0"},
		{"latest", "2.0.0"},
		{"*", "2.0.0"},
		{"", "2.0.0"},
		{"9.9.9", "2.0.0"},
	}
	for _, tc := range cases {
		if got := resolveVersion(tc.constraint, mod); got != tc.want {
			t.Errorf("resolveVersion(%q) = %q, want %q", tc.constraint, got, tc.want)
		}
	}
}
