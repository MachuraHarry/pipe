package main

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/harry/pipe/pkg/analysis"
)

const testURI = "file:///test.pipe"

func newTestServer() (*Server, <-chan []byte) {
	out := make(chan []byte, 16)
	return NewServer(out), out
}

func openDoc(t *testing.T, s *Server, text string) {
	t.Helper()
	params := `{"textDocument":{"uri":"` + testURI + `","languageId":"pipe","version":1,"text":` + mustJSON(text) + `}}`
	if err := s.didOpen(json.RawMessage(params)); err != nil {
		t.Fatalf("didOpen: %v", err)
	}
}

func mustJSON(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func posParams(uri string, line, char int) string {
	return `{"textDocument":{"uri":"` + uri + `"},"position":{"line":` + itoa(line) + `,"character":` + itoa(char) + `}}`
}

func itoa(n int) string {
	return strconv.Itoa(n)
}

func TestInitializeCapabilities(t *testing.T) {
	s, _ := newTestServer()
	res, err := s.initialize(nil)
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	caps, ok := res.(map[string]any)["capabilities"].(map[string]any)
	if !ok {
		t.Fatal("capabilities missing")
	}
	for _, k := range []string{"textDocumentSync", "hoverProvider", "completionProvider", "signatureHelpProvider", "definitionProvider", "referencesProvider", "renameProvider", "documentFormattingProvider"} {
		if _, ok := caps[k]; !ok {
			t.Errorf("capability %q missing", k)
		}
	}
}

func TestDidOpenPublishesDiagnostics(t *testing.T) {
	s, out := newTestServer()
	openDoc(t, s, "print missing_var\n")
	select {
	case msg := <-out:
		var notif rpcMessage
		if err := json.Unmarshal(msg, &notif); err != nil {
			t.Fatalf("bad notification: %v", err)
		}
		if notif.Method != "textDocument/publishDiagnostics" {
			t.Fatalf("method = %q", notif.Method)
		}
		var params struct {
			Diagnostics []lspDiagnostic `json:"diagnostics"`
		}
		_ = json.Unmarshal(notif.Params, &params)
		found := false
		for _, d := range params.Diagnostics {
			if d.Code == "E001" && strings.Contains(d.Message, "missing_var") {
				found = true
			}
		}
		if !found {
			t.Errorf("E001 for missing_var not published: %+v", params.Diagnostics)
		}
	case <-time.After(time.Second):
		t.Fatal("no diagnostics published")
	}
}

func TestCompletionHandler(t *testing.T) {
	s, _ := newTestServer()
	openDoc(t, s, "fn greet name\n    print name\n\nprint gr")
	res, err := s.completion(json.RawMessage(posParams(testURI, 3, 8)))
	if err != nil {
		t.Fatalf("completion: %v", err)
	}
	list := res.(lspCompletionList)
	names := map[string]bool{}
	for _, it := range list.Items {
		names[it.Label] = true
	}
	if !names["greet"] {
		t.Errorf("completion missing greet; got %v", names)
	}
	for _, it := range list.Items {
		if it.TextEdit == nil {
			t.Errorf("item %q missing textEdit", it.Label)
		}
	}
}

func TestHoverHandler(t *testing.T) {
	s, _ := newTestServer()
	openDoc(t, s, "print \"hi\"\n")
	res, err := s.hover(json.RawMessage(posParams(testURI, 0, 0)))
	if err != nil {
		t.Fatalf("hover: %v", err)
	}
	h, ok := res.(lspHover)
	if !ok {
		t.Fatalf("hover result type %T", res)
	}
	if !strings.Contains(h.Contents.Value, "print") {
		t.Errorf("hover content missing print: %q", h.Contents.Value)
	}
}

func TestDefinitionHandler(t *testing.T) {
	s, _ := newTestServer()
	openDoc(t, s, "fn greet name\n    print name\n\ngreet \"x\"\n")
	res, err := s.definition(json.RawMessage(posParams(testURI, 3, 0)))
	if err != nil {
		t.Fatalf("definition: %v", err)
	}
	loc, ok := res.(lspLocation)
	if !ok {
		t.Fatalf("definition result type %T", res)
	}
	if loc.Range.Start.Line != 0 || loc.Range.Start.Character != 3 {
		t.Errorf("definition range = %+v, want line 0 char 3", loc.Range.Start)
	}
}

func TestReferencesHandlerExcludesDeclaration(t *testing.T) {
	s, _ := newTestServer()
	openDoc(t, s, "fn greet name\n    print name\n    greet name\n")
	params, err := json.Marshal(map[string]any{
		"textDocument": map[string]any{"uri": testURI},
		"position":     map[string]any{"line": 2, "character": 4},
		"context":      map[string]any{"includeDeclaration": false},
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.references(params)
	if err != nil {
		t.Fatalf("references: %v", err)
	}
	locs, ok := res.([]lspLocation)
	if !ok {
		t.Fatalf("references result type %T", res)
	}
	if len(locs) != 1 {
		t.Fatalf("expected 1 reference, got %d", len(locs))
	}
}

func TestRenameHandler(t *testing.T) {
	s, _ := newTestServer()
	openDoc(t, s, "fn greet name\n    print name\n    greet name\n")
	params, err := json.Marshal(map[string]any{
		"textDocument": map[string]any{"uri": testURI},
		"position":     map[string]any{"line": 2, "character": 4},
		"newName":      "hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.rename(params)
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	we, ok := res.(lspWorkspaceEdit)
	if !ok {
		t.Fatalf("rename result type %T", res)
	}
	edits := we.Changes[testURI]
	if len(edits) != 2 {
		t.Fatalf("expected 2 edits, got %d", len(edits))
	}
}

func TestSemanticTokensHandler(t *testing.T) {
	s, _ := newTestServer()
	openDoc(t, s, "fn foo x\n    print \"hi\"\n")
	res, err := s.semanticTokens(json.RawMessage(`{"textDocument":{"uri":"` + testURI + `"}}`))
	if err != nil {
		t.Fatalf("semanticTokens: %v", err)
	}
	m, ok := res.(map[string]any)
	if !ok {
		t.Fatalf("semanticTokens result type %T", res)
	}
	data, ok := m["data"].([]int)
	if !ok {
		t.Fatalf("semanticTokens data type %T", m["data"])
	}
	if len(data) == 0 || len(data)%5 != 0 {
		t.Fatalf("semantic token data must be non-empty and a multiple of 5, got %d", len(data))
	}
	types := map[int]bool{}
	for i := 3; i < len(data); i += 5 {
		types[data[i]] = true
	}
	if !types[analysis.SemKeyword] {
		t.Error("no keyword token emitted")
	}
	if !types[analysis.SemFunction] {
		t.Error("no function token emitted")
	}
	if !types[analysis.SemString] {
		t.Error("no string token emitted")
	}
}

func TestFormattingHandler(t *testing.T) {
	s, _ := newTestServer()
	openDoc(t, s, "fn foo\nprint 1\n")
	res, err := s.formatting(json.RawMessage(`{"textDocument":{"uri":"` + testURI + `"}}`))
	if err != nil {
		t.Fatalf("formatting: %v", err)
	}
	edits, ok := res.([]lspTextEdit)
	if !ok {
		t.Fatalf("formatting result type %T", res)
	}
	if len(edits) != 1 {
		t.Fatalf("expected 1 edit, got %d", len(edits))
	}
}
