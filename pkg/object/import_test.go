package object

import "testing"

func TestParseModuleSpec(t *testing.T) {
	tests := []struct {
		input      string
		expectName string
		expectVer  string
	}{
		{"log-analyzer", "log-analyzer", ""},
		{"log-analyzer@1.0.0", "log-analyzer", "1.0.0"},
		{"log-analyzer@0.9.0", "log-analyzer", "0.9.0"},
		{"mylib", "mylib", ""},
		{"mylib@2", "mylib", "2"},
		{"./mylib.pipe", "./mylib.pipe", ""},
		{"../module/lib", "../module/lib", ""},
	}

	for _, tt := range tests {
		name, ver := parseModuleSpec(tt.input)
		if name != tt.expectName || ver != tt.expectVer {
			t.Errorf("parseModuleSpec(%q) = (%q, %q), want (%q, %q)",
				tt.input, name, ver, tt.expectName, tt.expectVer)
		}
	}
}

func TestParseModuleSpecEdgeCases(t *testing.T) {
	// @ at the start should be ignored
	name, ver := parseModuleSpec("@latest")
	if name != "@latest" || ver != "" {
		t.Errorf("parseModuleSpec(@latest) = (%q, %q), want (@latest, '')", name, ver)
	}
}
