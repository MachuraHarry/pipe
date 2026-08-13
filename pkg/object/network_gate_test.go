package object

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func withSandboxFlags(enabled, allowNet bool) func() {
	prevEnabled, prevAllowNet := Sandbox.Enabled, Sandbox.AllowNet
	prevProfile := ActiveProfile.Load()
	Sandbox.Enabled, Sandbox.AllowNet = enabled, allowNet
	ActiveProfile.Store(profileRegistry["none"])
	return func() {
		Sandbox.Enabled, Sandbox.AllowNet = prevEnabled, prevAllowNet
		ActiveProfile.Store(prevProfile)
	}
}

func assertSandboxBlocked(t *testing.T, feature string, result Object) {
	t.Helper()
	if result == nil || result.Type() != ERROR {
		t.Fatalf("%s: expected a sandbox error, got %v", feature, result)
	}
	msg := result.(*Error).Message
	if !strings.Contains(msg, "SANDBOX") {
		t.Fatalf("%s: expected a SANDBOX message, got %q", feature, msg)
	}
}

// The CLI --sandbox flag keeps ActiveProfile at "none" and governs network via
// Sandbox.AllowNet. Every network-capable builtin must honor that flag, not
// just the registered-profile path.
func TestNetworkBuiltinsBlockedBySandboxFlag(t *testing.T) {
	defer withSandboxFlags(true, false)()

	cases := []struct {
		name string
		call func() Object
	}{
		{"http_get", func() Object {
			return bHttpGet(&String{Value: "http://127.0.0.1:1/"})
		}},
		{"http_post", func() Object {
			return bHttpPost(&String{Value: "http://127.0.0.1:1/"}, &String{Value: "{}"})
		}},
		{"http_request", func() Object {
			return bHttpRequest(&String{Value: "GET"}, &String{Value: "http://127.0.0.1:1/"})
		}},
		{"http_stream_open", func() Object {
			return bHttpStreamOpen(&String{Value: "http://127.0.0.1:1/"})
		}},
		{"http_server", func() Object {
			return bHttpServer(&String{Value: "127.0.0.1:1"}, &String{Value: ""})
		}},
		{"tcp_connect", func() Object {
			return bTcpConnect(&String{Value: "127.0.0.1"}, &Integer{Value: 1})
		}},
		{"tcp_listen", func() Object {
			return bTcpListen(&String{Value: "127.0.0.1"}, &Integer{Value: 1})
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertSandboxBlocked(t, tc.name, tc.call())
		})
	}
}

func TestNetworkBuiltinsAllowedBySandboxFlag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	defer withSandboxFlags(true, true)()

	get := bHttpGet(&String{Value: srv.URL})
	if get.Type() == ERROR {
		t.Fatalf("http_get should be allowed with AllowNet=true, got %v", get)
	}
	if status := get.(*Map).Pairs["status"].(*Integer).Value; status != 200 {
		t.Fatalf("http_get status = %d, want 200", status)
	}

	post := bHttpPost(&String{Value: srv.URL}, &String{Value: `{"a":1}`})
	if post.Type() == ERROR {
		t.Fatalf("http_post should be allowed with AllowNet=true, got %v", post)
	}
	if status := post.(*Map).Pairs["status"].(*Integer).Value; status != 200 {
		t.Fatalf("http_post status = %d, want 200", status)
	}
}

func TestNetworkBuiltinsBlockedByProfile(t *testing.T) {
	defer withProfile(testProfile("net-gate-block", FSFull, false, false, false, nil))()

	for name, call := range map[string]func() Object{
		"http_post": func() Object {
			return bHttpPost(&String{Value: "http://127.0.0.1:1/"}, &String{Value: "{}"})
		},
		"tcp_connect": func() Object {
			return bTcpConnect(&String{Value: "127.0.0.1"}, &Integer{Value: 1})
		},
		"tcp_listen": func() Object {
			return bTcpListen(&String{Value: "127.0.0.1"}, &Integer{Value: 1})
		},
	} {
		t.Run(name, func(t *testing.T) {
			assertSandboxBlocked(t, name, call())
		})
	}
}

func TestNetworkBuiltinsAllowedByProfile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	defer withProfile(testProfile("net-gate-allow", FSFull, true, false, false, nil))()

	post := bHttpPost(&String{Value: srv.URL}, &String{Value: `{"a":1}`})
	if post.Type() == ERROR {
		t.Fatalf("http_post should be allowed under network:true profile, got %v", post)
	}
	if status := post.(*Map).Pairs["status"].(*Integer).Value; status != 200 {
		t.Fatalf("http_post status = %d, want 200", status)
	}
}
