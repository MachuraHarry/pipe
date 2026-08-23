package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/MachuraHarry/pipe/pkg/build"
)

func makeTarball(t *testing.T, binaryContent string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: BinaryName(runtime.GOOS), Mode: 0755, Size: int64(len(binaryContent))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(binaryContent)); err != nil {
		t.Fatal(err)
	}
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

func shaHex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// serveRelease points the package at a fake GitHub: /releases/latest plus the
// matching tarball and checksum for this platform.
func serveRelease(t *testing.T, tag, payload string) {
	t.Helper()
	tarball := makeTarball(t, payload)
	sum := shaHex(tarball) + "  " + ArtifactName(runtime.GOOS, runtime.GOARCH) + "\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/releases/latest":
			fmt.Fprintf(w, `{"tag_name": %q}`, tag)
		case r.URL.Path == "/download/"+tag+"/"+ArtifactName(runtime.GOOS, runtime.GOARCH):
			w.Write(tarball)
		case r.URL.Path == "/download/"+tag+"/"+ArtifactName(runtime.GOOS, runtime.GOARCH)+".sha256":
			w.Write([]byte(sum))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	oldURL, oldBase := latestReleaseURL, downloadBase
	t.Cleanup(func() { latestReleaseURL, downloadBase = oldURL, oldBase })
	latestReleaseURL = srv.URL + "/releases/latest"
	downloadBase = srv.URL + "/download"
}

func writeFakeBinary(t *testing.T, dir, content string) string {
	t.Helper()
	p := filepath.Join(dir, BinaryName(runtime.GOOS))
	if err := os.WriteFile(p, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
	return p
}

func readTarget(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestRunUpdateInstallsNewVersion(t *testing.T) {
	dir := t.TempDir()
	target := writeFakeBinary(t, dir, "OLD")
	serveRelease(t, "v9.9.9", "NEW")

	if err := runUpdate("v1.0.0", false, target); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if got := readTarget(t, target); got != "NEW" {
		t.Fatalf("binary not replaced, content = %q", got)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0755 {
		t.Fatalf("permissions = %v, want 0755", info.Mode().Perm())
	}
	leftovers, _ := filepath.Glob(filepath.Join(dir, "*.new"))
	if len(leftovers) != 0 {
		t.Fatalf("staged files left behind: %v", leftovers)
	}
}

func TestRunUpdateUpToDateLeavesBinaryAlone(t *testing.T) {
	dir := t.TempDir()
	target := writeFakeBinary(t, dir, "OLD")
	serveRelease(t, "v1.0.0", "NEW")

	for _, checkOnly := range []bool{false, true} {
		if err := runUpdate("v1.0.0", checkOnly, target); err != nil {
			t.Fatalf("runUpdate(checkOnly=%v): %v", checkOnly, err)
		}
	}
	if got := readTarget(t, target); got != "OLD" {
		t.Fatalf("binary was modified: %q", got)
	}
}

func TestRunUpdateCheckOnlyReportsWithoutInstalling(t *testing.T) {
	dir := t.TempDir()
	target := writeFakeBinary(t, dir, "OLD")
	serveRelease(t, "v2.0.0", "NEW")

	if err := runUpdate("v1.0.0", true, target); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if got := readTarget(t, target); got != "OLD" {
		t.Fatalf("check-only modified the binary: %q", got)
	}
}

func TestRunUpdateRefusesEmbeddedBinary(t *testing.T) {
	os.Unsetenv("PIPE_UPDATE_EMBEDDED")

	dir := t.TempDir()
	src := filepath.Join(dir, "prog.pipe")
	if err := os.WriteFile(src, []byte("print(1)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	embedded := filepath.Join(dir, "embedded-pipe")
	if err := build.BuildWithFiles(src, embedded, nil); err != nil {
		t.Skipf("cannot create embedded binary here: %v", err)
	}
	if _, ok := build.LoadEmbedded(embedded); !ok {
		t.Fatal("test setup: LoadEmbedded does not recognize built binary")
	}
	serveRelease(t, "v9.9.9", "NEW")

	before := readTarget(t, embedded)
	err := runUpdate("v1.0.0", false, embedded)
	if err == nil || !strings.Contains(err.Error(), "PIPE_UPDATE_EMBEDDED") {
		t.Fatalf("expected embedded refusal, got %v", err)
	}
	if readTarget(t, embedded) != before {
		t.Fatal("refused update still replaced the binary")
	}
}

func TestRunUpdateEmbeddedOverrideInstalls(t *testing.T) {
	t.Setenv("PIPE_UPDATE_EMBEDDED", "1")

	dir := t.TempDir()
	src := filepath.Join(dir, "prog.pipe")
	if err := os.WriteFile(src, []byte("print(1)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	embedded := filepath.Join(dir, "embedded-pipe")
	if err := build.BuildWithFiles(src, embedded, nil); err != nil {
		t.Skipf("cannot create embedded binary here: %v", err)
	}
	serveRelease(t, "v9.9.9", "NEW")

	if err := runUpdate("v1.0.0", false, embedded); err != nil {
		t.Fatalf("runUpdate with override: %v", err)
	}
	if got := readTarget(t, embedded); got != "NEW" {
		t.Fatalf("override update did not replace binary: %q", got)
	}
}
