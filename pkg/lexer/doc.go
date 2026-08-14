package lexer

import "strings"

// DocLine is a single docstring line captured from a `--!` comment.
// Line is 1-based; Text is the trimmed content after the `--!` marker.
type DocLine struct {
	Line int
	Text string
}

// CollectDocstrings scans source for docstring comment lines starting with
// `--!`. Normal `--` comments are ignored. The scan is line-based and does
// not disturb the regular token stream, so it can be used as a non-invasive
// pre-pass by documentation tools.
func CollectDocstrings(input string) []DocLine {
	var out []DocLine
	lines := strings.Split(input, "\n")
	for i, raw := range lines {
		trimmed := strings.TrimLeft(raw, " \t")
		if !strings.HasPrefix(trimmed, "--!") {
			continue
		}
		text := strings.TrimSpace(trimmed[3:])
		out = append(out, DocLine{Line: i + 1, Text: text})
	}
	return out
}
