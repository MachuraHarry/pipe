# Functions and Closures

## Function Definition

Functions are defined with the `fn` keyword followed by the function name and parameter list:

```pipe
fn name param1 param2 param3
  body
      
```

Parameters are separated by **spaces**, not commas. The body is defined by indentation.

```pipe
fn add a b
  a + b

fn greet name
  "Hello, " ++ name ++ "!"
```

The **last expression** in the function body is the return value. No `return` keyword is needed for normal returns — use `return` only for early exits (see [Control Flow](./04-control-flow.md#return)).

## Function Calls

Calling a function uses space-separated arguments:

```pipe
-- 8
print (add 3 5)
-- Hello, Pipe!
print (greet "Pipe")
```

When a function call appears inside a larger expression, wrap it in parentheses to disambiguate:

```pipe
-- (1+2) * (3+4) = 21
result: (add 1 2) * (add 3 4)
```

For top-level calls (where they stand alone as a statement), parentheses are optional:

```pipe
-- no parens needed
print "direct call"
-- standalone call
greet "world"
```

**Pipeline calls** use the `>` operator and are covered in [Chapter 6](./06-pipelines.md).

## Anonymous Functions

A function without a name can be assigned to a variable or passed inline:

```pipe
double: fn x
  x * 2

square: fn x
  x * x

-- inline anonymous function
-- (3 * 4) + 10 = 22
compute: fn a b
    (a * b) + 10
result: compute 3 4
```

### Inline Lambda Syntax (v0.8+)

For single-expression functions, Pipe supports a compact inline syntax using a colon:

```pipe
-- Inline: fn param: expression
double: fn x: x * 2

-- Multi-parameter inline lambda
add: fn a b: a + b

-- As argument to filter
filter [1, 2, 3, 4, 5] (fn x: x > 2)

-- As argument to map
map [1, 2, 3] (fn x: x * 10)

-- In pipeline chains
[1, 2, 3, 4, 5]
    > filter (fn x: x % 2 == 0)
    > map (fn x: x * 3)
    > print
```

The inline form is equivalent to the multi-line block form but allows writing
simple function literals on a single line. Parameters before the colon become
function parameters; the expression after the colon is the function body.

> **Whitespace rule for `[`**: directly attached to a value, `[...]` is an
> index/slice postfix (`xs[0]`, `xs[1..3]`); separated by whitespace, it starts
> a fresh list literal — as a statement (`xs: [1, 2]`) or as a call argument
> (`map [1, 2, 3] (fn x: x * 10)` above). Prefer a variable when you need an
> index into an expression result: `r: map nums f` then `r[0]`.

For multi-statement function bodies, use the indented block form (`fn params\n    body`).

Anonymous functions are first-class values — the same as named functions, just without a name in scope.

## Closures

A **closure** is a function that captures variables from its enclosing scope. The captured variables remain accessible even after the enclosing function returns.

```pipe
fn make_counter start
  fn counter
    start: start + 1
    start

counter: make_counter 0
-- 1
print (counter)
-- 2
print (counter)
-- 3
print (counter)

counter2: make_counter 100
-- 101
print (counter2)
-- 102
print (counter2)
-- 4   (different counter, independent state)
print (counter)
```

Each call to `make_counter` creates a new closure with its own captured `start` variable. The closures are independent — modifying one does not affect the other.

### make_adder Example

A classic closure example: a function factory that creates adders with a fixed increment:

```pipe
make_adder: fn increment
  fn n
    n + increment

add5: make_adder 5
add10: make_adder 10

-- 8
print (add5 3)
-- 12
print (add5 7)
-- 13
print (add10 3)
-- 17
print (add10 7)
```

The parameter `increment` is captured by the inner function and persists across calls.

## Recursion

Functions can call themselves. The classic factorial example:

```pipe
fn factorial n
  if n <= 1
    1
  else
    n * (factorial (n - 1))

-- 120
print (factorial 5)
-- 3628800
print (factorial 10)
```

### Tail Call Optimization (TCO)

Pipe's VM mode detects and optimizes **tail-recursive** calls — calls where the function's return value is the direct result of a recursive call. When TCO applies, the call stack frame is reused rather than growing.

A tail-recursive factorial:

```pipe
fn factorial_tail n acc
  if n <= 1
    acc
  else
    factorial_tail (n - 1) (n * acc)

-- 120
print (factorial_tail 5 1)
-- works without stack overflow
print (factorial_tail 5000 1)
```

In tree-walker mode, deep recursion may exhaust the Go stack. In VM mode, TCO allows arbitrary recursion depth as long as the call is in tail position. Pipe's implementation has been tested to a depth of 5000 nested tail calls without overflow.

A call is in **tail position** if it is the last expression evaluated before the function returns, with no work remaining after the call completes.

```pipe
-- TAIL position: good
fn sum list acc
  if (len list) == 0
    acc
  else
    sum (tail list) (acc + list[0])

-- NOT tail position: n * ... after recursive call
fn factorial n
  if n <= 1
    1
  else
    n * (factorial (n - 1))
```

## Higher-Order Functions

Functions can accept other functions as arguments and return functions as results.

### Passing Functions as Arguments

```pipe
fn apply_twice f x
  f (f x)

increment: fn n
    n + 1
-- 7  (increment(increment(5)))
print (apply_twice increment 5)
```

A custom map function:

```pipe
fn my_map list f
  result: []
  for item in list
    push result (f item)
  result

nums: [1, 2, 3, 4, 5]
double: fn x
    x * 2
-- [2, 4, 6, 8, 10]
print (my_map nums double)
```

A filtering function:

```pipe
fn my_filter list pred
  result: []
  for item in list
    if pred item
      push result item
  result

even: fn x
    x % 2 == 0
-- [2, 4, 6]
print (my_filter ([1, 2, 3, 4, 5, 6]) even)
```

### Functions as Return Values

Functions can create and return new functions customized by their parameters:

```pipe
fn multiplier factor
  fn x
    x * factor

double: multiplier 2
triple: multiplier 3

-- 10
print (double 5)
-- 15
print (triple 5)
```

Function composition:

```pipe
fn compose f g
  fn x
    f (g x)

add1: fn x
    x + 1
square: fn x
    x * x

add1_then_square: compose square add1
square_then_add1: compose add1 square

-- (3+1)^2 = 16
print (add1_then_square 3)
-- (3^2)+1 = 10
print (square_then_add1 3)
```

## Function Scope and Variable Visibility

### Parameter Scope

Function parameters are local to the function body. They shadow variables from outer scopes:

```pipe
x: 10
fn foo x
  -- prints the parameter, not the outer x
    print x

-- prints 42
foo 42
-- prints 10 (outer x unchanged)
print x
```

### Lexical Scoping

Pipe uses **lexical scoping**: a variable's scope is determined by its position in the source code. Inner functions can access variables from their enclosing functions:

```pipe
fn outer a
  fn inner b
    a + b
  inner

add10: outer 10
-- 15
print (add10 5)

-- 'a' is captured from outer's scope and used inside inner
```

### Block Scope

Variables defined inside blocks (`if`, `while`, `for`, `match`) are scoped to that block:

```pipe
if true
  temp: 42
  -- 42
    print temp

-- print temp  -- error: temp is not in scope here
```

### Global/Top-Level Scope

Variables defined at the top level of a file (outside any function or block) are visible throughout the entire program. Functions can read and assign to top-level variables:

```pipe
counter: 0

fn increment
  counter: counter + 1
  counter

fn reset
  counter: 0
```

Top-level variables provide shared state accessible from any function. Use them sparingly — prefer passing values through parameters and return values when practical.
