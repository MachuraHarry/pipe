package object

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ---- Connection type for TCP ----

type ConnHandle int

var (
	connMu     sync.Mutex
	connNextID ConnHandle = 1
	connStore             = make(map[ConnHandle]net.Conn)
	listeners             = make(map[ConnHandle]net.Listener)
)

type TcpConn struct {
	Handle ConnHandle
}

func (tc *TcpConn) Type() ObjectType { return "TCP_CONN" }
func (tc *TcpConn) Inspect() string  { return fmt.Sprintf("tcp:%d", tc.Handle) }

type TcpListener struct {
	Handle ConnHandle
}

func (tl *TcpListener) Type() ObjectType { return "TCP_LISTENER" }
func (tl *TcpListener) Inspect() string  { return fmt.Sprintf("tcp-listener:%d", tl.Handle) }

// ---- Network ----

// sandboxHTTPClient returns an http.Client whose redirect hops are re-checked
// against the active profile's network whitelist.
func sandboxHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("E_SANDBOX: too many redirects")
			}
			if ActiveProfile.Load().Name != "none" {
				if canErr := ActiveProfile.Load().CanNetworkTo(req.URL.String()); canErr != nil {
					return canErr
				}
			}
			return nil
		},
	}
}

func bHttpGet(args ...Object) Object {
	if blockErr := checkNetworkAccess("http_get (network)"); blockErr != nil {
		return blockErr
	}
	if len(args) != 1 {
		return err("http_get expects 1 argument (URL)")
	}
	url, ok := args[0].(*String)
	if !ok {
		return err("http_get: URL must be a string")
	}
	if whitelistErr := ActiveProfile.Load().CanNetworkTo(url.Value); whitelistErr != nil {
		return err(whitelistErr.Error())
	}
	ActiveProfile.Load().Audit("http_get", url.Value)
	client := sandboxHTTPClient(10 * time.Second)
	resp, e := client.Get(url.Value)
	if e != nil {
		return err("http_get: " + e.Error())
	}
	defer resp.Body.Close()
	body, e := io.ReadAll(resp.Body)
	if e != nil {
		return err("http_get: " + e.Error())
	}
	result := make(map[string]Object)
	result["status"] = &Integer{Value: int64(resp.StatusCode)}
	result["body"] = &String{Value: string(body)}
	return MapFromGo(result)
}

func bHttpPost(args ...Object) Object {
	if blockErr := checkNetworkAccess("http_post (network)"); blockErr != nil {
		return blockErr
	}
	if len(args) < 1 || len(args) > 2 {
		return err("http_post expects 1-2 arguments (URL, Body?)")
	}
	url, ok := args[0].(*String)
	if !ok {
		return err("http_post: URL must be a string")
	}
	if whitelistErr := ActiveProfile.Load().CanNetworkTo(url.Value); whitelistErr != nil {
		return err(whitelistErr.Error())
	}
	ActiveProfile.Load().Audit("http_post", url.Value)
	var bodyStr string
	if len(args) >= 2 {
		if b, ok := args[1].(*String); ok {
			bodyStr = b.Value
		} else {
			bodyStr = args[1].Inspect()
		}
	}
	client := sandboxHTTPClient(10 * time.Second)
	var bodyReader io.Reader
	if bodyStr != "" {
		bodyReader = strings.NewReader(bodyStr)
	}
	resp, e := client.Post(url.Value, "application/json", bodyReader)
	if e != nil {
		return err("http_post: " + e.Error())
	}
	defer resp.Body.Close()
	respBody, e := io.ReadAll(resp.Body)
	if e != nil {
		return err("http_post: " + e.Error())
	}
	result := make(map[string]Object)
	result["status"] = &Integer{Value: int64(resp.StatusCode)}
	result["body"] = &String{Value: string(respBody)}
	return MapFromGo(result)
}

