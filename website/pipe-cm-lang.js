// CodeMirror 6 language support for Pipe — a hand-written StreamLanguage
// (single-pass token scanner), not a Lezer grammar, so no build step is
// needed to compile it. Ports the regex/word-lists this page's editor used
// before the CodeMirror move (keywords, strings, numbers, comments,
// operators) into CM6's token(stream, state) API.
import { StreamLanguage } from "https://esm.sh/codemirror@6.0.1";
import { HighlightStyle } from "https://esm.sh/codemirror@6.0.1";
// Pinned to the EXACT version codemirror@6.0.1 resolves internally for
// @lezer/highlight (verified by fetching https://esm.sh/@lezer/highlight@^1.0.0
// and reading its resolved x-esm-path — currently 1.2.3). This is not
// optional: esm.sh serves each distinct version string as a genuinely
// separate module, so a Tag built from a *different* pinned version here
// than the one CodeMirror's own highlighter uses internally is a foreign
// object as far as tag-identity matching is concerned — HighlightStyle
// rules built from such tags silently match nothing (no error, no console
// warning, just zero syntax highlighting). If codemirror's pinned version
// above is ever bumped, re-check this resolution and update to match.
import { tags } from "https://esm.sh/@lezer/highlight@1.2.3";

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
      // matching the old regex-overlay's pipe-fn classification. CM6's
      // StreamLanguage resolves a returned token string against
      // @lezer/highlight's `tags` object, splitting on "." for
      // base-tag+modifier composition (see @codemirror/language's
      // stream-parser.ts, createTokenType) — `tags.function` is a
      // *modifier*, not a standalone tag, so it MUST come after a base
      // tag (here "variableName"), never bare. Returning bare "function"
      // silently produces zero styling for every call target.
      var rest = stream.string.slice(stream.pos);
      if (/^\s*\(/.test(rest)) return "variableName.function";
      return "variableName";
    }
    stream.next();
    return null;
  }
};

export var pipeLanguage = StreamLanguage.define(pipeStreamParser);

// Colors reuse the sitewide CSS custom properties (see website/style.css)
// where a reasonable match exists, so the editor's palette matches the
// rest of pipe-lang.com instead of an unrelated one-off palette.
export var pipeHighlightStyle = HighlightStyle.define([
  { tag: tags.keyword, color: "var(--accent)" },
  { tag: tags.operator, color: "#f97316" },
  { tag: tags.number, color: "#38bdf8" },
  { tag: tags.string, color: "var(--green)" },
  { tag: tags.function(tags.variableName), color: "#60a5fa" },
  { tag: tags.variableName, color: "var(--fg)" },
  { tag: tags.comment, color: "var(--fg3)", fontStyle: "italic" }
]);
