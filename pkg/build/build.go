package build

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

const marker = "PIPEBUILD"

func Build(inputPath, outputPath string) error {
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

	return os.Chmod(outputPath, 0755)
}

func LoadEmbedded(path string) ([]byte, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	markerStr := "\n" + marker + "\n"
	for i := len(data) - 1; i >= len(markerStr); i-- {
		if string(data[i-len(markerStr)+1:i+1]) == markerStr {
			rest := string(data[i+1:])
			var size int
			var newlinePos int
			for j, c := range rest {
				if c == '\n' {
					newlinePos = j
					break
				}
			}
			if _, err := fmt.Sscanf(rest[:newlinePos], "%d", &size); err != nil {
				return nil, false
			}
			srcStart := i + 1 + newlinePos + 1
			if srcStart+size <= len(data) {
				return data[srcStart : srcStart+size], true
			}
			return nil, false
		}
	}
	return nil, false
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
