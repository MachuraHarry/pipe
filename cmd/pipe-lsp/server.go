package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/MachuraHarry/pipe/pkg/analysis"
	"github.com/MachuraHarry/pipe/pkg/formatter"
)

// Document is the state the server keeps for one open file.
type Document struct {
	URI      string
	Version  int
	Text     string
	Analysis *analysis.Analysis
}

// Server implements the LSP request handlers.
type Server struct {
	mu   sync.Mutex
	docs map[string]*Document
	out  chan []byte
}

func NewServer(out chan []byte) *Server {
	return &Server{
		docs: make(map[string]*Document),
		out:  out,
	}
}

func (s *Server) publish(d *Document) {
	res := analysis.Lint(d.Text)
	diags := make([]lspDiagnostic, 0, len(res.Diagnostics))
	for _, dg := range res.Diagnostics {
		diags = append(diags, lspDiagnostic{
			Range:    lspRangeOfAnalysis(dg.Range),
			Severity: int(dg.Severity),
			Code:     dg.Code,
			Source:   "pipe",
			Message:  dg.Message,
		})
	}
	s.notify("textDocument/publishDiagnostics", map[string]any{
		"uri":         d.URI,
		"version":     d.Version,
		"diagnostics": diags,
	})
}

func (s *Server) notify(method string, params any) {
	msg := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	s.out <- data
}

func uriPath(uri string) string {
	u := strings.TrimPrefix(uri, "file://")
	if i := strings.IndexByte(u, '?'); i >= 0 {
		u = u[:i]
	}
	return u
}

// ---- notifications ----

func (s *Server) didOpen(params json.RawMessage) error {
	var p struct {
		TextDocument struct {
			URI     string `json:"uri"`
			Version int    `json:"version"`
			Text    string `json:"text"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return err
	}
	a, _ := analysis.Analyze(p.TextDocument.Text)
	s.mu.Lock()
	s.docs[p.TextDocument.URI] = &Document{
		URI:      p.TextDocument.URI,
		Version:  p.TextDocument.Version,
		Text:     p.TextDocument.Text,
		Analysis: a,
	}
	d := s.docs[p.TextDocument.URI]
	s.mu.Unlock()
	s.publish(d)
	return nil
}

func (s *Server) didChange(params json.RawMessage) error {
	var p struct {
		TextDocument struct {
			URI     string `json:"uri"`
			Version int    `json:"version"`
		} `json:"textDocument"`
		ContentChanges []struct {
			Text string `json:"text"`
		} `json:"contentChanges"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return err
	}
	s.mu.Lock()
	d, ok := s.docs[p.TextDocument.URI]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("document not open: %s", p.TextDocument.URI)
	}
	if len(p.ContentChanges) > 0 {
		d.Text = p.ContentChanges[len(p.ContentChanges)-1].Text
	}
	d.Version = p.TextDocument.Version
	d.Analysis, _ = analysis.Analyze(d.Text)
	s.mu.Unlock()
	s.publish(d)
	return nil
}

func (s *Server) didClose(params json.RawMessage) error {
	var p struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.docs, p.TextDocument.URI)
	s.mu.Unlock()
	return nil
}

