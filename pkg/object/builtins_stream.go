package object

import (
	"bufio"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// ---- HTTP Streaming (SSE / long-lived responses) ----

type streamHandle struct {
	body   io.ReadCloser
	reader *bufio.Reader
}

var (
	streamRegistry   = map[int]*streamHandle{}
	streamRegistryMu sync.Mutex
	nextStreamHandle = 1
)

func init() {
	Builtins = append(Builtins,
		BuiltinInfo{Name: "http_stream_open", Fn: bHttpStreamOpen},
		BuiltinInfo{Name: "http_stream_read", Fn: bHttpStreamRead},
		BuiltinInfo{Name: "http_stream_read_line", Fn: bHttpStreamReadLine},
		BuiltinInfo{Name: "http_stream_close", Fn: bHttpStreamClose},
	)
}

func getStream(handle int) (*streamHandle, bool) {
	streamRegistryMu.Lock()
	defer streamRegistryMu.Unlock()
	s, ok := streamRegistry[handle]
	return s, ok
}

func bHttpStreamOpen(args ...Object) Object {
	if ActiveProfile.Load().Name != "none" {
		if canErr := ActiveProfile.Load().CanNetwork(); canErr != nil {
			return err(canErr.Error())
		}
	} else if Sandbox.Enabled && !Sandbox.AllowNet {
		return sandboxBlock("http_stream_open (network)")
	}
	if len(args) < 1 || len(args) > 2 {
		return err("http_stream_open expects 1-2 arguments (url, headers?)")
	}
	url, ok := args[0].(*String)
	if !ok {
		return err("http_stream_open: URL must be a string")
	}
	if whitelistErr := ActiveProfile.Load().CanNetworkTo(url.Value); whitelistErr != nil {
		return err(whitelistErr.Error())
	}
	ActiveProfile.Load().Audit("http_stream_open", url.Value)

	headers := make(map[string]string)
	if len(args) >= 2 {
		if h, ok := args[1].(*Map); ok {
			for k, v := range h.Pairs {
				if sv, ok := v.(*String); ok {
					headers[k] = sv.Value
				} else if iv, ok := v.(*Integer); ok {
					headers[k] = strconv.FormatInt(iv.Value, 10)
				} else {
					headers[k] = v.Inspect()
				}
			}
		} else {
			return err("http_stream_open: headers must be a map")
		}
	}

	req, e := http.NewRequest(http.MethodGet, url.Value, nil)
	if e != nil {
		return err("http_stream_open: " + e.Error())
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// No client timeout: the stream stays open until closed or the server ends it.
	client := sandboxHTTPClient(0)
	resp, e := client.Do(req)
	if e != nil {
		return err("http_stream_open: " + e.Error())
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		return err("http_stream_open: HTTP " + strconv.Itoa(resp.StatusCode) + ": " + string(body))
	}

	streamRegistryMu.Lock()
	handle := nextStreamHandle
	nextStreamHandle++
	streamRegistry[handle] = &streamHandle{
		body:   resp.Body,
		reader: bufio.NewReader(resp.Body),
	}
	streamRegistryMu.Unlock()

	return &Integer{Value: int64(handle)}
}

func bHttpStreamRead(args ...Object) Object {
	if len(args) != 1 {
		return err("http_stream_read expects 1 argument (handle)")
	}
	h, ok := ToInt(args[0])
	if !ok {
		return err("http_stream_read: handle must be a number")
	}
	s, exists := getStream(int(h))
	if !exists {
		return err("http_stream_read: invalid handle")
	}
	buf := make([]byte, 4096)
	n, e := s.reader.Read(buf)
	if n > 0 {
		return &String{Value: string(buf[:n])}
	}
	if e != nil {
		if e == io.EOF {
			return NILOBJ
		}
		return err("http_stream_read: " + e.Error())
	}
	return &String{Value: ""}
}

func bHttpStreamReadLine(args ...Object) Object {
	if len(args) != 1 {
		return err("http_stream_read_line expects 1 argument (handle)")
	}
	h, ok := ToInt(args[0])
	if !ok {
		return err("http_stream_read_line: handle must be a number")
	}
	s, exists := getStream(int(h))
	if !exists {
		return err("http_stream_read_line: invalid handle")
	}
	line, e := s.reader.ReadString('\n')
	if e != nil && e != io.EOF {
		return err("http_stream_read_line: " + e.Error())
	}
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	if e == io.EOF && line == "" {
		return NILOBJ
	}
	return &String{Value: line}
}

func bHttpStreamClose(args ...Object) Object {
	if len(args) != 1 {
		return err("http_stream_close expects 1 argument (handle)")
	}
	h, ok := ToInt(args[0])
	if !ok {
		return err("http_stream_close: handle must be a number")
	}

	streamRegistryMu.Lock()
	s, exists := streamRegistry[int(h)]
	if exists {
		delete(streamRegistry, int(h))
	}
	streamRegistryMu.Unlock()

	if !exists {
		return err("http_stream_close: invalid handle")
	}
	if e := s.body.Close(); e != nil {
		return err("http_stream_close: " + e.Error())
	}
	return NILOBJ
}
