// Package rpcframing implements the Content-Length-prefixed message framing
// shared by pipe's stdio-based protocol servers (cmd/pipe-lsp's LSP server,
// cmd/pipe-dap's DAP server). Both protocols use the same wire framing --
// a small header block terminated by a blank line, followed by exactly
// Content-Length bytes of a JSON body -- and differ only in the shape of
// that JSON body, so the framing itself has zero protocol-specific
// knowledge and is safe to share verbatim.
package rpcframing

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ReadMessage reads one Content-Length framed message from r and returns its
// body.
func ReadMessage(r *bufio.Reader) ([]byte, error) {
	contentLength := -1
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "content-length:") {
			n, err := strconv.Atoi(strings.TrimSpace(line[len("Content-Length:"):]))
			if err != nil {
				return nil, errors.New("invalid Content-Length")
			}
			contentLength = n
		}
	}
	if contentLength < 0 {
		return nil, errors.New("missing Content-Length header")
	}
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, err
	}
	return body, nil
}

// WriteMessage writes one Content-Length framed message to w.
func WriteMessage(w io.Writer, body []byte) error {
	if _, err := fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		return err
	}
	_, err := w.Write(body)
	return err
}