func (s *Server) didSave(params json.RawMessage) error {
	var p struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Text string `json:"text,omitempty"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return err
	}
	s.mu.Lock()
	d, ok := s.docs[p.TextDocument.URI]
	if ok && p.Text != "" {
		d.Text = p.Text
		d.Analysis, _ = analysis.Analyze(d.Text)
	}
	s.mu.Unlock()
	if ok {
		s.publish(d)
	}
	return nil
}

// ---- requests ----

func (s *Server) initialize(params json.RawMessage) (any, error) {
	return map[string]any{
		"capabilities": map[string]any{
			"textDocumentSync": map[string]any{
				"openClose": true,
				"change":    1,
			},
			"hoverProvider": true,
			"completionProvider": map[string]any{
				"triggerCharacters": []string{".", ":", "_"},
			},
			"signatureHelpProvider": map[string]any{
				"triggerCharacters": []string{"(", ","},
			},
			"definitionProvider":         true,
			"referencesProvider":         true,
			"renameProvider":             map[string]any{"prepareProvider": false},
			"documentFormattingProvider": true,
			"semanticTokensProvider": map[string]any{
				"legend": map[string]any{
					"tokenTypes":     analysis.SemanticTokenTypes,
					"tokenModifiers": []string{},
				},
				"full": true,
			},
		},
		"serverInfo": map[string]any{
			"name":    "pipe-lsp",
			"version": "0.1.0",
		},
	}, nil
}

func (s *Server) docFor(uri string) *Document {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.docs[uri]
}

type positionParams struct {
	TextDocument struct {
		URI string `json:"uri"`
	} `json:"textDocument"`
	Position lspPosition `json:"position"`
}

func (s *Server) completion(params json.RawMessage) (any, error) {
	var p struct {
		positionParams
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	d := s.docFor(p.TextDocument.URI)
	if d == nil {
		return nil, fmt.Errorf("document not open: %s", p.TextDocument.URI)
	}
	line := p.Position.Line + 1
	col := p.Position.Character + 1

	var items []analysis.CompletionItem
	if d.Analysis != nil {
		items = d.Analysis.Complete(d.Text, line, col)
	} else {
		items = (&analysis.Analysis{}).Complete(d.Text, line, col)
	}

	_, wordCol := analysis.CompletionWord(d.Text, line, col)
	editRange := lspRange{
		Start: lspPosition{Line: p.Position.Line, Character: wordCol - 1},
		End:   p.Position,
	}

	out := make([]lspCompletionItem, 0, len(items))
	for _, it := range items {
		te := lspTextEdit{Range: editRange, NewText: it.InsertText}
		out = append(out, lspCompletionItem{
			Label:            it.Label,
			Kind:             it.Kind,
			Detail:           it.Detail,
			Documentation:    it.Documentation,
			SortText:         it.SortText,
			TextEdit:         &te,
			InsertTextFormat: it.InsertTextFormat,
		})
	}
	return lspCompletionList{IsIncomplete: false, Items: out}, nil
}

func (s *Server) hover(params json.RawMessage) (any, error) {
	var p struct{ positionParams }
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	d := s.docFor(p.TextDocument.URI)
	if d == nil || d.Analysis == nil {
		return nil, nil
	}
	line := p.Position.Line + 1
	col := p.Position.Character + 1
	info, ok := d.Analysis.Hover(d.Text, line, col)
	if !ok {
		return nil, nil
	}
	r := lspRangeOf(info.Start, info.End)
	return lspHover{
		Contents: lspMarkupContent{Kind: "markdown", Value: strings.Join(info.Contents, "\n")},
		Range:    &r,
	}, nil
}

func (s *Server) signatureHelp(params json.RawMessage) (any, error) {
	var p struct{ positionParams }
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	d := s.docFor(p.TextDocument.URI)
	if d == nil || d.Analysis == nil {
		return nil, nil
	}
	line := p.Position.Line + 1
	col := p.Position.Character + 1
	info, ok := d.Analysis.SignatureHelp(d.Text, line, col)
	if !ok {
		return nil, nil
	}
	var doc *lspMarkupContent
	if info.Doc != "" {
		doc = &lspMarkupContent{Kind: "markdown", Value: info.Doc}
	}
	paramInfos := make([]lspParameterInformation, 0, len(info.Params))
	for _, prm := range info.Params {
		paramInfos = append(paramInfos, lspParameterInformation{Label: prm.Name})
	}
	sig := lspSignatureInformation{Label: info.Label, Documentation: doc, Parameters: paramInfos}
	active := info.ActiveParam
	if active < 0 || active >= len(info.Params) {
		active = 0
	}
	return lspSignatureHelp{
		Signatures:      []lspSignatureInformation{sig},
		ActiveSignature: 0,
		ActiveParameter: active,
	}, nil
}

func (s *Server) definition(params json.RawMessage) (any, error) {
	var p struct{ positionParams }
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	d := s.docFor(p.TextDocument.URI)
	if d == nil || d.Analysis == nil {
		return nil, nil
	}
	line := p.Position.Line + 1
	col := p.Position.Character + 1
	loc, ok := d.Analysis.DefinitionAt(line, col)
	if !ok {
		return nil, nil
	}
	return lspLocationOf(loc, p.TextDocument.URI), nil
}

func (s *Server) references(params json.RawMessage) (any, error) {
	var p struct {
		positionParams
		Context struct {
			IncludeDeclaration bool `json:"includeDeclaration"`
		} `json:"context"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	d := s.docFor(p.TextDocument.URI)
	if d == nil || d.Analysis == nil {
		return nil, nil
	}
	line := p.Position.Line + 1
	col := p.Position.Character + 1
	locs := d.Analysis.ReferencesAt(line, col)
	if p.Context.IncludeDeclaration == false && len(locs) > 0 {
		locs = locs[1:] // drop the definition, which ReferencesAt returns first
	}
	out := make([]lspLocation, 0, len(locs))
	for _, l := range locs {
		out = append(out, lspLocationOf(l, p.TextDocument.URI))
	}
	return out, nil
}

