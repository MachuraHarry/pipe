// Pipe Prism grammar — shared by blog pages.
// Mirrors the Pipe language tokens (pkg/lexer/token.go) and AI/MCP builtins.
(function () {
  if (typeof Prism === 'undefined' || Prism.languages && Prism.languages.pipe) return;
  Prism.languages.pipe = {
    comment: /--.*/,
    string: {
      pattern: /"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'/,
      greedy: true
    },
    keyword: /\b(?:fn|match|if|else|elif|while|for|in|break|continue|import|export|enum|defer|return|try|catch|try_ai|not|test|struct|and|or)\b/,
    boolean: /\b(?:true|false|nil)\b/,
    number: /\b\d+(?:\.\d+)?\b/,
    builtin: /\b(?:ai_provider|ai_batch|ai_tool|ask|classify|extract|embed|embed_batch|nearest|agent|agent_ask|sandbox|sandbox_profile|mcp_server|mcp_use_stdio|mcp_serve_stdio|http_get|http_post|to_num|to_str|at|len|spawn|now|print|read_file|write_file|exec|save|split|join|upper|lower|filter|map|sort|contains|append|sleep|load|title|en|de)\b/,
    operator: />>|->|\+\+|\*\*|\.\.|->>|\+=|-=|\*=|\/=|%=|==|!=|<=|>=|&&|\|\||[+\-*\/%<>=!:]/,
    punctuation: /[{}[\]();,.]/,
    'function': {
      pattern: /\b[a-zA-Z_]\w*(?=\s*\()/,
      alias: 'function'
    },
    variable: /\b[a-zA-Z_]\w*\b/
  };
})();
