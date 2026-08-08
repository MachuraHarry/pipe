package object

import (
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

func bHttpGet(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanNetwork(); canErr != nil {
			return err(canErr.Error())
		}
	} else if Sandbox.Enabled && !Sandbox.AllowNet {
		return sandboxBlock("http_get (network)")
	}
	if len(args) != 1 {
		return err("http_get expects 1 argument (URL)")
	}
	url, ok := args[0].(*String)
	if !ok {
		return err("http_get: URL must be a string")
	}
	if whitelistErr := ActiveProfile.CanNetworkTo(url.Value); whitelistErr != nil {
		return err(whitelistErr.Error())
	}
	ActiveProfile.Audit("http_get", url.Value)
	client := &http.Client{Timeout: 10 * time.Second}
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
	return &Map{Pairs: result}
}

func bHttpPost(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanNetwork(); canErr != nil {
			return err(canErr.Error())
		}
	}
	if len(args) < 1 || len(args) > 2 {
		return err("http_post expects 1-2 arguments (URL, Body?)")
	}
	url, ok := args[0].(*String)
	if !ok {
		return err("http_post: URL must be a string")
	}
	if whitelistErr := ActiveProfile.CanNetworkTo(url.Value); whitelistErr != nil {
		return err(whitelistErr.Error())
	}
	ActiveProfile.Audit("http_post", url.Value)
	var bodyStr string
	if len(args) >= 2 {
		if b, ok := args[1].(*String); ok {
			bodyStr = b.Value
		} else {
			bodyStr = args[1].Inspect()
		}
	}
	client := &http.Client{Timeout: 10 * time.Second}
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
	return &Map{Pairs: result}
}

func bHttpRequest(args ...Object) Object {
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanNetwork(); canErr != nil {
			return err(canErr.Error())
		}
	} else if Sandbox.Enabled && !Sandbox.AllowNet {
		return sandboxBlock("http_request (network)")
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
	if whitelistErr := ActiveProfile.CanNetworkTo(url.Value); whitelistErr != nil {
		return err(whitelistErr.Error())
	}
	ActiveProfile.Audit("http_request", url.Value)

	var bodyStr string
	headers := make(map[string]string)

	if len(args) >= 3 {
		if h, ok := args[2].(*Map); ok {
			for k, v := range h.Pairs {
				if sv, ok := v.(*String); ok {
					headers[k] = sv.Value
				} else if iv, ok := v.(*Integer); ok {
					headers[k] = strconv.FormatInt(iv.Value, 10)
				} else {
					headers[k] = v.Inspect()
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

	client := &http.Client{Timeout: 30 * time.Second}
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
	result["headers"] = &Map{Pairs: respHeaders}
	result["body"] = &String{Value: string(respBody)}
	return &Map{Pairs: result}
}

func bHttpGetJSON(args ...Object) Object {
	resp := bHttpGet(args...)
	if resp.Type() == ERROR {
		return resp
	}
	respMap := resp.(*Map)
	body := respMap.Pairs["body"].(*String)
	return jsonToObject(body.Value)
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
		return &Map{Pairs: pairs}
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
		for k, val := range v.Pairs {
			m[k] = objectToJSON(val)
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
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanNetwork(); canErr != nil {
			return err(canErr.Error())
		}
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
	if ActiveProfile.Name != "none" {
		if canErr := ActiveProfile.CanNetwork(); canErr != nil {
			return err(canErr.Error())
		}
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
	c, e := net.Dial("tcp", addr)
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
	buf := make([]byte, 4096)
	n, e := c.Read(buf)
	if e != nil && e != io.EOF {
		return err("tcp_read: " + e.Error())
	}
	return &String{Value: string(buf[:n])}
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
