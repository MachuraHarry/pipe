# Appendix A: Formal Grammar

This appendix defines the complete formal grammar of the Pipe programming language in Extended Backus-Naur Form (EBNF). The grammar covers all lexical and syntactic constructs through version 0.5.1.

## Lexical Tokens

### Comments

```ebnf
comment = "--" , { ? any character except newline ? } , ? end of line ? ;
```

### Literals

```ebnf
integer     = digit , { digit } ;
float       = digit , { digit } , "." , digit , { digit } ;
string      = '"' , { char | escape } , '"'
            | '`' , { ? any character except backtick ? } , '`' ;
boolean     = "true" | "false" ;
nil         = "nil" ;
digit       = "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" ;
```

### Identifiers

```ebnf
identifier  = ( letter | "_" ) , { letter | digit | "_" } ;
letter      = "a" | ... | "z" | "A" | ... | "Z" ;
```

### Escape Sequences

```ebnf
escape      = "\" , ( "n" | "t" | "r" | "\" | '"' | "0" ) ;
char        = ? any printable character ? ;
```

### Character Classes

```ebnf
whitespace  = " " | "\t" | "\r" ;
newline     = "\n" ;
```

## Token Types

The lexer produces the following token stream:

```ebnf
Token       = ILLEGAL | EOF
            | IDENT | INT | FLOAT | STRING
            | ASSIGN | PLUS | MINUS | STAR | SLASH | PERCENT
            | EQ | NOT_EQ | LT | GT | LTE | GTE
            | CONCAT | BANG | AND | OR
            | PLUSEQ | MINUSEQ | STAREQ | SLASHEQ | PERCENTEQ
            | POWER | DOTDOT
            | PIPE | ARROW | ARROW2 | MATCH
            | LPAREN | RPAREN | LBRACKET | RBRACKET
            | LBRACE | RBRACE | COMMA | DOT | COLON
            | NEWLINE | INDENT | DEDENT
            | FN | MATCHKW | IF | ELSE | WHILE | FOR
            | BREAK | CONTINUE | IMPORT | EXPORT | ENUM
            | DEFER | RETURN | TRY | CATCH
            | TRUE | FALSE | NIL ;
```

## INDENT/DEDENT Rules

Pipe uses significant indentation to define block structure. The lexer maintains an indentation stack:

1. At the beginning of each logical line (after any leading whitespace), the indentation level is measured as the number of space characters. Tabs are treated as 4 spaces.
2. Empty lines and lines containing only a comment (`-- ...`) are ignored and produce no tokens.
3. If the indentation level exceeds the current top of the indent stack, an `INDENT` token is emitted and the new level is pushed onto the stack.
4. If the indentation level is less than the top of the stack, one or more `DEDENT` tokens are emitted (popping levels from the stack) until the stack top matches the new indentation level. If no stack level matches, an `ILLEGAL` token is emitted.
5. When inside brackets — `()`, `[]`, or `{}` — INDENT/DEDENT tracking is suspended. Content within brackets is parsed as a single logical line regardless of line breaks.
6. At end-of-file, sufficient `DEDENT` tokens are emitted to bring the indent stack back to `[0]`.

```
IndentStack: initially [0]

At each line start (not inside brackets):
  indent = count_spaces(line)
  if indent > stack.top():
    push(indent) -> emit INDENT
  elif indent < stack.top():
    while stack.top() > indent:
      pop() -> emit DEDENT
    if stack.top() != indent:
      emit ILLEGAL
  else:
    same level, scan normally

At EOF:
  while stack.size() > 1:
    pop() -> emit DEDENT
  emit EOF
```

## Keywords

```ebnf
keyword     = "fn" | "match" | "if" | "else" | "while" | "for"
            | "break" | "continue" | "import" | "export" | "enum"
            | "defer" | "return" | "try" | "catch"
            | "true" | "false" | "nil" ;
```

These are reserved words and cannot be used as identifiers.

## Built-in Types

| Type | Description |
|------|-------------|
| `INTEGER` | 64-bit signed integer |
| `FLOAT` | 64-bit floating-point number |
| `STRING` | UTF-8 encoded text string |
| `BOOLEAN` | `true` or `false` |
| `NIL` | The nil/null value |
| `LIST` | Ordered collection of values |
| `MAP` | Key-value collection (string keys) |
| `FUNCTION` | User-defined function (tree-walker) |
| `COMPILED_FUNCTION` | User-defined function (VM) |
| `CLOSURE` | Closure over a compiled function |
| `ERROR` | Runtime error value |
| `RESULT` | `Ok(val)` or `Err(msg)` |
| `TCP_CONN` | TCP client connection handle |
| `TCP_LISTENER` | TCP server listener handle |