func (s *Server) rename(params json.RawMessage) (any, error) {
	var p struct {
		positionParams
		NewName string `json:"newName"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	d := s.docFor(p.TextDocument.URI)
	if d == nil || d.Analysis == nil {
		return nil, nil
	}
	line := p.Position.Line + 1
	col := p.Position.Character + 1
	locs, err := d.Analysis.RenameAt(line, col, p.NewName)
	if err != nil {
		return nil, fmt.Errorf("invalid rename: %v", err)
	}
	edits := make([]lspTextEdit, 0, len(locs))
	for _, l := range locs {
		edits = append(edits, lspTextEdit{
			Range:   lspRangeOf(l.Start, l.End),
			NewText: p.NewName,
		})
	}
	return lspWorkspaceEdit{Changes: map[string][]lspTextEdit{p.TextDocument.URI: edits}}, nil
}

// semanticTokens handles textDocument/semanticTokens/full. The LSP encodes
// tokens as a flat int array using relative line/character deltas.
func (s *Server) semanticTokens(params json.RawMessage) (any, error) {
	var p struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	d := s.docFor(p.TextDocument.URI)
	if d == nil {
		return nil, nil
	}
	toks := analysis.SemanticTokens(d.Text, d.Analysis)

	data := make([]int, 0, len(toks)*5)
	prevLine, prevChar := 0, 0
	for _, tk := range toks {
		lspLine := tk.Line - 1
		lspChar := tk.Col - 1
		deltaLine := lspLine - prevLine
		deltaChar := lspChar
		if deltaLine == 0 {
			deltaChar = lspChar - prevChar
		}
		data = append(data, deltaLine, deltaChar, tk.Length, tk.Type, 0)
		prevLine = lspLine
		prevChar = lspChar
	}
	return map[string]any{"data": data}, nil
}

func (s *Server) formatting(params json.RawMessage) (any, error) {
	var p struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	d := s.docFor(p.TextDocument.URI)
	if d == nil {
		return nil, nil
	}
	formatted := formatter.FormatSource(d.Text)
	if formatted == d.Text {
		return []lspTextEdit{}, nil
	}
	// Replace the whole document with a range covering every line.
	lines := strings.Count(d.Text, "\n") + 1
	full := lspRange{
		Start: lspPosition{Line: 0, Character: 0},
		End:   lspPosition{Line: lines, Character: 0},
	}
	return []lspTextEdit{{Range: full, NewText: formatted}}, nil
}

// sortCompletions is kept for callers that want a deterministic order.
func sortCompletions(items []lspCompletionItem) {
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Label < items[j].Label
	})
}
