package object

import (
	"encoding/base64"
	"net/http"
	"os"
	"strings"

	"github.com/MachuraHarry/pipe/pkg/ai"
)

var acceptedImageMIMEs = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// resolveImageURL turns an ai_vision image argument into a URL string ready
// for ai.VisionChat. An http(s) URL is passed through untouched — the
// provider's servers fetch it, not Pipe. A *String that isn't a URL is
// treated as a local file path and read via the same fs-read gate as
// read_file (reads are intentionally not restricted by the CLI --sandbox
// flag, only writes are — see checkFSWriteAccess and the round-7 audit).
// A *Bytes value is used directly, no filesystem access needed.
func resolveImageURL(img Object) (string, *Error) {
	switch v := img.(type) {
	case *String:
		if strings.HasPrefix(v.Value, "http://") || strings.HasPrefix(v.Value, "https://") {
			return v.Value, nil
		}
		path := v.Value
		if ActiveProfile.Load().Name != "none" {
			var cerr error
			path, cerr = ActiveProfile.Load().canonicalRead(v.Value)
			if cerr != nil {
				return "", &Error{Message: cerr.Error()}
			}
		}
		data, e := os.ReadFile(path)
		if e != nil {
			return "", err("ai_vision: " + e.Error())
		}
		return dataURLFor(data)
	case *Bytes:
		return dataURLFor(v.Value)
	default:
		return "", err("ai_vision: image must be a file path, an http(s) URL, or bytes")
	}
}

// dataURLFor content-sniffs raw image bytes and base64-encodes them into a
// data: URL. Sniffing (rather than trusting a file extension) works
// uniformly for both file-path and raw-*Bytes input and matches what the
// provider itself validates against (JPEG, PNG, GIF, WebP).
func dataURLFor(data []byte) (string, *Error) {
	mime := http.DetectContentType(data)
	if !acceptedImageMIMEs[mime] {
		return "", err("ai_vision: unsupported image type '" + mime + "' (jpeg, png, gif, webp only)")
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

func bAiVision(args ...Object) Object {
	if len(args) < 2 || len(args) > 3 {
		return err("ai_vision expects 2-3 arguments (image, prompt, [max_tokens])")
	}
	prompt, ok := args[1].(*String)
	if !ok {
		return err("ai_vision: prompt must be a string")
	}
	maxTokens := 0
	if len(args) == 3 {
		n, ok := ToInt(args[2])
		if !ok {
			return err("ai_vision: max_tokens must be a number")
		}
		maxTokens = int(n)
	}

	// Same two-branch gate as ai_chat/ai_swarm: profile.CanAI() under a
	// registered profile, CLI --sandbox flag otherwise. ai.VisionChat's own
	// gateEgress call is the authoritative backstop (round-5 architecture);
	// this early check just avoids resolving/reading the image for a call
	// that's going to be rejected anyway.
	if ActiveProfile.Load().Name != "none" {
		if canErr := ActiveProfile.Load().CanAI(); canErr != nil {
			return err(canErr.Error())
		}
	} else if Sandbox.Enabled && !Sandbox.AllowAI {
		return sandboxBlock("ai_vision (AI calls)")
	}

	imageURL, rerr := resolveImageURL(args[0])
	if rerr != nil {
		return rerr
	}

	content, visErr := ai.VisionChat(prompt.Value, imageURL, maxTokens)
	if visErr != nil {
		return err("ai_vision: " + visErr.Error())
	}
	return &String{Value: content}
}
