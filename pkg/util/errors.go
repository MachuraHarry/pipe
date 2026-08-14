package util

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var parserErrRe = regexp.MustCompile(`^line (\d+) col (\d+): (.*)$`)

// ParseParserError splits a parser error string like
// "line 3 col 5: message" into its components. Returns ok=false when the
// string does not follow the parser error format.
func ParseParserError(e string) (line, col int, msg string, ok bool) {
	m := parserErrRe.FindStringSubmatch(strings.TrimSpace(e))
	if m == nil {
		return 0, 0, "", false
	}
	line, err1 := strconv.Atoi(m[1])
	col, err2 := strconv.Atoi(m[2])
	if err1 != nil || err2 != nil {
		return 0, 0, "", false
	}
	return line, col, m[3], true
}

// LineText returns the 1-based line of source, or ok=false when out of range.
func LineText(source string, line int) (string, bool) {
	if line <= 0 {
		return "", false
	}
	lines := strings.Split(source, "\n")
	if line > len(lines) {
		return "", false
	}
	return lines[line-1], true
}

// Snippet renders a source context block with a caret marker under the
// offending column, e.g.:
//
//	3 | read_file "x.txt" ++ 1
//	  |                   ^
//
// Returns "" when the position cannot be resolved in source.
func Snippet(source string, line, col int) string {
	text, ok := LineText(source, line)
	if !ok {
		return ""
	}
	if col < 1 {
		col = 1
	}
	if col-1 > len(text) {
		col = len(text) + 1
	}
	gutter := fmt.Sprintf("%d | ", line)
	padding := strings.Repeat(" ", len(gutter))
	caret := strings.Repeat(" ", col-1) + "^"
	return gutter + text + "\n" + padding + caret
}

// FormatErrorWithSnippet renders a parser error string, appending a source
// snippet whenever the position is resolvable.
func FormatErrorWithSnippet(source, err string) string {
	line, col, msg, ok := ParseParserError(err)
	if !ok {
		return err
	}
	out := fmt.Sprintf("line %d col %d: %s", line, col, msg)
	if snip := Snippet(source, line, col); snip != "" {
		out += "\n" + snip
	}
	return out
}
