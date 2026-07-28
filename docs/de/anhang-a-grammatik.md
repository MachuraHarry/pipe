# Anhang A: Formale Grammatik

Die vollständige EBNF-Grammatik der Pipe-Sprache.

## A.1 Lexikalische Tokens

```ebnf
(* Kommentare *)
comment       = "--" { any_character } newline ;

(* Literale *)
integer       = digit { digit } ;
float         = digit { digit } "." digit { digit } ;
string        = '"' { char | escape } '"' | '`' { any_char_except_backtick } '`' ;
boolean       = "true" | "false" ;
nil           = "nil" ;
identifier    = letter { letter | digit | "_" } ;

(* Escape-Sequenzen *)
escape        = "\" ( "n" | "t" | "r" | "\" | '"' | "0" ) ;

(* Zeichenklassen *)
letter        = "a".."z" | "A".."Z" | "_" ;
digit         = "0".."9" ;
newline       = "\n" ;
```

## A.2 Grammatik

```ebnf
(* Programm *)
program       = { statement | newline } ;

(* Statements *)
statement     = function_def
              | variable_def
              | enum_def
              | export_statement
              | import_statement
              | defer_statement
              | return_statement
              | break_statement
              | continue_statement
              | expression_statement
              ;

function_def  = "fn" identifier { identifier } newline INDENT statement { statement } DEDENT ;

variable_def  = identifier ":" expression newline
              | identifier compound_assign expression newline ;

compound_assign = "+=" | "-=" | "*=" | "/=" | "%=" ;

enum_def      = "enum" identifier ":" identifier { "," identifier } newline ;

export_statement = "export" function_def ;

import_statement = "import" string ( "as" identifier )? newline ;

defer_statement = "defer" expression newline ;

return_statement = "return" [ expression ] newline ;

break_statement = "break" newline ;

continue_statement = "continue" newline ;

expression_statement = expression newline ;

(* Ausdrücke *)
expression    = logical_or ;

logical_or    = logical_and { "||" logical_and } ;

logical_and   = pipeline { "&&" pipeline } ;

pipeline      = comparison { ">" comparison }           (* horizontale Pipeline *)
              | comparison newline INDENT ">" { comparison } DEDENT ;  (* vertikale Pipeline *)

comparison    = concat { ( "==" | "!=" | "<" | "<=" | ">" | ">=" ) concat } ;

concat        = additive { "++" additive } ;

additive      = multiplicative { ( "+" | "-" ) multiplicative } ;

multiplicative = power { ( "*" | "/" | "%" ) power } ;

power         = unary { "**" unary } ;

unary         = { "!" | "-" } call ;

call          = dot { "(" [ expression { "," expression } ] ")"
                    | "[" expression ( ".." expression )? "]"
                    | identifier } ;                     (* implicit call *)

dot           = primary { "." identifier } ;

primary       = integer
              | float
              | string
              | boolean_expr
              | nil_expr
              | identifier
              | list_literal
              | map_literal
              | function_literal
              | if_expression
              | match_expression
              | while_expression
              | for_expression
              | try_expression
              | group
              ;

boolean_expr  = "true" | "false" ;
nil_expr      = "nil" ;

function_literal = "fn" { identifier } newline INDENT statement { statement } DEDENT ;

list_literal  = "[" [ expression { "," expression } ] "]" ;

map_literal   = "{" [ identifier ":" expression { "," identifier ":" expression } ] "}" ;

group         = "(" expression ")" ;

(* Kontrollfluss *)

if_expression = "if" expression newline INDENT statement { statement } DEDENT
                { "else" "if" expression newline INDENT statement { statement } DEDENT }
                [ "else" newline INDENT statement { statement } DEDENT ] ;

match_expression = "match" expression newline INDENT
                   match_case { match_case }
                   DEDENT ;

match_case    = "|" expression "->" expression newline ;

while_expression = "while" expression newline INDENT statement { statement } DEDENT ;

for_expression  = "for" identifier "in" expression newline INDENT statement { statement } DEDENT ;

try_expression = "try" newline INDENT statement { statement } DEDENT
                 "catch" identifier newline INDENT statement { statement } DEDENT ;

(* Block-Struktur *)
block         = INDENT statement { statement } DEDENT ;
```

## A.3 Operator-Präzedenz

| Ebene | Operatoren | Assoziativität |
|-------|-----------|----------------|
| 13 | `.` | Links |
| 12 | `()` `[]` | Links |
| 11 | `!`, `-` (unär) | Rechts |
| 10 | `++` | Links |
| 9 | `**` | Rechts |
| 8 | `*`, `/`, `%` | Links |
| 7 | `+`, `-` | Links |
| 6 | `<`, `>`, `<=`, `>=` | Links |
| 5 | `==`, `!=` | Links |
| 4 | `>` (Pipeline) | Links |
| 3 | `&&` | Links |
| 2 | `\|\|` | Links |
| 1 | `:` (Zuweisung) | Rechts |

## A.4 Keywords (reserviert)

```
fn, match, if, else, while, for, in, break, continue,
import, export, enum, defer, return, try, catch,
true, false, nil
```

## A.5 Eingebaute Typen

```
nil, bool, num (Integer | Float), str, list, map, fn
```

## A.6 INDENT/DEDENT Regeln

- Einrückung definiert Blöcke (wie Python)
- Empfohlen: 4 Leerzeichen pro Ebene
- INDENT wird emittiert, wenn die Einrückung zunimmt
- DEDENT wird emittiert, wenn die Einrückung abnimmt
- Innerhalb von `()`, `[]`, `{}` wird Einrückung ignoriert
- Leerzeilen und reine Kommentarzeilen werden bei der Einrückung ignoriert
