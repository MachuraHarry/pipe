// CodeMirror 6 language support for Pipe — a hand-written StreamLanguage
// (single-pass token scanner), not a Lezer grammar, so no build step is
// needed to compile it. Ports the regex/word-lists this page's editor used
// before the CodeMirror move (keywords, strings, numbers, comments,
// operators) into CM6's token(stream, state) API.
import { StreamLanguage } from "https://esm.sh/codemirror@6.0.1";
import { HighlightStyle } from "https://esm.sh/codemirror@6.0.1";
import { tags } from "https://esm.sh/@lezer/highlight@1.2.1";

var KEYWORDS = /^(?:if|else|elif|elseif|while|for|in|fn|return|and|or|not|true|false|none|null|pipe|end|from|import|as|sandbox_profile|include|expand|stage|assume|exec|spawn|await|map|out|def|let|var|const|func|fun|λ|lambda)(?![\w])/;
var OPERATORS = /^(?:->|<-|=>|==|!=|<=|>=|\+|-|\*|\/|%|=|<|>|&&|\|\|)/;
var IDENT = /^[a-zA-Z_][a-zA-Z0-9_]*/;

var pipeStreamParser = {
  token: function (stream, state) {
    if (state.inBlockComment) {
      // Pipe has no block comments today, kept as a harmless no-op hook
      state.inBlockComment = false;
    }
    if (stream.match(/^--[^\n]*/)) return "comment";
    if (stream.match(/^"(?:[^"\\]|\\.)*"/)) return "string";
    if (stream.match(/^'(?:[^'\\]|\\.)*'/)) return "string";
    if (stream.match(/^\d+(?:\.\d+)?/)) return "number";
    var m = stream.match(KEYWORDS);
    if (m) return "keyword";
    if (stream.match(OPERATORS)) return "operator";
    var id = stream.match(IDENT);
    if (id) {
      // A following '(' (skipping whitespace) marks this as a call target,
      // matching the old regex-overlay's pipe-fn classification.
      var rest = stream.string.slice(stream.pos);
      if (/^\s*\(/.test(rest)) return "function";
      return "variableName";
    }
    stream.next();
    return null;
  }
};

export var pipeLanguage = StreamLanguage.define(pipeStreamParser);

// Colors mirror the old .token.pipe-* CSS classes exactly.
export var pipeHighlightStyle = HighlightStyle.define([
  { tag: tags.keyword, color: "#c084fc" },
  { tag: tags.operator, color: "#f97316" },
  { tag: tags.number, color: "#38bdf8" },
  { tag: tags.string, color: "#34d399" },
  { tag: tags.function(tags.variableName), color: "#60a5fa" },
  { tag: tags.variableName, color: "#e4e5f1" },
  { tag: tags.comment, color: "#64748b", fontStyle: "italic" }
]);
