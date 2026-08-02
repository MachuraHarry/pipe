package analysis

import (
	"sort"
	"strings"
)

// LSP CompletionItemKind values (subset used here).
const (
	LSPKindFunction   = 3
	LSPKindVariable   = 6
	LSPKindModule     = 9
	LSPKindEnum       = 13
	LSPKindKeyword    = 14
	LSPKindSnippet    = 15
	LSPKindEnumMember = 20
)

const (
	InsertPlain   = 1
	InsertSnippet = 2
)

// CompletionItem mirrors the subset of an LSP completion item the server emits.
type CompletionItem struct {
	Label            string
	Kind             int
	Detail           string
	Documentation    string
	InsertText       string
	InsertTextFormat int
	SortText         string
}

// CompletionWord returns the identifier being typed at the position together
// with the 1-based column of its first character (for the completion range).
func CompletionWord(source string, line, col int) (string, int) {
	return wordAt(source, line, col)
}

// Complete returns a sorted list of completion items for the position.
// analysis should be the analysis of the same source (or nil to skip symbols).
func (a *Analysis) Complete(source string, line, col int) []CompletionItem {
	prefix, _ := wordAt(source, line, col)

	items := make(map[string]CompletionItem)

	// User symbols visible in the active scope chain.
	if a != nil {
		for _, sym := range a.VisibleSymbolsAt(line, col) {
			items[sym.Name] = symbolCompletionItem(sym)
		}
	}

	// Builtins (user symbols override builtins of the same name).
	for _, b := range builtinDocs {
		items[b.Name] = builtinCompletionItem(b)
	}

	// Keywords.
	for _, kw := range Keywords {
		items[kw] = keywordCompletionItem(kw)
	}

	// Snippets.
	for _, sn := range Snippets {
		items[sn.Label] = snippetCompletionItem(sn)
	}

	// Filter by prefix and sort.
	var out []CompletionItem
	for _, it := range items {
		if strings.HasPrefix(strings.ToLower(it.Label), strings.ToLower(prefix)) {
			out = append(out, it)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].SortText != out[j].SortText {
			return out[i].SortText < out[j].SortText
		}
		return out[i].Label < out[j].Label
	})

	// If nothing matched the prefix, return an empty slice (caller decides
	// whether to show the fallback list).
	return out
}

func symbolCompletionItem(sym *Symbol) CompletionItem {
	kind := LSPKindVariable
	switch sym.Kind {
	case KindFunction:
		kind = LSPKindFunction
	case KindEnum:
		kind = LSPKindEnum
	case KindEnumMember:
		kind = LSPKindEnumMember
	case KindModule:
		kind = LSPKindModule
	}
	detail := sym.Kind.String()
	if sym.Kind == KindFunction {
		detail = "fn " + sym.Name
	}
	if sym.Builtin != nil {
		return builtinCompletionItem(*sym.Builtin)
	}
	return CompletionItem{
		Label:         sym.Name,
		Kind:          kind,
		Detail:        detail,
		Documentation: sym.Doc,
		InsertText:    sym.Name,
		SortText:      "a",
	}
}

func builtinCompletionItem(b BuiltinDoc) CompletionItem {
	detail := b.Signature
	if b.ReturnType != "" {
		detail += " -> " + b.ReturnType
	}
	doc := b.Description
	if doc == "" {
		doc = b.Signature
	}
	return CompletionItem{
		Label:         b.Name,
		Kind:          LSPKindFunction,
		Detail:        detail,
		Documentation: doc,
		InsertText:    b.Name,
		SortText:      "b",
	}
}

func keywordCompletionItem(kw string) CompletionItem {
	return CompletionItem{
		Label:      kw,
		Kind:       LSPKindKeyword,
		Detail:     "keyword",
		InsertText: kw,
		SortText:   "c",
	}
}

func snippetCompletionItem(sn Snippet) CompletionItem {
	return CompletionItem{
		Label:            sn.Label,
		Kind:             LSPKindSnippet,
		Detail:           sn.Detail,
		InsertText:       sn.InsertText,
		InsertTextFormat: InsertSnippet,
		SortText:         "z",
	}
}

// FilterCompletions applies an additional case-insensitive prefix filter.
// Useful when the server wants an exact word boundary match.
func FilterCompletions(items []CompletionItem, prefix string) []CompletionItem {
	if prefix == "" {
		return items
	}
	var out []CompletionItem
	for _, it := range items {
		if strings.HasPrefix(strings.ToLower(it.Label), strings.ToLower(prefix)) {
			out = append(out, it)
		}
	}
	return out
}
