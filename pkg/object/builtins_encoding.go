package object

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
)

func bBase64Encode(args ...Object) Object {
	if len(args) != 1 {
		return err("base64_encode expects 1 argument")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("base64_encode expects a string")
	}
	return &String{Value: base64.StdEncoding.EncodeToString([]byte(s.Value))}
}

func bBase64Decode(args ...Object) Object {
	if len(args) != 1 {
		return err("base64_decode expects 1 argument")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("base64_decode expects a string")
	}
	b, e := base64.StdEncoding.DecodeString(s.Value)
	if e != nil {
		return err("base64_decode: " + e.Error())
	}
	return &String{Value: string(b)}
}

func bBase64URLEncode(args ...Object) Object {
	if len(args) != 1 {
		return err("base64url_encode expects 1 argument")
	}
	switch v := args[0].(type) {
	case *String:
		return &String{Value: base64.RawURLEncoding.EncodeToString([]byte(v.Value))}
	case *Bytes:
		return &String{Value: base64.RawURLEncoding.EncodeToString(v.Value)}
	default:
		return err("base64url_encode expects a string or bytes")
	}
}

func bBase64URLDecode(args ...Object) Object {
	if len(args) != 1 {
		return err("base64url_decode expects 1 argument")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("base64url_decode expects a string")
	}
	b, e := base64.RawURLEncoding.DecodeString(s.Value)
	if e != nil {
		return err("base64url_decode: " + e.Error())
	}
	return &String{Value: string(b)}
}

func bURLEncode(args ...Object) Object {
	if len(args) != 1 {
		return err("url_encode expects 1 argument")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("url_encode expects a string")
	}
	// QueryEscape leaves spaces as '+'; RFC 3986 percent-encoding requires '%20'.
	return &String{Value: strings.ReplaceAll(url.QueryEscape(s.Value), "+", "%20")}
}

func bURLDecode(args ...Object) Object {
	if len(args) != 1 {
		return err("url_decode expects 1 argument")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("url_decode expects a string")
	}
	dec, e := url.QueryUnescape(s.Value)
	if e != nil {
		return err("url_decode: " + e.Error())
	}
	return &String{Value: dec}
}

func bSha256(args ...Object) Object {
	if len(args) != 1 {
		return err("sha256 expects 1 argument (text)")
	}
	switch v := args[0].(type) {
	case *String:
		h := sha256.Sum256([]byte(v.Value))
		return &String{Value: fmt.Sprintf("%x", h)}
	case *Bytes:
		h := sha256.Sum256(v.Value)
		return &Bytes{Value: h[:]}
	default:
		return err("sha256: argument must be a string or bytes")
	}
}

func bMd5(args ...Object) Object {
	if len(args) != 1 {
		return err("md5 expects 1 argument (text)")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("md5: argument must be a string")
	}
	h := md5.Sum([]byte(s.Value))
	return &String{Value: fmt.Sprintf("%x", h)}
}

func bSha1(args ...Object) Object {
	if len(args) != 1 {
		return err("sha1 expects 1 argument (text)")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("sha1: argument must be a string")
	}
	h := sha1.Sum([]byte(s.Value))
	return &String{Value: fmt.Sprintf("%x", h)}
}

func bSha512(args ...Object) Object {
	if len(args) != 1 {
		return err("sha512 expects 1 argument (text)")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("sha512: argument must be a string")
	}
	h := sha512.Sum512([]byte(s.Value))
	return &String{Value: fmt.Sprintf("%x", h)}
}
