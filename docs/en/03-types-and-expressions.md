# Types and Expressions

## Data Types

Pipe has seven built-in data types, plus user-definable structs. Every value belongs to exactly one of these types.

### nil

The absence of a value. Used for uninitialized variables, empty returns, and missing map keys.

```
Internal representation: nil (singleton)
Literal: nil
```

```pipe
x: nil
-- "nil"
print (type_of nil)
```

### bool

Boolean truth values.

```
Internal representation: Go bool
Literals: true, false
```

```pipe
flag: true
done: false
-- "bool"
print (type_of true)
```

#### Truthiness Rules

Only `false` and `nil` are falsy. Every other value — including `0`, `""`, `[]`, `{}` — evaluates to `true` in boolean contexts.

```pipe
if 0
  -- executes, because 0 is truthy
    print "reached"

if nil
  -- does not execute
    print "unreachable"
```

### num

Numbers in Pipe are stored internally with a dual representation: if the value can be represented as a 64-bit integer (`int64`), it is; otherwise it is stored as a 64-bit float (`float64`). Conversion between the two happens automatically during arithmetic operations.

```
Internal representation: int64 or float64
Literals: 42, -7, 3.14159, 2.5e3, 0xFF
```

```pipe
-- stored as int64
age: 42
-- stored as float64
pi: 3[14159]
-- 255, stored as int64
hex: 0xFF
-- 1500[0], stored as float64
sci: 1.5e3

-- integer arithmetic produces integers
-- 15 (int64)
a: 10 + 5

-- mixing int and float promotes to float
-- 13[5] (float64)
b: 10 + 3[5]

-- division of integers that doesn't divide evenly
-- 2 (int64, truncates toward zero)
c: 10 / 4
-- 2[5] (float64)
d: 10 / 4[0]
```

### str

Strings are immutable sequences of UTF-8 characters.

```
Internal representation: Go string
Literals: "double-quoted", \`backtick multi-line\`
```

**Double-quoted strings** support escape sequences:

| Sequence | Character |
|---|---|
| `\n` | Newline |
| `\t` | Tab |
| `\r` | Carriage return |
| `\\` | Backslash |
| `\"` | Double quote |
| `\0` | Null byte |

```pipe
greeting: "Hello\nWorld"
path: "C:\\Users\\pipe"
quote: "She said \"Hi\""
```

**Backtick strings** span multiple lines and do not process escape sequences:

```pipe
poem: `Roses are red
Violets are blue
Pipe is concise
And pipelines are too`
```

### list

An ordered, dynamically-sized collection of values. Lists can hold elements of mixed types.

```
Internal representation: Go []interface{} (slice of interface{})
Literal: [element1, element2, ...]
```

```pipe
empty: []
nums: [1, 2, 3, 4, 5]
mixed: [42, "hello", true, nil, [1, 2]]

-- access by index (0-based)
-- 1
first: nums[0]
-- 3
third: nums[2]

-- modify element
-- nums is now ([1, 2, 99, 4, 5])
set nums 2 99

-- length
-- 5
len nums

-- append
-- nums is now ([1, 2, 99, 4, 5, 6])
push nums 6
```

### map

An associative collection mapping string keys to values. Only `str` keys are allowed.

```
Internal representation: Go map[string]interface{}
Literal: {"key1": value1, "key2": value2}
```

```pipe
empty: {}
person: {name: "Alice", age: 30, active: true}

-- access by key
-- "Alice"
person["name"]

-- set/update key
set person "city" "NYC"

-- check existence
-- true
has person "age"
-- false
has person "email"

-- length
-- 3
len person
```

### fn

Functions are first-class values. They can be assigned to variables, passed as arguments, and returned from other functions.

Pipe supports two forms: the indented **block form** for multi-statement bodies, and the **inline form** (`fn x: expr`) for single-expression bodies.

```
Internal representation: compiled function reference or AST node
Literal: fn params body (block form), fn params: expr (inline form)
```

```pipe
-- Block form (multi-statement)
double: fn x
  x * 2

-- Inline form (single expression)
square: fn x: x * x

-- functions are values
some_fn: double
-- 10
some_fn 5

-- anonymous functions as arguments
triple: fn x
    x * 3
result: (map ([1, 2, 3]) triple)
```

### struct

Structs are user-defined record types with named fields. They allow grouping related data into a single value with positional construction and dot-notation field access.

```pipe
-- Block form with optional defaults
struct Person
    name: "Unknown"
    age: 0
    active: true

-- Inline form (no defaults, comma-separated)
struct Point: x, y

-- Create instances (positional arguments, space-separated)
alice: Person "Alice" 30 true
origin: Point 0 0

-- Field access via dot notation
alice.name    -- "Alice"
origin.x      -- 0

-- Field access error on unknown field
origin.z      -- ERROR: struct Point has no field 'z'

-- Dot access only works on structs, maps, and for .message on errors
42.x          -- ERROR: cannot use .x on INTEGER
```

## Operators

### Arithmetic Operators

| Operator | Description | Example | Result |
|---|---|---|---|
| `+` | Addition | `10 + 5` | `15` |
| `-` | Subtraction | `10 - 5` | `5` |
| `*` | Multiplication | `10 * 5` | `50` |
| `/` | Division | `10 / 4` | `2` (int) or `2.5` (float) |
| `%` | Modulo (remainder) | `10 % 3` | `1` |
| `**` | Exponentiation | `2 ** 8` | `256` |

