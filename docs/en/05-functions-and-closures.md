# Functions and Closures

## Function Definition

Functions are defined with the `fn` keyword followed by the function name and parameter list:

```pipe
fn name param1 param2 param3
  body
  ...
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
print (add 3 5)      -- 8
print (greet "Pipe")  -- Hello, Pipe!
```

When a function call appears inside a larger expression, wrap it in parentheses to disambiguate:

```pipe
result: (add 1 2) * (add 3 4)   -- (1+2) * (3+4) = 21
```

For top-level calls (where they stand alone as a statement), parentheses are optional:

```pipe
print "direct call"        -- no parens needed
greet "world"               -- standalone call
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
result: (fn a b (a * b) + 10) 3 4   -- (3 * 4) + 10 = 22
```

Anonymous functions are first-class values — the same as named functions, just without a name in scope.

## Closures

A **closure** is a function that captures variables from its enclosing scope. The captured variables remain accessible even after the enclosing function returns.

```pipe
fn make_counter start
  fn
    start += 1
    start

counter: make_counter 0
print (counter)   -- 1
print (counter)   -- 2
print (counter)   -- 3

counter2: make_counter 100
print (counter2)  -- 101
print (counter2)  -- 102
print (counter)   -- 4   (different counter, independent state)
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

print (add5 3)    -- 8
print (add5 7)    -- 12
print (add10 3)   -- 13
print (add10 7)   -- 17
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

print (factorial 5)    -- 120
print (factorial 10)   -- 3628800
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

print (factorial_tail 5 1)     -- 120
print (factorial_tail 5000 1)  -- works without stack overflow
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

increment: fn n n + 1
print (apply_twice increment 5)   -- 7  (increment(increment(5)))
```

A custom map function:

```pipe
fn my_map list f
  result: []
  for item in list
    push result (f item)
  result

nums: [1, 2, 3, 4, 5]
double: fn x x * 2
print (my_map nums double)   -- [2, 4, 6, 8, 10]
```

A filtering function:

```pipe
fn my_filter list pred
  result: []
  for item in list
    if pred item
      push result item
  result

even: fn x x % 2 == 0
print (my_filter [1, 2, 3, 4, 5, 6] even)   -- [2, 4, 6]
```

### Functions as Return Values

Functions can create and return new functions customized by their parameters:

```pipe
fn multiplier factor
  fn x
    x * factor

double: multiplier 2
triple: multiplier 3

print (double 5)    -- 10
print (triple 5)    -- 15
```

Function composition:

```pipe
fn compose f g
  fn x
    f (g x)

add1: fn x x + 1
square: fn x x * x

add1_then_square: compose square add1
square_then_add1: compose add1 square

print (add1_then_square 3)     -- (3+1)^2 = 16
print (square_then_add1 3)     -- (3^2)+1 = 10
```

## Function Scope and Variable Visibility

### Parameter Scope

Function parameters are local to the function body. They shadow variables from outer scopes:

```pipe
x: 10
fn foo x
  print x    -- prints the parameter, not the outer x

foo 42        -- prints 42
print x       -- prints 10 (outer x unchanged)
```

### Lexical Scoping

Pipe uses **lexical scoping**: a variable's scope is determined by its position in the source code. Inner functions can access variables from their enclosing functions:

```pipe
fn outer a
  fn inner b
    a + b
  inner

add10: outer 10
print (add10 5)   -- 15

-- 'a' is captured from outer's scope and used inside inner
```

### Block Scope

Variables defined inside blocks (`if`, `while`, `for`, `match`) are scoped to that block:

```pipe
if true
  temp: 42
  print temp   -- 42

-- print temp  -- error: temp is not in scope here
```

### Global/Top-Level Scope

Variables defined at the top level of a file (outside any function or block) are visible throughout the entire program. Functions can read and assign to top-level variables:

```pipe
counter: 0

fn increment
  counter += 1
  counter

fn reset
  counter: 0
```

Top-level variables provide shared state accessible from any function. Use them sparingly — prefer passing values through parameters and return values when practical.