## Full Grammar

### Program

```ebnf
program     = { statement } ;
```

### Statements

```ebnf
statement   = function_def
            | variable_def
            | enum_def
            | export_statement
            | import_statement
            | defer_statement
            | return_statement
            | break_statement
            | continue_statement
            | expression_statement ;
```

#### Function Definition

```ebnf
function_def = "fn" , identifier , { identifier } , block ;
```

A function definition consists of the `fn` keyword, the function name, zero or more parameter identifiers, and a block body. If the block body is omitted (inline syntax), the next expression becomes the function body.

#### Variable Definition

```ebnf
variable_def = identifier , ":" , expression ;
```

Variables are declared with a colon. The colon may be followed by a newline and an indented expression for multi-line initializers.

#### Compound Assignment

```ebnf
compound_assign = identifier , compound_op , expression ;
compound_op     = "+=" | "-=" | "*=" | "/=" | "%=" ;
```

Compound assignment desugars to: `x += expr` → `x: x + expr`.

#### Enum Definition

```ebnf
enum_def    = "enum" , identifier , ":" , identifier , { "," , identifier } ;
```

#### Export Statement

```ebnf
export_statement = "export" , function_def ;
```

#### Import Statement

```ebnf
import_statement = "import" , string , [ "as" , identifier ] ;
```

#### Defer Statement

```ebnf
defer_statement = "defer" , expression ;
```

#### Return Statement

```ebnf
return_statement = "return" , expression ;
```

#### Break Statement

```ebnf
break_statement = "break" ;
```

#### Continue Statement

```ebnf
continue_statement = "continue" ;
```

#### Expression Statement

```ebnf
expression_statement = expression , { vertical_pipeline } ;
```

### Expressions

```ebnf
expression  = logical_or ;
```

#### Logical OR

```ebnf
logical_or  = logical_and , { "||" , logical_and } ;
```

The `||` operator short-circuits: if the left operand is truthy, the right operand is not evaluated.

#### Logical AND

```ebnf
logical_and = pipeline , { "&&" , pipeline } ;
```

The `&&` operator short-circuits: if the left operand is not truthy, the right operand is not evaluated.

#### Pipeline

```ebnf
pipeline    = comparison , { (">" | ">>") , comparison } ;
```

The `>` operator pipes the left value as the first argument to the right function.
The `>>` operator starts the right function in the background and returns a Future.

If the right side is a call expression and contains the `_` placeholder, the piped
value replaces the placeholder instead.

#### Vertical Pipeline

```ebnf
vertical_pipeline = NEWLINE , INDENT , { vertical_stage , NEWLINE } , DEDENT ;
vertical_stage    = (">" | ">>") , expression ;
```

Vertical pipelines are syntactic sugar for horizontal pipelines chained across multiple indented lines. `>>` stages start parallel execution.

#### Comparison

```ebnf
comparison  = concat , { ("==" | "!=" | "<" | ">" | "<=" | ">=") , concat } ;
```

#### Concatenation

```ebnf
concat      = additive , { "++" , additive } ;
```

The `++` operator concatenates two strings.

#### Additive

```ebnf
additive    = multiplicative , { ("+" | "-") , multiplicative } ;
```

Note: the `+` operator on strings performs concatenation (runtime dispatch; `++` is preferred for clarity).

#### Multiplicative

```ebnf
multiplicative = power , { ("*" | "/" | "%") , power } ;
```

#### Power

```ebnf
power       = unary , { "**" , unary } ;
```

The `**` operator is right-associative.

#### Unary

```ebnf
unary       = { "-" | "!" } , call ;
```

Prefix operators: `-` (arithmetic negation), `!` (logical negation).

#### Call

```ebnf
call        = dot , { explicit_call | implicit_call | index_slice } ;

explicit_call = "(" , [ expression , { "," , expression } ] , ")" ;
implicit_call = value_token , { value_token } ;
index_slice   = "[" , expression , [ ".." , [ expression ] ] , "]" ;
```

Explicit calls use parentheses. Implicit calls allow calling functions without parentheses when the argument is a value token (identifier, literal, or grouped expression). Each value token on the same logical line becomes a separate argument.

For indexing: `x[i]` desugars to a `[]` infix expression. For slicing: `x[i..j]` produces a `SliceExpression`.

#### Dot