func bHttpRequest(args ...Object) Object {
	if blockErr := checkNetworkAccess("http_request (network)"); blockErr != nil {
		return blockErr
	}
	if len(args) < 2 || len(args) > 4 {
		return err("http_request expects 2-4 arguments (method, url, headers?, body?)")
	}
	method, ok := args[0].(*String)
	if !ok {
		return err("http_request: method must be a string")
	}
	url, ok := args[1].(*String)
	if !ok {
		return err("http_request: URL must be a string")
	}
	if whitelistErr := ActiveProfile.Load().CanNetworkTo(url.Value); whitelistErr != nil {
		return err(whitelistErr.Error())
	}
	ActiveProfile.Load().Audit("http_request", url.Value)

	var bodyStr string
	headers := make(map[string]string)

	if len(args) >= 3 {
		if h, ok := args[2].(*Map); ok {
			for _, p := range h.Pairs {
				if sv, ok := p.Value.(*String); ok {
					headers[p.Key] = sv.Value
				} else if iv, ok := p.Value.(*Integer); ok {
					headers[p.Key] = strconv.FormatInt(iv.Value, 10)
				} else {
					headers[p.Key] = p.Value.Inspect()
				}
			}
		}
	}

	if len(args) >= 4 {
		if b, ok := args[3].(*String); ok {
			bodyStr = b.Value
		} else {
			bodyStr = args[3].Inspect()
		}
	}

	client := sandboxHTTPClient(30 * time.Second)
	var bodyReader io.Reader
	if bodyStr != "" {
		bodyReader = strings.NewReader(bodyStr)
	}

	req, e := http.NewRequest(method.Value, url.Value, bodyReader)
	if e != nil {
		return err("http_request: " + e.Error())
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, e := client.Do(req)
	if e != nil {
		return err("http_request: " + e.Error())
	}
	defer resp.Body.Close()

	respBody, e := io.ReadAll(resp.Body)
	if e != nil {
		return err("http_request: " + e.Error())
	}

	respHeaders := make(map[string]Object)
	for k, vals := range resp.Header {
		if len(vals) > 0 {
			respHeaders[k] = &String{Value: vals[0]}
		}
	}

	result := make(map[string]Object)
	result["status"] = &Integer{Value: int64(resp.StatusCode)}
	result["headers"] = MapFromGo(respHeaders)
	result["body"] = &String{Value: string(respBody)}
	return MapFromGo(result)
}

func bHttpGetJSON(args ...Object) Object {
	resp := bHttpGet(args...)
	if resp.Type() == ERROR {
		return resp
	}
	respMap := resp.(*Map)
	body, _ := respMap.Get("body")
	return jsonToObject(body.(*String).Value)
}

func bParseJSON(args ...Object) Object {
	if len(args) != 1 {
		return err("parse_json expects 1 argument")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("parse_json expects a string")
	}
	return jsonToObject(s.Value)
}

func bToJSON(args ...Object) Object {
	if len(args) != 1 {
		return err("to_json expects 1 argument")
	}
	j, e := json.Marshal(objectToJSON(args[0]))
	if e != nil {
		return err("to_json: " + e.Error())
	}
	return &String{Value: string(j)}
}

func jsonToObject(raw string) Object {
	var data interface{}
	if e := json.Unmarshal([]byte(raw), &data); e != nil {
		return err("parse_json: " + e.Error())
	}
	return convertJSON(data)
}

func convertJSON(data interface{}) Object {
	switch v := data.(type) {
	case map[string]interface{}:
		pairs := make(map[string]Object)
		for k, val := range v {
			pairs[k] = convertJSON(val)
		}
		return MapFromGo(pairs)
	case []interface{}:
		elems := make([]Object, len(v))
		for i, val := range v {
			elems[i] = convertJSON(val)
		}
		return &List{Elements: elems}
	case float64:
		if v >= math.MinInt64 && v <= math.MaxInt64 && v == float64(int64(v)) {
			return &Integer{Value: int64(v)}
		}
		return &Float{Value: v}
	case string:
		return &String{Value: v}
	case bool:
		return NativeBoolToBoolean(v)
	case nil:
		return NILOBJ
	}
	return NILOBJ
}

func objectToJSON(obj Object) interface{} {
	switch v := obj.(type) {
	case *Map:
		m := make(map[string]interface{})
		for _, p := range v.Pairs {
			m[p.Key] = objectToJSON(p.Value)
		}
		return m
	case *List:
		l := make([]interface{}, len(v.Elements))
		for i, val := range v.Elements {
			l[i] = objectToJSON(val)
		}
		return l
	case *Integer:
		return v.Value
	case *Float:
		return v.Value
	case *String:
		return v.Value
	case *Bytes:
		return base64.StdEncoding.EncodeToString(v.Value)
	case *Boolean:
		return v.Value
	case *NilObject:
		return nil
	}
	return obj.Inspect()
}

// ---- TCP ----

func bTcpListen(args ...Object) Object {
	if blockErr := checkNetworkAccess("tcp_listen (network)"); blockErr != nil {
		return blockErr
	}
	if len(args) != 2 {
		return err("tcp_listen expects 2 arguments (Host, Port)")
	}
	host, ok := args[0].(*String)
	if !ok {
		return err("tcp_listen: Host must be a string")
	}
	port, ok := ToInt(args[1])
	if !ok {
		return err("tcp_listen: Port must be a number")
	}
	addr := fmt.Sprintf("%s:%d", host.Value, port)
	if ActiveProfile.Load().Name != "none" {
		if canErr := ActiveProfile.Load().CanNetworkTo(addr); canErr != nil {
			return err(canErr.Error())
		}
	}
	ln, e := net.Listen("tcp", addr)
	if e != nil {
		return err("tcp_listen: " + e.Error())
	}
	connMu.Lock()
	h := connNextID
	connNextID++
	listeners[h] = ln
	connMu.Unlock()
	return &TcpListener{Handle: h}
}

func bTcpConnect(args ...Object) Object {
	if blockErr := checkNetworkAccess("tcp_connect (network)"); blockErr != nil {
		return blockErr
	}
	if len(args) != 2 {
		return err("tcp_connect expects 2 arguments (Host, Port)")
	}
	host, ok := args[0].(*String)
	if !ok {
		return err("tcp_connect: Host must be a string")
	}
	port, ok := ToInt(args[1])
	if !ok {
		return err("tcp_connect: Port must be a number")
	}
	addr := net.JoinHostPort(host.Value, strconv.FormatInt(port, 10))
	if ActiveProfile.Load().Name != "none" {
		if canErr := ActiveProfile.Load().CanNetworkTo(addr); canErr != nil {
			return err(canErr.Error())
		}
	}
	profile := ActiveProfile.Load()
	var c net.Conn
	var e error
	if profile.Name != "none" && profile.Timeout > 0 {
		c, e = net.DialTimeout("tcp", addr, time.Duration(profile.Timeout)*time.Second)
	} else {
		c, e = net.Dial("tcp", addr)
	}
	if e != nil {
		return err("tcp_connect: " + e.Error())
	}
	connMu.Lock()
	h := connNextID
	connNextID++
	connStore[h] = c
	connMu.Unlock()
	return &TcpConn{Handle: h}
}

func bTcpConnectTLS(args ...Object) Object {
	if blockErr := checkNetworkAccess("tcp_connect_tls (network)"); blockErr != nil {
		return blockErr
	}
	if len(args) < 2 || len(args) > 4 {
		return err("tcp_connect_tls expects 2-4 arguments (host, port, servername?, insecure?)")
	}
	host, ok := args[0].(*String)
	if !ok {
		return err("tcp_connect_tls: host must be a string")
	}
	port, ok := ToInt(args[1])
	if !ok {
		return err("tcp_connect_tls: port must be a number")
	}

	servername := host.Value
	if len(args) >= 3 {
		if sn, ok := args[2].(*String); ok && sn.Value != "" {
			servername = sn.Value
		}
	}
	insecure := false
	if len(args) >= 4 {
		if b, ok := args[3].(*Boolean); ok {
			insecure = b.Value
		}
	}

	addr := net.JoinHostPort(host.Value, strconv.FormatInt(port, 10))
	profile := ActiveProfile.Load()
	if profile.Name != "none" {
		if canErr := profile.CanNetworkTo(addr); canErr != nil {
			return err(canErr.Error())
		}
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	if profile.Name != "none" && profile.Timeout > 0 {
		dialer.Timeout = time.Duration(profile.Timeout) * time.Second
	}

	conn, e := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
		ServerName:         servername,
		InsecureSkipVerify: insecure,
	})
	if e != nil {
		return err("tcp_connect_tls: " + e.Error())
	}

	connMu.Lock()
	h := connNextID
	connNextID++
	connStore[h] = conn
	connMu.Unlock()
	return &TcpConn{Handle: h}
}

