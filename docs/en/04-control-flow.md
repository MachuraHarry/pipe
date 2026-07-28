# Control Flow

## if / else if / else

The `if` expression evaluates a condition and executes the corresponding block. `if` is an **expression** — it returns the value of the block that executes.

```pipe
x: 10
msg: if x > 0
  "positive"
elif x < 0
  "negative"
else
  "zero"

print msg   -- "positive"
```

Each branch evaluates to the last expression in its block. If a branch has no expression, it returns `nil`. All branches are encouraged to return the same type, but this is not enforced.

**`elif`** is syntactic sugar for `else if` and can be chained:

```pipe
grade: if score >= 90
  "A"
elif score >= 80
  "B"
elif score >= 70
  "C"
elif score >= 60
  "D"
else
  "F"
```

### Nested if Expressions

`if` expressions can be nested arbitrarily:

```pipe
category: if age < 13
  "child"
else
  if age < 20
    "teenager"
  else
    if age < 65
      "adult"
    else
      "senior"
```

Using `elif` makes nested conditionals more readable:

```pipe
category: if age < 13
  "child"
elif age < 20
  "teenager"
elif age < 65
  "adult"
else
  "senior"
```

## match — Pattern Matching

The `match` expression compares a value against a series of patterns and evaluates the corresponding branch. Each branch is written as `| pattern -> expression`. The wildcard `_` matches any value.

```pipe
match value
| 0 -> "zero"
| 1 -> "one"
| _ -> "many"
```

`match` is an expression and returns the value of the matched branch.

### Fibonacci with match

A recursive Fibonacci function using `match` for base cases:

```pipe
fn fib n
  match n
  | 0 -> 0
  | 1 -> 1
  | _ -> (fib (n - 1)) + (fib (n - 2))

print (fib 10)   -- 55
```

### Calculator with match

A simple arithmetic calculator dispatching on operator strings:

```pipe
fn calculate a op b
  match op
  | "+" -> a + b
  | "-" -> a - b
  | "*" -> a * b
  | "/" -> a / b
  | _ -> nil

print (calculate 10 "+" 5)    -- 15
print (calculate 10 "*" 3)    -- 30
print (calculate 10 "^" 2)    -- nil (unknown operator)
```

### Multiple patterns per branch

A single branch can match multiple patterns separated by `|`:

```pipe
fn is_vowel ch
  match ch
  | "a" | "e" | "i" | "o" | "u" -> true
  | "A" | "E" | "I" | "O" | "U" -> true
  | _ -> false
```

## while Loop

The `while` loop repeatedly executes its body as long as the condition remains truthy.

```pipe
count: 5
while count > 0
  print count
  count -= 1
print "Liftoff!"
```

Output:

```
5
4
3
2
1
Liftoff!
```

### break

`break` immediately exits the innermost enclosing loop:

```pipe
i: 0
while true
  i += 1
  print i
  if i >= 5
    break
```

Output:

```
1
2
3
4
5
```

### continue

`continue` skips the rest of the current iteration and jumps to the next condition check:

```pipe
i: 0
while i < 10
  i += 1
  if i % 2 == 0
    continue
  print i
```

Output:

```
1
3
5
7
9
```

## for-in Loop

The `for-in` loop iterates over elements of a list:

```pipe
fruits: ["apple", "banana", "cherry"]
for fruit in fruits
  print fruit
```

### for-in with break

```pipe
found: nil
nums: [10, 20, 30, 40, 50]
for n in nums
  if n > 25
    found: n
    break

print found   -- 30
```

### for-in on range()

The `range` function generates a sequence of numbers for iteration:

```pipe
-- one argument: 0 to stop-1
for i in range 5
  print i
-- prints: 0, 1, 2, 3, 4

-- two arguments: start to stop-1
for i in range 3 8
  print i
-- prints: 3, 4, 5, 6, 7

-- three arguments: start to stop-1 with step
for i in range 0 10 2
  print i
-- prints: 0, 2, 4, 6, 8

-- negative step
for i in range 10 0 (-1)
  print i
-- prints: 10, 9, 8, 7, 6, 5, 4, 3, 2, 1
```

The stop value is **exclusive** — the loop runs while the counter is less than stop.

## return

`return` provides an early exit from a function, optionally with a value:

```pipe
fn find_first list target
  for item in list
    if item == target
      return item
  nil

print (find_first [1, 2, 3, 4] 3)   -- 3
print (find_first [1, 2, 3, 4] 9)   -- nil
```

Without `return`, a function returns the value of its last expression. `return` with no argument returns `nil`.

```pipe
fn early_out x
  if x < 0
    return       -- returns nil
  x * 2

print (early_out (-5))   -- nil
print (early_out 5)      -- 10
```

## defer

`defer` schedules an expression to execute when the enclosing scope exits. Deferred calls run in **LIFO** (Last-In, First-Out) order — the most recently deferred expression runs first.

```pipe
fn process_file path
  f: (open path)
  defer (close f)
  defer print "cleaning up"

  -- do work with f
  print "processing..."

print "calling process_file"
print (process_file "data.txt")
print "done"
```

Output:

```
calling process_file
processing...
cleaning up
close called
nil
done
```

### defer on top-level

`defer` can also be used at the top level of a script. Top-level deferred expressions run when the program exits (normally).

```pipe
defer print "Goodbye!"
print "Hello!"
```

Output:

```
Hello!
Goodbye!
```

## enum

`enum` defines a group of named integer constants starting from 0 and incrementing by 1 for each name:

```pipe
enum Color
  Red Green Blue

print Red    -- 0
print Green  -- 1
print Blue   -- 2
```

Multiple `enum` blocks can be defined. Each one starts counting from 0:

```pipe
enum Status
  Pending Active Completed

enum Role
  Admin Editor Viewer

if user.role == Admin
  print "has admin access"

if task.status == Completed
  print "done"
```

Each name in the `enum` block becomes a constant variable in the enclosing scope. The type is `num` (int64).
