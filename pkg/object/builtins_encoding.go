package object

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
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

func bSha256(args ...Object) Object {
	if len(args) != 1 {
		return err("sha256 expects 1 argument (text)")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("sha256: argument must be a string")
	}
	h := sha256.Sum256([]byte(s.Value))
	return &String{Value: fmt.Sprintf("%x", h)}
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