func bTcpAccept(args ...Object) Object {
	if len(args) != 1 {
		return err("tcp_accept expects 1 argument (Listener)")
	}
	ln, ok := args[0].(*TcpListener)
	if !ok {
		return err("tcp_accept expects a TCP listener")
	}
	connMu.Lock()
	listener, exists := listeners[ln.Handle]
	connMu.Unlock()
	if !exists {
		return err("tcp_accept: listener does not exist")
	}
	c, e := listener.Accept()
	if e != nil {
		return err("tcp_accept: " + e.Error())
	}
	connMu.Lock()
	h := connNextID
	connNextID++
	connStore[h] = c
	connMu.Unlock()
	return &TcpConn{Handle: h}
}

func bTcpRead(args ...Object) Object {
	if len(args) != 1 {
		return err("tcp_read expects 1 argument (connection)")
	}
	conn, ok := args[0].(*TcpConn)
	if !ok {
		return err("tcp_read expects a TCP connection")
	}
	connMu.Lock()
	c, exists := connStore[conn.Handle]
	connMu.Unlock()
	if !exists {
		return err("tcp_read: connection does not exist")
	}
	profile := ActiveProfile.Load()
	if profile.Name != "none" && profile.Timeout > 0 {
		c.SetReadDeadline(time.Now().Add(time.Duration(profile.Timeout) * time.Second))
	}
	buf := make([]byte, 4096)
	n, e := c.Read(buf)
	if e != nil && e != io.EOF {
		return err("tcp_read: " + e.Error())
	}
	return &String{Value: string(buf[:n])}
}

