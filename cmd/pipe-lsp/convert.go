package main

import (
	"encoding/json"
	"fmt"

	"github.com/harry/pipe/pkg/analysis"
	"github.com/harry/pipe/pkg/ast"
)

// ---- LSP protocol types (subset) ----

type lspPosition struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type lspRange struct {
	Start lspPosition `json:"start"`
	End   lspPosition `json:"end"`
}

type lspLocation struct {
	URI   string   `json:"uri"`
	Range lspRange `json:"range"`
}

type lspTextEdit struct {
	Range   lspRange `json:"range"`
	NewText string   `json:"newText"`
}

type lspMarkupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type lspDiagnostic struct {
	Range    lspRange `json:"range"`
	Severity int      `json:"severity,omitempty"`
	Code     string   `json:"code,omitempty"`
	Source   string   `json:"source,omitempty"`
	Message  string   `json:"message"`
}

type lspCompletionItem struct {
	Label            string       `json:"label"`
	Kind             int          `json:"kind"`
	Detail           string       `json:"detail,omitempty"`
	Documentation    string       `json:"documentation,omitempty"`
	SortText         string       `json:"sortText,omitempty"`
	TextEdit         *lspTextEdit `json:"textEdit,omitempty"`
	InsertTextFormat int          `json:"insertTextFormat,omitempty"`
}

type lspCompletionList struct {
	IsIncomplete bool                `json:"isIncomplete"`
	Items        []lspCompletionItem `json:"items"`
}

type lspHover struct {
	Contents lspMarkupContent `json:"contents"`
	Range    *lspRange        `json:"range,omitempty"`
}

type lspParameterInformation struct {
	Label string `json:"label"`
}

type lspSignatureInformation struct {
	Label         string                    `json:"label"`
	Documentation *lspMarkupContent         `json:"documentation,omitempty"`
	Parameters    []lspParameterInformation `json:"parameters,omitempty"`
}

type lspSignatureHelp struct {
	Signatures      []lspSignatureInformation `json:"signatures"`
	ActiveSignature int                       `json:"activeSignature"`
	ActiveParameter int                       `json:"activeParameter"`
}

type lspWorkspaceEdit struct {
	Changes map[string][]lspTextEdit `json:"changes"`
}

// ---- conversions: Pipe uses 1-based line/col, LSP 0-based ----

func lspStart(p ast.Position) lspPosition {
	return lspPosition{Line: p.Line - 1, Character: p.Col - 1}
}

// lspEnd converts an inclusive Pipe position to an exclusive LSP position.
func lspEnd(p ast.Position) lspPosition {
	return lspPosition{Line: p.Line - 1, Character: p.Col}
}

func lspRangeOf(start, end ast.Position) lspRange {
	return lspRange{Start: lspStart(start), End: lspEnd(end)}
}

func lspRangeOfAnalysis(r analysis.Range) lspRange {
	return lspRangeOf(r.Start, r.End)
}

func lspLocationOf(loc analysis.Location, uri string) lspLocation {
	return lspLocation{URI: uri, Range: lspRangeOf(loc.Start, loc.End)}
}

func marshal(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		return []byte(fmt.Sprintf(`{"jsonrpc":"2.0","error":{"code":-32603,"message":%q}}`, err.Error()))
	}
	return data
}