**Division behavior**: if both operands are integers and the division is exact, the result is `int64`. If the division is not exact, the result is truncated toward zero. If either operand is a float, the result is `float64`.

### Comparison Operators

| Operator | Description | Example |
|---|---|---|
| `==` | Equal | `x == y` |
| `!=` | Not equal | `x != y` |
| `<` | Less than | `x < 10` |
| `>` | Greater than | `x > 0` |
| `<=` | Less than or equal | `x <= 100` |
| `>=` | Greater than or equal | `x >= 0` |

Comparisons work across compatible types. Numbers compare numerically, strings compare lexicographically, booleans compare by value. Comparing incompatible types (e.g. `3 == "three"`) returns `false`.

### Logical Operators

| Operator | Description | Short-circuit? |
|---|---|---|
| `!` | Logical NOT (unary) | N/A |
| `&&` | Logical AND | Yes — if left is falsy, right is not evaluated |
| `\|\|` | Logical OR | Yes — if left is truthy, right is not evaluated |

```pipe
check: fn x
  print "checking " ++ (to_str x)
  x > 0

-- short-circuit: check(5) is never called
-- prints nothing, returns false
false && (check 5)

-- short-circuit: check(-3) is never called
-- prints nothing, returns true
true || (check -3)
```

### String Concatenation Operator

The `++` operator concatenates two strings. Both operands must be of type `str`.

```pipe
-- "Hello, World"
greeting: "Hello, " ++ "World"
path: dir ++ "/" ++ filename
```

### Compound Assignment Operators

| Operator | Equivalent to |
|---|---|
| `x += y` | `x: x + y` |
| `x -= y` | `x: x - y` |
| `x *= y` | `x: x * y` |
| `x /= y` | `x: x / y` |
| `x %= y` | `x: x % y` |

```pipe
counter: 0
-- counter: counter + 1
counter: counter + 1
-- counter: counter * 2
counter: counter * 2
```

### Unary Operators

| Operator | Description | Example |
|---|---|---|
| `-` | Numeric negation | `-x`, `-(3 + 4)` |
| `!` | Logical negation | `!done`, `!(x > 10)` |

## Operator Precedence

Pipe defines 13 precedence levels, from highest (tightest binding) to lowest:

| Level | Operators | Associativity | Example |
|---|---|---|---|
| 1 | `.` (field access) | Left | `obj.field` |
| 2 | `[]` (index) | Left | `list[0]`, `map["key"]` |
| 3 | `fn` (function literal) | Right | `fn x x * 2` |
| 4 | `-` (negation), `!` (not) | Right | `-x`, `!done` |
| 5 | `**` (exponent) | Right | `2 ** 3 ** 2` → `2 ** 9` |
| 6 | `*`, `/`, `%` | Left | `a * b / c` |
| 7 | `+`, `-` (binary) | Left | `a + b - c` |
| 8 | `++` (concat) | Left | `a ++ b ++ c` |
| 9 | `<`, `>`, `<=`, `>=` | Non-assoc | `a < b` |
| 10 | `==`, `!=` | Non-assoc | `a == b` |
| 11 | `&&` | Left | `a && b && c` |
| 12 | `\|\|` | Left | `a \|\| b \|\| c` |
| 13 | `>`, `>>` (pipeline) | Left | `a > b >> c` |

**Non-associative** means chaining without parentheses is an error: `a < b < c` is not valid. Use `(a < b) && (b < c)` instead.

**Precedence examples**:

```pipe
-- negation binds tighter than multiplication
-- (-3) * 4 = -12
-3 * 4

-- exponent binds tighter than negation
-- -(2 ** 4) = -16
-2 ** 4

-- pipeline has lowest precedence
-- (x + 1) > double
x + 1 > double

-- logical operators bind lower than comparisons
-- (x > 0) && (x < 10)
x > 0 && x < 10

-- string concat binds between arithmetic and comparison
-- (a ++ b) == "ab"
a ++ b == "ab"
```

When in doubt, use parentheses to make intent explicit:

```pipe
-- explicit: (-2) ** 4? no, -(2 ** 4)
-(2 ** 4)
-- explicit grouping
(a ++ b) ++ c
```

## Type Checking Functions

```pipe
-- returns the type as a string: "nil", "bool", "num", "str", "list", "map", "fn", "STRUCT"
type_of value
-- true if value is nil
is_nil value
-- true if value is a number
is_num value
-- true if value is a string
is_str value
-- true if value is a list
is_list value
-- true if value is a map
is_map value
```

```pipe
-- "num"
type_of 42
-- "str"
type_of "hello"
-- "list"
type_of ([1, 2])
-- "fn"
type_of (fn x
    x)

-- true
is_num 3[14]
-- false
is_num "three"
-- true
is_list []
-- true
is_nil nil

-- "STRUCT"
struct P: x, y
type_of (P)
```

## Type Conversion

```pipe
-- convert any value to its string representation
to_str value
-- parse a string into a number, or convert a number (no-op)
to_num value
```

```pipe
-- "42"
to_str 42
-- "true"
to_str true
-- "[1, 2, 3]"
to_str ([1, 2, 3])

-- 42 (int64)
to_num "42"
-- 3[14] (float64)
to_num "3[14]"
-- nil (parse failure)
to_num "abc"
-- 42 (no change)
to_num 42

-- safe conversion pattern
n: to_num input
if is_nil n
  print "invalid number"
else
  print n * 2
```
