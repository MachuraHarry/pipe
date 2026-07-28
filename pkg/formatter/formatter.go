package formatter

import (
	"bytes"
	"os"
	"strings"
)

func Format(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	result := FormatSource(string(data))
	return os.WriteFile(path, []byte(result), 0644)
}

func FormatSource(src string) string {
	lines := strings.Split(src, "\n")
	var out bytes.Buffer

	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t\r")
		if trimmed == "" {
			out.WriteByte('\n')
			continue
		}

		currentIndent := countLeadingSpaces(trimmed)
		content := strings.TrimLeft(trimmed, " \t")
		normIndent := (currentIndent / 4) * 4

		for i := 0; i < normIndent; i++ {
			out.WriteByte(' ')
		}
		content = normalizeSpacing(content)
		out.WriteString(content)
		out.WriteByte('\n')
	}

	result := out.String()
	if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	return result
}

func countLeadingSpaces(s string) int {
	count := 0
	for _, ch := range s {
		if ch == ' ' {
			count++
		} else if ch == '\t' {
			count += 4
		} else {
			break
		}
	}
	return count
}

func normalizeSpacing(s string) string {
	s = strings.ReplaceAll(s, " ,", ",")
	s = strings.ReplaceAll(s, ", ", ", ")
	return strings.TrimSpace(s)
}
