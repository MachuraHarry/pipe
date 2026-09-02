package build

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	marker      = "PIPEBUILD"
	filesMarker = "PIPEFILES"
)

type EmbedFile struct {
	Path string
	Data []byte
}

// safeEmbedName returns the relative, slash-normalized path to store for an
// embedded file, preserving subdirectory structure for legitimate project-
// relative paths (e.g. "modules/telegram.pipe") while refusing to bake in
// anything that climbs outside the current directory (absolute paths, or a
// ".." component anywhere) -- those fall back to the bare base name instead,
// exactly like every embedded file used to be stored before subdirectory
// support was added.
func safeEmbedName(path string) string {
	slashed := filepath.ToSlash(filepath.Clean(path))
	if filepath.IsAbs(slashed) {
		return filepath.Base(path)
	}
	for _, part := range strings.Split(slashed, "/") {
		if part == ".." {
			return filepath.Base(path)
		}
	}
	return slashed
}

func Build(inputPath, outputPath string) error {
	return BuildWithFiles(inputPath, outputPath, nil)
}

func BuildWithFiles(inputPath, outputPath string, embedFiles []EmbedFile) error {
	srcData, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", inputPath, err)
	}

	pipeBin, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding pipe binary: %w", err)
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating %s: %w", outputPath, err)
	}
	defer out.Close()

	pipeFile, err := os.Open(pipeBin)
	if err != nil {
		return fmt.Errorf("opening pipe binary: %w", err)
	}
	defer pipeFile.Close()

	if _, err := io.Copy(out, pipeFile); err != nil {
		return fmt.Errorf("copying pipe binary: %w", err)
	}

	header := fmt.Sprintf("\n%s\n%d\n", marker, len(srcData))
	if _, err := out.WriteString(header); err != nil {
		return err
	}
	if _, err := out.Write(srcData); err != nil {
		return err
	}

	if len(embedFiles) > 0 {
		fh := fmt.Sprintf("\n%s\n%d\n", filesMarker, len(embedFiles))
		if _, err := out.WriteString(fh); err != nil {
			return err
		}
		for _, ef := range embedFiles {
			// Preserve the path as given (e.g. "modules/telegram.pipe"), not
			// just its base name -- a project made of multiple .pipe files
			// that import each other via relative paths (import
			// "modules/foo.pipe") needs that structure to still exist after
			// extraction, or those imports fail to resolve. Stored with
			// forward slashes regardless of OS so the embedded name is a
			// stable, portable identifier; ExtractFiles below converts it
			// back to the host's separator when writing to disk.
			//
			// A path that climbs out of the current directory (absolute, or
			// containing a ".." component) is NOT preserved -- falls back to
			// the pre-existing bare-basename behavior instead, so a stray or
			// malicious --embed-file argument can never bake a traversal
			// path into the binary (see TestBuildWithFilesStripsPaths).
			// Legitimate multi-file projects only ever pass paths relative
			// to (and underneath) their own project root.
			name := safeEmbedName(ef.Path)
			entry := fmt.Sprintf("%s\n%d\n", name, len(ef.Data))
			if _, err := out.WriteString(entry); err != nil {
				return err
			}
			if _, err := out.Write(ef.Data); err != nil {
				return err
			}
		}
	}

	return os.Chmod(outputPath, 0755)
}

func LoadEmbedded(path string) ([]byte, bool) {
	src, _, ok := LoadEmbeddedFiles(path)
	return src, ok
}

func LoadEmbeddedFiles(path string) (src []byte, files map[string][]byte, ok bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, false
	}

	src = extractSection(data, marker)
	if src == nil {
		return nil, nil, false
	}

	filesData := extractFilesSection(data)
	files = make(map[string][]byte)
	if filesData != nil {
		pos := 0

		nl := indexByte(filesData, '\n')
		if nl < 0 {
			return nil, nil, false
		}
		pos = nl + 1

		for pos < len(filesData) {
			nl := indexByte(filesData[pos:], '\n')
			if nl < 0 {
				break
			}
			name := string(filesData[pos : pos+nl])
			pos += nl + 1

			nl2 := indexByte(filesData[pos:], '\n')
			if nl2 < 0 {
				break
			}
			var size int
			if _, err := fmt.Sscanf(string(filesData[pos:pos+nl2]), "%d", &size); err != nil {
				break
			}
			pos += nl2 + 1

			if pos+size <= len(filesData) {
				files[name] = filesData[pos : pos+size]
				pos += size
			} else {
				break
			}
		}
	}

	return src, files, true
}

func extractSection(data []byte, markerName string) []byte {
	markerStr := "\n" + markerName + "\n"
	markerLen := len(markerStr)

	for i := len(data) - 1; i >= markerLen; i-- {
		if string(data[i-markerLen+1:i+1]) == markerStr {
			rest := string(data[i+1:])
			var size int
			nl := indexByteStr(rest, '\n')
			if nl < 0 {
				return nil
			}
			if _, err := fmt.Sscanf(rest[:nl], "%d", &size); err != nil {
				return nil
			}
			srcStart := i + 1 + nl + 1
			if srcStart+size <= len(data) {
				result := make([]byte, size)
				copy(result, data[srcStart:srcStart+size])
				return result
			}
			return nil
		}
	}
	return nil
}

func extractFilesSection(data []byte) []byte {
	markerStr := "\n" + filesMarker + "\n"
	markerLen := len(markerStr)

	for i := len(data) - 1; i >= markerLen; i-- {
		if string(data[i-markerLen+1:i+1]) == markerStr {
			return data[i+1:]
		}
	}
	return nil
}

func indexByte(data []byte, b byte) int {
	for i, c := range data {
		if c == b {
			return i
		}
	}
	return -1
}

func indexByteStr(s string, b rune) int {
	for i, c := range s {
		if c == b {
			return i
		}
	}
	return -1
}

func ExtractFiles(path string) (string, error) {
	_, files, ok := LoadEmbeddedFiles(path)
	if !ok {
		return "", fmt.Errorf("no embedded content found in %s", path)
	}
	if len(files) == 0 {
		return "", nil
	}

	dir, err := os.MkdirTemp("", "pipe_embedded_")
	if err != nil {
		return "", err
	}

	for name, data := range files {
		// Recreate the embedded relative path (e.g. "modules/telegram.pipe")
		// under dir instead of flattening to just the base name -- see the
		// matching comment in BuildWithFiles. filepath.Clean + rejecting any
		// result that escapes dir guards against a maliciously/accidentally
		// crafted embedded name (e.g. "../../etc/passwd") writing outside the
		// temp directory (a "zip slip"-style path traversal).
		cleaned := filepath.Clean(filepath.FromSlash(name))
		dst := filepath.Join(dir, cleaned)
		if dst != dir && !strings.HasPrefix(dst, dir+string(os.PathSeparator)) {
			return dir, fmt.Errorf("embedded file %q escapes extraction directory", name)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return dir, err
		}
		if err := os.WriteFile(dst, data, 0644); err != nil {
			return dir, err
		}
	}
	return dir, nil
}

func IsWritable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode()&0200 != 0
}

func GoBuild(inputPath, outputPath string) error {
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("go compiler not found in PATH")
	}
	return Build(inputPath, outputPath)
}
