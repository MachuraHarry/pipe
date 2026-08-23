package selfupdate

import "testing"

func TestParseSemver(t *testing.T) {
	cases := []struct {
		in    string
		want  [3]int
		valid bool
	}{
		{"v1.1.1", [3]int{1, 1, 1}, true},
		{"1.2.3", [3]int{1, 2, 3}, true},
		{" v0.10.20 ", [3]int{0, 10, 20}, true},
		{"v1.1.1-5-gfcadff0", [3]int{1, 1, 1}, true},
		{"v1.1.1-dirty", [3]int{1, 1, 1}, true},
		{"v1.1", [3]int{1, 1, 0}, true},
		{"v1", [3]int{1, 0, 0}, true},
		{"dev", [3]int{}, false},
		{"fcadff0", [3]int{}, false},
		{"", [3]int{}, false},
		{"vx.y.z", [3]int{}, false},
	}
	for _, c := range cases {
		got, ok := parseSemver(c.in)
		if ok != c.valid {
			t.Errorf("parseSemver(%q) valid = %v, want %v", c.in, ok, c.valid)
			continue
		}
		if c.valid && got != c.want {
			t.Errorf("parseSemver(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestUpdateNeeded(t *testing.T) {
	cases := []struct {
		current, latest string
		want            bool
	}{
		{"v1.1.1", "v1.1.1", false},
		{"1.1.1", "v1.1.1", false},
		{"v1.1.1", "v1.1.2", true},
		{"v1.1.1", "v1.2.0", true},
		{"v1.1.1", "v2.0.0", true},
		{"v1.1.2", "v1.1.1", false},
		{"v2.0.0", "v1.9.9", false},
		{"v1.1.1-5-gfcadff0", "v1.1.1", false},
		{"v1.1.1-dirty", "v1.1.1", false},
		{"v1.1.1-5-gfcadff0", "v1.1.2", true},
		{"v1.1.0-3-gabc1234", "v1.1.1", true},
		{"dev", "v1.1.1", true},
		{"fcadff0", "v1.1.1", true},
		{"v1.1.1", "garbage", false},
		{"", "v1.1.1", true},
	}
	for _, c := range cases {
		if got := UpdateNeeded(c.current, c.latest); got != c.want {
			t.Errorf("UpdateNeeded(%q, %q) = %v, want %v", c.current, c.latest, got, c.want)
		}
	}
}

func TestArtifactAndBinaryNames(t *testing.T) {
	cases := []struct {
		goos, goarch, artifact, binary string
		supported                      bool
	}{
		{"linux", "amd64", "pipe-linux-amd64.tar.gz", "pipe", true},
		{"linux", "arm64", "pipe-linux-arm64.tar.gz", "pipe", true},
		{"darwin", "amd64", "pipe-darwin-amd64.tar.gz", "pipe", true},
		{"darwin", "arm64", "pipe-darwin-arm64.tar.gz", "pipe", true},
		{"windows", "amd64", "pipe-windows-amd64.tar.gz", "pipe.exe", true},
		{"windows", "arm64", "pipe-windows-arm64.tar.gz", "pipe.exe", false},
		{"freebsd", "amd64", "pipe-freebsd-amd64.tar.gz", "pipe", false},
	}
	for _, c := range cases {
		if got := ArtifactName(c.goos, c.goarch); got != c.artifact {
			t.Errorf("ArtifactName(%s, %s) = %q, want %q", c.goos, c.goarch, got, c.artifact)
		}
		if got := BinaryName(c.goos); got != c.binary {
			t.Errorf("BinaryName(%s) = %q, want %q", c.goos, got, c.binary)
		}
		if got := SupportedPlatform(c.goos, c.goarch); got != c.supported {
			t.Errorf("SupportedPlatform(%s, %s) = %v, want %v", c.goos, c.goarch, got, c.supported)
		}
	}
}

func TestReleaseURL(t *testing.T) {
	want := "https://github.com/MachuraHarry/pipe/releases/tag/v1.1.1"
	if got := ReleaseURL("v1.1.1"); got != want {
		t.Errorf("ReleaseURL = %q, want %q", got, want)
	}
}
