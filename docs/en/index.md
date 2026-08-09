# Pipe Documentation — English

Welcome to the complete documentation for the **Pipe** scripting language.

## Where should I start?

| I want to… | Start here |
|------------|-------------|
| 🚀 **Try it in 5 minutes** | → [Getting Started](01-getting-started.md) — install, hello world, first pipeline |
| 🐍 **I know Python/Bash/JS** | → [Language Tour](02-language-tour.md) — key differences in 10 minutes |
| ⚙️ **Understand the VM** | → [Architecture](14-architecture.md) — lexer → parser → bytecode VM |
| 🤖 **Use AI features** | → [AI Builtins](19-ai-builtins.md) — 27 AI operations as language primitives |

## Table of Contents

1. [Getting Started](01-getting-started.md) — Installation, Hello World, 5-minute quickstart
2. [Language Tour](02-language-tour.md) — Comments, indentation, variables, function calls, all keywords
3. [Types and Expressions](03-types-and-expressions.md) — Data types, literals, operators, precedence
4. [Control Flow](04-control-flow.md) — if/else, match, while, break, continue, for-in, return, defer
5. [Functions and Closures](05-functions-and-closures.md) — fn definition, parameters, anonymous fns, closures, TCO
6. [Pipelines](06-pipelines.md) — The core feature: vertical/horizontal pipeline, \_ placeholder
7. [Data Structures](07-data-structures.md) — Lists, slicing, maps, structs, dot access, higher-order functions
8. [Error Handling](08-error-handling.md) — try/catch, stack traces, Result type
9. [Modules and Imports](09-modules-and-imports.md) — import, export, namespaces, PIPE\_PATH
10. [Builtin Reference](10-builtin-reference.md) — All 198 built-in functions
11. [Tooling](11-tooling.md) — CLI flags, REPL, formatter, test runner, build
12. [Execution Models](12-execution-models.md) — Tree-walker vs Bytecode VM
13. [Bytecode VM](13-bytecode-vm.md) — 42 opcodes, stack machine, symbol table
14. [Architecture](14-architecture.md) — Lexer, parser, AST, compiler, VM internals
15. [VSCode Extension](15-vscode-extension.md) — Installation, syntax highlighting
16. [Cookbook](16-cookbook.md) — 20+ practical code examples
17. [Migration from Other Languages](17-migration-from.md) — Python, Lua, Bash, JavaScript
18. [Roadmap](18-roadmap.md) — The future of the language
19. [AI Builtins](19-ai-builtins.md) — AI functions for LLM integration
20. [GitHub Action](20-github-action.md) — Run Pipe in CI/CD pipelines
21. [Module Ecosystem](21-ecosystem.md) — Find, install, and contribute modules
22. [Sandbox Profiles](22-sandbox-profiles.md) — Declarative security profiles for AI agents and untrusted code
23. [X (Twitter) Module](23-x-module.md) — OAuth 2.0 + API v2 client as a pure-Pipe module *(in development, not yet in the registry)*
24. [Discord Module](24-discord-module.md) — Webhook + Bot-Token client as a pure-Pipe module *(in development, not yet in the registry)*
25. [MCP — Model Context Protocol](25-mcp.md) — Zero-dependency MCP Server + Client, JSON-RPC 2.0 over stdio *(in development, not yet in the registry)*
26. [SQLite Module](26-sqlite-module.md) — Pure-Pipe relational database + benchmark vs Python/Lua

## Appendix

- [Appendix A: Formal Grammar](appendix-a-grammar.md) — Complete EBNF grammar

## Quick Reference

```text
-- Comment
-- Variable
x: 42
-- Compound assignment
x: x + 1
-- Function
fn name a b: ...
-- Conditional
if cond: ... else: ...
-- Pattern matching
match x | 0 -> ... | _ ->
-- Loop
while cond: ...
-- for-in
for x in list: ...
-- Error handling
try: ... catch e: ...
-- Early return
return value
-- Deferred execution
defer expr
-- Module import
import "file.pipe"
-- Symbol export
export fn name
A: 0
B: 1
-- Enumeration: 2
C

value
    -- Vertical pipeline
        > function
    > output
```

## Operators

```
+ - * / % **      Arithmetic + power
+= -= *= /= %=     Compound assignment
== != < > <= >=   Comparison
! && ||            Logic (not, and, or)
++                 String concatenation
>                  Pipeline (vertical with indentation)
>>                 Parallel pipeline (background execution)
..                 Range (slicing)
```
