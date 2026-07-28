# Pipe Documentation — English

Welcome to the complete documentation for the **Pipe** scripting language.

## Table of Contents

1. [Getting Started](01-getting-started.md) — Installation, Hello World, 5-minute quickstart
2. [Language Tour](02-language-tour.md) — Comments, indentation, variables, function calls, all keywords
3. [Types and Expressions](03-types-and-expressions.md) — Data types, literals, operators, precedence
4. [Control Flow](04-control-flow.md) — if/else, match, while, break, continue, for-in, return, defer
5. [Functions and Closures](05-functions-and-closures.md) — fn definition, parameters, anonymous fns, closures, TCO
6. [Pipelines](06-pipelines.md) — The core feature: vertical/horizontal pipeline, \_ placeholder
7. [Data Structures](07-data-structures.md) — Lists, slicing, maps, dot access, higher-order functions
8. [Error Handling](08-error-handling.md) — try/catch, stack traces, Result type
9. [Modules and Imports](09-modules-and-imports.md) — import, export, namespaces, PIPE\_PATH
10. [Builtin Reference](10-builtin-reference.md) — All 80+ built-in functions
11. [Tooling](11-tooling.md) — CLI flags, REPL, formatter, test runner, build
12. [Execution Models](12-execution-models.md) — Tree-walker vs Bytecode VM
13. [Bytecode VM](13-bytecode-vm.md) — 47 opcodes, stack machine, symbol table
14. [Architecture](14-architecture.md) — Lexer, parser, AST, compiler, VM internals
15. [VSCode Extension](15-vscode-extension.md) — Installation, syntax highlighting
16. [Cookbook](16-cookbook.md) — 20+ practical code examples
17. [Migration from Other Languages](17-migration-from.md) — Python, Lua, Bash, JavaScript
18. [Roadmap](18-roadmap.md) — The future of the language
19. [AI Builtins](19-ai-builtins.md) — AI functions for LLM integration

## Appendix

- [Appendix A: Formal Grammar](appendix-a-grammar.md) — Complete EBNF grammar

## Quick Reference

```pipe
-- Comment
x: 42                         -- Variable
x += 1                        -- Compound assignment
fn name a b: ...              -- Function
if cond: ... else: ...        -- Conditional
match x | 0 -> ... | _ ->     -- Pattern matching
while cond: ...               -- Loop
for x in list: ...            -- for-in
try: ... catch e: ...         -- Error handling
return value                  -- Early return
defer expr                    -- Deferred execution
import "file.pipe"            -- Module import
export fn name                -- Symbol export
enum Name: A, B, C            -- Enumeration

value
    > function                 -- Vertical pipeline
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
..                 Range (slicing)
```
