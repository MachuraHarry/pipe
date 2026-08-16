package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSource(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "script.pipe")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	return path
}

func TestBuildLoadEmbeddedRoundTrip(t *testing.T) {
	src := "print \"hello from embedded\"\nprint (1 + 2)\n"
	input := writeSource(t, src)
	output := filepath.Join(t.TempDir(), "my_prog")

	if err := Build(input, output); err != nil {
		t.Fatalf("Build: %v", err)
	}

	info, err := os.Stat(output)
	if err != nil {
		t.Fatalf("stat output: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Errorf("expected executable bit on %s, got %v", output, info.Mode())
	}

	got, ok := LoadEmbedded(output)
	if !ok {
		t.Fatal("LoadEmbedded: expected embedded content")
	}
	if string(got) != src {
		t.Errorf("source mismatch:\n got: %q\nwant: %q", string(got), src)
	}
}

func TestBuildWithFilesRoundTrip(t *testing.T) {
	src := "print \"with files\"\n"
	input := writeSource(t, src)
	output := filepath.Join(t.TempDir(), "prog")

	embedded := []EmbedFile{
		{Path: "assets/data.txt", Data: []byte("file one")},
		{Path: "assets/blob.bin", Data: []byte{0x00, 0x01, 0xFF}},
	}

	if err := BuildWithFiles(input, output, embedded); err != nil {
		t.Fatalf("BuildWithFiles: %v", err)
	}

	gotSrc, gotFiles, ok := LoadEmbeddedFiles(output)
	if !ok {
		t.Fatal("LoadEmbeddedFiles: expected embedded content")
	}
	if string(gotSrc) != src {
		t.Errorf("source mismatch:\n got: %q\nwant: %q", string(gotSrc), src)
	}
	if len(gotFiles) != 2 {
		t.Fatalf("expected 2 embedded files, got %d", len(gotFiles))
	}
	if string(gotFiles["data.txt"]) != "file one" {
		t.Errorf("data.txt mismatch: %q", string(gotFiles["data.txt"]))
	}
	if string(gotFiles["blob.bin"]) != string([]byte{0x00, 0x01, 0xFF}) {
		t.Errorf("blob.bin mismatch: %v", gotFiles["blob.bin"])
	}
}

func TestBuildWithFilesStripsPaths(t *testing.T) {
	input := writeSource(t, "print 1\n")
	output := filepath.Join(t.TempDir(), "prog")

	embedded := []EmbedFile{{Path: "../../evil/secret.txt", Data: []byte("secret")}}
	if err := BuildWithFiles(input, output, embedded); err != nil {
		t.Fatalf("BuildWithFiles: %v", err)
	}

	_, files, ok := LoadEmbeddedFiles(output)
	if !ok {
		t.Fatal("LoadEmbeddedFiles: expected embedded content")
	}
	if _, ok := files["secret.txt"]; !ok {
		t.Errorf("expected basename secret.txt in files, got %v", files)
	}
	for name := range files {
		if strings.Contains(name, "/") || strings.Contains(name, "..") {
			t.Errorf("embedded name must be a bare basename, got %q", name)
		}
	}
}

func TestBuildMissingInput(t *testing.T) {
	output := filepath.Join(t.TempDir(), "prog")
	err := Build(filepath.Join(t.TempDir(), "nope.pipe"), output)
	if err == nil {
		t.Fatal("expected error for missing input file")
	}
	if !strings.Contains(err.Error(), "nope.pipe") {
		t.Errorf("error should mention input path, got: %v", err)
	}
}

func TestLoadEmbeddedOnPlainFile(t *testing.T) {
	path := writeSource(t, "print 1\n")
	if _, ok := LoadEmbedded(path); ok {
		t.Error("LoadEmbedded: expected ok=false for a plain file")
	}
}

func TestLoadEmbeddedFilesNoEmbeddedSection(t *testing.T) {
	src := "print \"no files\"\n"
	input := writeSource(t, src)
	output := filepath.Join(t.TempDir(), "prog")

	if err := Build(input, output); err != nil {
		t.Fatalf("Build: %v", err)
	}

	gotSrc, files, ok := LoadEmbeddedFiles(output)
	if !ok {
		t.Fatal("LoadEmbeddedFiles: expected embedded content")
	}
	if string(gotSrc) != src {
		t.Errorf("source mismatch: %q", string(gotSrc))
	}
	if len(files) != 0 {
		t.Errorf("expected no embedded files, got %v", files)
	}
}

func TestExtractFiles(t *testing.T) {
	input := writeSource(t, "print 1\n")
	output := filepath.Join(t.TempDir(), "prog")

	embedded := []EmbedFile{
		{Path: "a.txt", Data: []byte("AAA")},
		{Path: "b.txt", Data: []byte("BBB")},
	}
	if err := BuildWithFiles(input, output, embedded); err != nil {
		t.Fatalf("BuildWithFiles: %v", err)
	}

	dir, err := ExtractFiles(output)
	if err != nil {
		t.Fatalf("ExtractFiles: %v", err)
	}
	defer os.RemoveAll(dir)

	for name, want := range map[string]string{"a.txt": "AAA", "b.txt": "BBB"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Errorf("read %s: %v", name, err)
			continue
		}
		if string(data) != want {
			t.Errorf("%s mismatch: got %q want %q", name, string(data), want)
		}
	}
}

func TestExtractFilesOnPlainFile(t *testing.T) {
	path := writeSource(t, "print 1\n")
	if _, err := ExtractFiles(path); err == nil {
		t.Error("expected error extracting from a plain file")
	}
}

func TestIsWritable(t *testing.T) {
	dir := t.TempDir()

	writable := filepath.Join(dir, "w.pipe")
	if err := os.WriteFile(writable, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if !IsWritable(writable) {
		t.Error("expected writable file to be writable")
	}

	ro := filepath.Join(dir, "ro.pipe")
	if err := os.WriteFile(ro, []byte("x"), 0444); err != nil {
		t.Fatal(err)
	}
	if IsWritable(ro) {
		t.Error("expected read-only file to be not writable")
	}

	if IsWritable(filepath.Join(dir, "missing.pipe")) {
		t.Error("expected missing file to be not writable")
	}
}

func TestGoBuildMissingGo(t *testing.T) {
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "/nonexistent-bin")
	defer os.Setenv("PATH", oldPath)

	err := GoBuild("whatever.pipe", "whatever")
	if err == nil {
		t.Fatal("expected error when go is not in PATH")
	}
	if !strings.Contains(err.Error(), "go compiler not found") {
		t.Errorf("unexpected error: %v", err)
	}
}
