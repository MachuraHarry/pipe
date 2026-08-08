package object

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"math/big"
)

func init() {
	Builtins = append(Builtins,
		BuiltinInfo{Name: "secure_random", Fn: bSecureRandom},
		BuiltinInfo{Name: "secure_random_int", Fn: bSecureRandomInt},
		BuiltinInfo{Name: "secure_random_range", Fn: bSecureRandomRange},
		BuiltinInfo{Name: "secure_random_bytes", Fn: bSecureRandomBytes},
		BuiltinInfo{Name: "encrypt", Fn: bEncrypt},
		BuiltinInfo{Name: "decrypt", Fn: bDecrypt},
		BuiltinInfo{Name: "hmac_sha1", Fn: bHmacSha1},
		BuiltinInfo{Name: "hmac_sha256", Fn: bHmacSha256},
		BuiltinInfo{Name: "hmac_sha512", Fn: bHmacSha512},
	)
}

func bSecureRandom(args ...Object) Object {
	if len(args) != 1 {
		return err("secure_random expects 1 argument (byte_count)")
	}
	count, ok := ToInt(args[0])
	if !ok || count <= 0 || count > 1024 {
		return err("secure_random: byte_count must be a number between 1 and 1024")
	}
	buf := make([]byte, count)
	if _, e := rand.Read(buf); e != nil {
		return err("secure_random: " + e.Error())
	}
	return &String{Value: hex.EncodeToString(buf)}
}

func bSecureRandomInt(args ...Object) Object {
	if len(args) > 0 {
		return err("secure_random_int takes no arguments")
	}
	buf := make([]byte, 8)
	if _, e := rand.Read(buf); e != nil {
		return err("secure_random_int: " + e.Error())
	}
	val := int64(buf[0])<<56 | int64(buf[1])<<48 | int64(buf[2])<<40 | int64(buf[3])<<32 |
		int64(buf[4])<<24 | int64(buf[5])<<16 | int64(buf[6])<<8 | int64(buf[7])
	return &Integer{Value: val}
}

func bSecureRandomRange(args ...Object) Object {
	if len(args) != 2 {
		return err("secure_random_range expects 2 arguments (min, max)")
	}
	min, ok1 := ToInt(args[0])
	max, ok2 := ToInt(args[1])
	if !ok1 || !ok2 {
		return err("secure_random_range: min and max must be numbers")
	}
	if min >= max {
		return err("secure_random_range: min must be less than max")
	}
	n, e := rand.Int(rand.Reader, big.NewInt(max-min))
	if e != nil {
		return err("secure_random_range: " + e.Error())
	}
	return &Integer{Value: min + n.Int64()}
}

func bSecureRandomBytes(args ...Object) Object {
	if len(args) != 1 {
		return err("secure_random_bytes expects 1 argument (byte_count)")
	}
	count, ok := ToInt(args[0])
	if !ok || count <= 0 || count > 1024 {
		return err("secure_random_bytes: byte_count must be a number between 1 and 1024")
	}
	buf := make([]byte, count)
	if _, e := rand.Read(buf); e != nil {
		return err("secure_random_bytes: " + e.Error())
	}
	return &Bytes{Value: buf}
}