```ebnf
dot         = primary , { "." , identifier } ;
```

Dot expressions provide field access on maps: `obj.field`.

#### Primary

```ebnf
primary     = identifier
            | integer
            | float
            | string
            | boolean
            | nil
            | function_literal
            | list_literal
            | map_literal
            | group
            | if_expression
            | match_expression
            | while_expression
            | for_expression
            | try_expression ;
```

#### Function Literal

```ebnf
function_literal = "fn" , { identifier } , block ;
```

An anonymous function (function literal) is an expression form of `fn` without a name. It can be assigned to a variable, passed as an argument, or returned from a function.

#### List Literal

```ebnf
list_literal = "[" , [ expression , { "," , expression } ] , "]" ;
```

#### Map Literal

```ebnf
map_literal  = "{" , [ identifier , ":" , expression , { "," , identifier , ":" , expression } ] , "}" ;
```

Map keys are identifiers (evaluated as string constants at compile time).

#### Group

```ebnf
group = "(" , expression , ")" ;
```

### Block Structure

```ebnf
block = ":" [ NEWLINE , INDENT , { statement , NEWLINE } , DEDENT ]
      | NEWLINE , INDENT , { statement , NEWLINE } , DEDENT ;
```

A block is introduced by an expression head (e.g., `if`, `while`, `fn`, `try`). If the head is followed by a colon and then a newline, the block body is indented. Alternatively, the colon can be omitted and the body is simply the indented block.

The body consists of zero or more statements, each typically on its own line. The block terminates with a `DEDENT` token.

### Control Flow Expressions

#### If Expression

```ebnf
if_expression = "if" , expression , block , { "else" , "if" , expression , block } , [ "else" , block ] ;
```

If/else chains are expressions: they evaluate to the value of the last expression in the executed branch. If no branch matches and there is no `else`, the result is `nil`.

#### Match Expression

```ebnf
match_expression = "match" , expression , NEWLINE , INDENT , { match_case } , DEDENT ;
match_case       = "|" , pattern , "->" , expression ;
pattern          = identifier | integer | string | boolean | nil | "_" ;
```

Each case pattern is compared for equality with the match value. The `_` wildcard matches any value. Cases are evaluated in order; the body of the first matching case is executed and its value becomes the result.

#### While Expression

```ebnf
while_expression = "while" , expression , block ;
```

The body is repeated while the condition is truthy. `break` exits the loop; `continue` jumps to the next condition check.

#### For Expression

```ebnf
for_expression  = "for" , identifier , "in" , expression , block ;
```

The for-in expression iterates over a list. On each iteration, the iterator variable is bound to the next element. The body is executed for each element.

#### Try Expression

```ebnf
try_expression  = "try" , block , "catch" , [ identifier ] , block ;
```

The try block is executed. If an error occurs (an `ERROR` object is produced), execution jumps to the catch block, with the error value bound to the catch parameter identifier. If no error occurs, the catch block is skipped.

### Operator Precedence

Operators are listed from lowest to highest precedence. Higher numbers indicate tighter binding.

| Precedence | Level | Operators | Associativity |
|------------|-------|-----------|---------------|
| 1 | Lowest | (none) | — |
| 2 | Or | `\|\|` | Left |
| 3 | And | `&&` | Left |
| 4 | Pipeline | `>`, `>>` | Left |
| 5 | Equals | `==`, `!=` | Left |
| 6 | Compare | `<`, `<=`, `>`, `>=` | Left |
| 7 | Sum | `+`, `-` | Left |
| 8 | Product | `*`, `/`, `%` | Left |
| 9 | Power | `**` | Right |
| 10 | Concat | `++` | Left |
| 11 | Prefix | `-x`, `!x` | Right (unary) |
| 12 | Call | `f(x)`, `x[i]` | Left |
| 13 | Dot | `x.y` | Left |

### Keywords

All reserved keywords (cannot be used as identifiers):

```
fn       match    if       else     while
for      break    continue import   export
enum     defer    return   try      catch
true     false    nil      in       not
as
```

Note: `in`, `not`, and `as` are context-sensitive keywords. `in` is recognized as `IDENT` with literal `"in"`; `not` is recognized as `BANG` (unary `!` equivalent); `as` is only recognized after an import path.

### Built-in Types (Runtime)

```
INTEGER     FLOAT       STRING      BOOLEAN     NIL
LIST        MAP         FUNCTION    COMPILED_FUNCTION
CLOSURE     ERROR       RESULT      TCP_CONN    TCP_LISTENER
BUILTIN
```