func bTcpReadBytes(args ...Object) Object {
	if len(args) != 2 {
		return err("tcp_read_bytes expects 2 arguments (connection, byte_count)")
	}
	conn, ok := args[0].(*TcpConn)
	if !ok {
		return err("tcp_read_bytes: first argument must be a TCP connection")
	}
	n, ok := ToInt(args[1])
	if !ok || n <= 0 {
		return err("tcp_read_bytes: byte_count must be a positive number")
	}
	connMu.Lock()
	c, exists := connStore[conn.Handle]
	connMu.Unlock()
	if !exists {
		return err("tcp_read_bytes: connection does not exist")
	}
	profile := ActiveProfile.Load()
	if profile.Name != "none" && profile.Timeout > 0 {
		c.SetReadDeadline(time.Now().Add(time.Duration(profile.Timeout) * time.Second))
	}
	buf := make([]byte, n)
	_, e := io.ReadFull(c, buf)
	if e != nil {
		return err("tcp_read_bytes: " + e.Error())
	}
	return &Bytes{Value: buf}
}

func bTcpSetReadTimeout(args ...Object) Object {
	if len(args) != 2 {
		return err("tcp_set_read_timeout expects 2 arguments (connection, milliseconds)")
	}
	conn, ok := args[0].(*TcpConn)
	if !ok {
		return err("tcp_set_read_timeout: first argument must be a TCP connection")
	}
	ms, ok := ToInt(args[1])
	if !ok || ms < 0 {
		return err("tcp_set_read_timeout: milliseconds must be a non-negative number")
	}
	connMu.Lock()
	c, exists := connStore[conn.Handle]
	connMu.Unlock()
	if !exists {
		return err("tcp_set_read_timeout: connection does not exist")
	}
	if ms == 0 {
		c.SetReadDeadline(time.Time{})
	} else {
		c.SetReadDeadline(time.Now().Add(time.Duration(ms) * time.Millisecond))
	}
	return NILOBJ
}

func bTcpWrite(args ...Object) Object {
	if len(args) != 2 {
		return err("tcp_write expects 2 arguments (connection, data)")
	}
	conn, ok := args[0].(*TcpConn)
	if !ok {
		return err("tcp_write expects a TCP connection")
	}
	data, ok := args[1].(*String)
	if !ok {
		return err("tcp_write: data must be a string")
	}
	connMu.Lock()
	c, exists := connStore[conn.Handle]
	connMu.Unlock()
	if !exists {
		return err("tcp_write: connection does not exist")
	}
	_, e := c.Write([]byte(data.Value))
	if e != nil {
		return err("tcp_write: " + e.Error())
	}
	return NILOBJ
}

func bTcpClose(args ...Object) Object {
	if len(args) != 1 {
		return err("tcp_close expects 1 argument")
	}
	switch v := args[0].(type) {
	case *TcpConn:
		connMu.Lock()
		if c, ok := connStore[v.Handle]; ok {
			c.Close()
			delete(connStore, v.Handle)
		}
		connMu.Unlock()
	case *TcpListener:
		connMu.Lock()
		if ln, ok := listeners[v.Handle]; ok {
			ln.Close()
			delete(listeners, v.Handle)
		}
		connMu.Unlock()
	default:
		return err("tcp_close: invalid type")
	}
	return NILOBJ
}