func bEncrypt(args ...Object) Object {
	if len(args) < 2 || len(args) > 3 {
		return err("encrypt expects 2-3 arguments (key, plaintext[, associated_data])")
	}
	key, ok := resolveKey(args[0])
	if !ok {
		return err("encrypt: key must be 16, 24, or 32 bytes (AES-128/192/256). Use secure_random 32 for hex key or secure_random_bytes 32 for raw key.")
	}

	var plainBytes []byte
	switch v := args[1].(type) {
	case *String:
		plainBytes = []byte(v.Value)
	case *Bytes:
		plainBytes = v.Value
	default:
		return err("encrypt: plaintext must be a string or bytes")
	}

	var additional []byte
	if len(args) == 3 {
		switch v := args[2].(type) {
		case *String:
			additional = []byte(v.Value)
		case *Bytes:
			additional = v.Value
		default:
			return err("encrypt: associated_data must be a string or bytes")
		}
	}

	block, e := aes.NewCipher(key)
	if e != nil {
		return err("encrypt: " + e.Error())
	}
	gcm, e := cipher.NewGCM(block)
	if e != nil {
		return err("encrypt: " + e.Error())
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, e := rand.Read(nonce); e != nil {
		return err("encrypt: failed to generate nonce")
	}
	ciphertext := gcm.Seal(nonce, nonce, plainBytes, additional)
	return &String{Value: base64.StdEncoding.EncodeToString(ciphertext)}
}

func bDecrypt(args ...Object) Object {
	if len(args) < 2 || len(args) > 3 {
		return err("decrypt expects 2-3 arguments (key, ciphertext[, associated_data])")
	}
	key, ok := resolveKey(args[0])
	if !ok {
		return err("decrypt: key must be 16, 24, or 32 bytes (AES-128/192/256)")
	}
	cipherStr, ok := args[1].(*String)
	if !ok {
		return err("decrypt: ciphertext must be a string (base64)")
	}
	ciphertext, e := base64.StdEncoding.DecodeString(cipherStr.Value)
	if e != nil {
		return err("decrypt: invalid base64: " + e.Error())
	}

	var additional []byte
	if len(args) == 3 {
		switch v := args[2].(type) {
		case *String:
			additional = []byte(v.Value)
		case *Bytes:
			additional = v.Value
		default:
			return err("decrypt: associated_data must be a string or bytes")
		}
	}

	block, e := aes.NewCipher(key)
	if e != nil {
		return err("decrypt: " + e.Error())
	}
	gcm, e := cipher.NewGCM(block)
	if e != nil {
		return err("decrypt: " + e.Error())
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return err("decrypt: ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, e := gcm.Open(nil, nonce, ciphertext, additional)
	if e != nil {
		return err("decrypt: authentication failed — wrong key or tampered data")
	}
	return &String{Value: string(plaintext)}
}

func bHmacSha1(args ...Object) Object {
	if len(args) != 2 {
		return err("hmac_sha1 expects 2 arguments (key, message)")
	}
	keyStr, ok := args[0].(*String)
	if !ok {
		return err("hmac_sha1: key must be a string")
	}
	msgStr, ok := args[1].(*String)
	if !ok {
		return err("hmac_sha1: message must be a string")
	}
	mac := hmac.New(sha1.New, []byte(keyStr.Value))
	mac.Write([]byte(msgStr.Value))
	return &String{Value: hex.EncodeToString(mac.Sum(nil))}
}

func bHmacSha256(args ...Object) Object {
	if len(args) != 2 {
		return err("hmac_sha256 expects 2 arguments (key, message)")
	}
	keyStr, ok := args[0].(*String)
	if !ok {
		return err("hmac_sha256: key must be a string")
	}
	msgStr, ok := args[1].(*String)
	if !ok {
		return err("hmac_sha256: message must be a string")
	}
	mac := hmac.New(sha256.New, []byte(keyStr.Value))
	mac.Write([]byte(msgStr.Value))
	return &String{Value: hex.EncodeToString(mac.Sum(nil))}
}

func bHmacSha512(args ...Object) Object {
	if len(args) != 2 {
		return err("hmac_sha512 expects 2 arguments (key, message)")
	}
	keyStr, ok := args[0].(*String)
	if !ok {
		return err("hmac_sha512: key must be a string")
	}
	msgStr, ok := args[1].(*String)
	if !ok {
		return err("hmac_sha512: message must be a string")
	}
	mac := hmac.New(sha512.New, []byte(keyStr.Value))
	mac.Write([]byte(msgStr.Value))
	return &String{Value: hex.EncodeToString(mac.Sum(nil))}
}

func resolveKey(obj Object) ([]byte, bool) {
	switch v := obj.(type) {
	case *String:
		raw := []byte(v.Value)
		if len(raw) == 16 || len(raw) == 24 || len(raw) == 32 {
			return raw, true
		}
		decoded, e := hex.DecodeString(v.Value)
		if e == nil && (len(decoded) == 16 || len(decoded) == 24 || len(decoded) == 32) {
			return decoded, true
		}
		return nil, false
	case *Bytes:
		if len(v.Value) == 16 || len(v.Value) == 24 || len(v.Value) == 32 {
			return v.Value, true
		}
		return nil, false
	}
	return nil, false
}
