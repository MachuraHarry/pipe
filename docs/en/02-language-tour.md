# Language Tour

## Comments

Single-line comments start with `--` and extend to the end of the line:

```pipe
-- This is a comment
-- inline comment after code
x: 10
```

Pipe has no multi-line comment syntax. Use `--` on each line for block comments.

## Indentation

Pipe uses significant whitespace to define blocks. Four spaces per indentation level is the recommended convention. Mixing tabs and spaces is not allowed — pick one style and be consistent.

Block context is established by keywords that introduce bodies (`if`, `else`, `while`, `for`, `fn`, `match`, `defer`):

```pipe
fn foo x
  if x > 0
    print "positive"
  else
    print "non-positive"
```

The block ends when indentation returns to the previous level or the enclosing context.

## Variable Definition and Reassignment

Variables are defined using the `name: value` syntax:

```pipe
name: "Pipe"
count: 42
pi: 3[14159]
active: true
```

Reassignment uses the same syntax — the name is already in scope, so it updates the existing binding:

```pipe
count: 10
-- count is now 11
count: count + 1
```

## Compound Assignment

Shorthand operators for updating a variable in place:

```pipe
x: 10
-- x is now 15
x: x + 5
-- x is now 12
x: x - 3
-- x is now 24
x: x * 2
-- x is now 6
x: x / 4
-- x is now 2
x: x % 4
```

These expand to `x: x + 5`, `x: x - 3`, etc. The variable must already be defined.

## Function Calls

Pipe uses **space-based application** — arguments follow the function name, separated by spaces:

```pipe
print "hello"
len ([1, 2, 3])
push list 42
```

No commas between arguments. No parentheses needed for top-level calls.

**Parentheses for nested expressions**: when a function call appears inside an expression, wrap it in parentheses to disambiguate:

```pipe
-- (3 + 4) * 2 = 14
result: (add 3 4) * 2
```

**Implicit calls** occur when Pipe encounters a function name followed by arguments without a prior definition context:

```pipe
print (square 5)
```

## File Extension

Pipe source files use the `.pipe` extension:

```
program.pipe
utils.pipe
main.pipe
```

## All 19 Keywords

| Keyword | Description |
|---|---|
| `fn` | Define a function (named or anonymous) |
| `if` | Conditional branch |
| `else` | Alternative branch of `if` |
| `elif` | Shorthand for `else if` |
| `match` | Pattern matching expression |
| `case` | A branch within `match` |
| `while` | Conditional loop |
| `for` | Iteration loop with `in` |
| `in` | Specifies the collection for a `for` loop |
| `break` | Exit the nearest enclosing loop |
| `continue` | Skip to the next iteration of the nearest loop |
| `return` | Exit the current function early with a value |
| `defer` | Schedule code to run when the enclosing scope exits |
| `enum` | Define an enumeration of named integer constants |
| `true` | Boolean literal, not falsy |
| `false` | Boolean literal, falsy |
| `nil` | Absence of value, falsy |
| `and` | Logical AND (short-circuit) |
| `or` | Logical OR (short-circuit) |

## Operators Overview

### Arithmetic Operators

| Operator | Meaning | Example |
|---|---|---|
| `+` | Addition | `3 + 4` → `7` |
| `-` | Subtraction | `10 - 3` → `7` |
| `*` | Multiplication | `6 * 7` → `42` |
| `/` | Division | `10 / 4` → `2` (int) or `2.5` (float) |
| `%` | Modulo | `10 % 3` → `1` |
| `**` | Exponentiation | `2 ** 10` → `1024` |
| `++` | String concatenation | `"hello" ++ " world"` → `"hello world"` |

### Compound Assignment Operators

| Operator | Expands to |
|---|---|
| `+=` | `x: x + y` |
| `-=` | `x: x - y` |
| `*=` | `x: x * y` |
| `/=` | `x: x / y` |
| `%=` | `x: x % y` |

### Comparison Operators

| Operator | Meaning |
|---|---|
| `==` | Equal |
| `!=` | Not equal |
| `<` | Less than |
| `>` | Greater than |
| `<=` | Less than or equal |
| `>=` | Greater than or equal |

### Logical Operators

| Operator | Meaning |
|---|---|
| `!` (unary) | Logical NOT |
| `&&` | Logical AND (short-circuit) |
| `\|\|` | Logical OR (short-circuit) |

### String Operator

| Operator | Meaning |
|---|---|
| `++` | Concatenate two strings |

### Pipeline Operator

| Operator | Meaning |
|---|---|
| `>` | Pass left-hand value as first argument to right-hand function |
| `>>` | Start right-hand function in background, return Future immediately |

### Range Operator

| Operator | Meaning |
|---|---|
| `..` | Create a range (used with `range()` function) |

### Assignment

| Syntax | Meaning |
|---|---|
| `name: value` | Define or reassign a variable |

## Types Overview

| Type | Literal Examples | Description |
|---|---|---|
| `nil` | `nil` | Absence of value |
| `bool` | `true`, `false` | Boolean true/false |
| `num` | `42`, `3.14`, `-7` | Number (integer or float) |
| `str` | `"hello"`, `` `multi\nline` `` | String of UTF-8 characters |
| `list` | `[1, 2, 3]`, `["a", true]` | Ordered, dynamic array |
| `map` | `{"key": "value"}` | Associative array (string keys) |
| `fn` | `fn x x * 2` | First-class function value |

## Truthiness

Only two values are considered falsy in Pipe:

- `false`
- `nil`

Everything else — including `0`, `""` (empty string), `[]` (empty list), and `{}` (empty map) — is **truthy**.

```pipe
if 0
  -- this executes
    print "0 is truthy"

if ""
  -- this executes
    print "empty str is truthy"

if nil
  -- this does NOT execute
    print "nil is falsy"
```

## Syntax Cheatsheet

```
-- comment
x: 42                        -- variable definition
x: x + 1                     -- reassignment
x += 1                       -- compound assignment

fn add a b                   -- function definition
  a + b

if x > 0                     -- conditional
  print "positive"
elif x < 0
  print "negative"
else
  print "zero"

match x                      -- pattern matching
| 0 -> "zero"
| _ -> "other"

while x > 0                  -- while loop
  print x
  x -= 1

for item in list             -- for-in loop
  print item

for i in range 5             -- numeric range loop
  print i

value > transform > output   -- pipeline
value >> async_op >> next_op  -- parallel pipeline
return value                 -- early return
defer print "cleanup"        -- deferred execution

enum Color                   -- enumeration
  Red Green Blue
```

## Recommended Project Structure

For projects with multiple files, a typical layout:

```
my-project/
  main.pipe          -- entry point
  utils.pipe         -- shared utilities
  lib/
    math.pipe        -- math helpers
    strings.pipe     -- string functions
    io.pipe          -- file and I/O operations
  data/
    input.txt        -- sample data files
  README.md
```

Pipe supports a full module system with `import`/`export`, a central module registry, and versioned dependencies (`@1.0.0`). See [Chapter 9: Modules and Imports](09-modules-and-imports.md) and the [Module Ecosystem](21-ecosystem.md).
