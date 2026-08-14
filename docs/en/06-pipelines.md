# Pipelines

## The Central Language Feature

The pipeline is the defining feature of Pipe. It allows data to flow through a sequence of transformations, expressed as a top-to-bottom chain of operations. This design makes data processing scripts read like a description of the data's journey rather than a sequence of nested function calls.

```
┌──────────────────────────────────────────────────────┐
│                  PIPE DATA FLOW                       │
│                                                      │
│  input_value                                         │
│       │                                              │
│       ▼                                              │
│  ┌─────────┐    ┌──────────┐    ┌──────────┐        │
│  │ stage 1 │───▶│ stage 2  │───▶│ stage 3  │──▶ out │
│  │ > split │    │ > filter │    │ > print  │        │
│  └─────────┘    └──────────┘    └──────────┘        │
│       │              │               │               │
│  Sequential          │               │               │
│  (same thread)       ▼               ▼               │
│                 >> parallel    >> parallel            │
│                 (background)   (background)          │
│                                                      │
│  >  : top-to-bottom, one after another              │
│  >> : start in background, return Future            │
│  _  : placeholder for piped value                   │
└──────────────────────────────────────────────────────┘
```

## Vertical Pipeline Syntax (Recommended)

The vertical pipeline uses indented lines, each starting with `>` followed by a function call:

```pipe
initial_value
    > transformation1
    > transformation2
    > transformation3
```

The pipeline starts with a value on its own line. Each subsequent indented line begins with `>` and specifies the next transformation.

### Data Flow Direction

Data flows **top to bottom**. The value on the first line becomes the first argument to the function on the second line. The result of that function becomes the first argument to the function on the third line, and so on:

```pipe
5
    > double
    > square
    > to_str

-- equivalent to: to_str(square(double(5)))
-- result: "100"
```

This reads naturally: "take 5, double it, square the result, convert to string."

### Syntax Rules

1. The **initial value** must be on its own line at the base indentation level of the pipeline.
2. **Pipeline steps** are indented (4 spaces recommended) and begin with `>`.
3. The value from the previous step is **inserted as the first argument** to the function call.
4. Additional arguments follow the function name as usual.

```pipe
"hello"
    > replace "l" "x"
    > upper
    > print

-- equivalent to: print(upper(replace("hello", "l", "x")))
-- output: "HEXXO"
```

### Pipeline with Extra Arguments

When a pipeline step needs arguments beyond the piped value, list them after the function name:

```pipe
is_even: fn x
    x % 2 == 0
triple: fn x
    x * 3

[1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
    > filter is_even
    > map triple
    > sort
    > print

-- filter even numbers, multiply by 3, sort, print
-- output: [6, 12, 18, 24, 30]
```

The piped value (`[1, 2, ...]`) is inserted as the first argument to `filter`. The anonymous function is the second argument.

## Parallel Pipelines (`>>`)

The `>>` (double arrow) operator starts a pipeline stage **in the background** and returns immediately with a **Future** — a placeholder that will hold the result once the computation finishes. The program continues executing without waiting.

When the result is actually needed (used in arithmetic, passed to a function, printed, etc.), Pipe **automatically waits** for the background computation to finish.

### Syntax

```pipe
initial_value
    -- starts in background, returns Future immediately
    >> slow_operation
    -- continues when Future is resolved
    > next_operation
    > print
```

### Why Parallel Pipelines?

Many operations — especially AI calls — take seconds to complete. With `>`, each step waits for the previous one:

```pipe
-- Sequential: ~9 seconds (3 calls × 3 seconds each)
"Question 1" > ask > print
"Question 2" > ask > print
"Question 3" > ask > print
```

With `>>`, all three start simultaneously:

```pipe
-- Parallel: ~3 seconds (all 3 calls run concurrently)
"Question 1" >> ask
"Question 2" >> ask
"Question 3" >> ask

-- waits automatically
print answer1 ++ answer2 ++ answer3
```

### Futures: Automatic Resolution

When a `>>` stage returns, it does so with a **Future** — a promise that a value will arrive later. Futures resolve automatically when consumed:

```pipe
-- Store a Future in a variable
result: 10
    -- result is a Future
    >> slow_double

-- Arithmetic automatically waits for the Future
result + 100
    -- Future is resolved before addition
    > print

-- String concatenation waits too
-- waits for slow_double to finish
"I got: " ++ result

-- Function arguments wait
-- waits before comparing
max result 50
```

### Mixing `>` and `>>`

You can freely mix sequential and parallel pipeline stages:

```pipe
data
    -- parallel: start API call
    >> fetch_from_api
    -- sequential: wait, then parse
    > parse_json
    -- parallel: start analysis
    >> analyze
    -- sequential: wait, then format
    > format
    > print
```

Each `>>` starts its operation immediately when reached. Each `>` waits for the previous step's result. This gives precise control over which operations run in parallel and which must be sequential.

### Execution Modes

| Mode | `>>` Behavior |
|------|--------------|
| Tree-Walker (`./bin/pipe`) | True parallelism via goroutines for all functions |
| Bytecode VM (`./bin/pipe -vm`) | True parallelism for all functions (builtins and user-defined closures) |

Each parallel stage runs on its own VM with a snapshot of the globals, so a
spawned function sees the global state as it was when the `>>` stage started,
and writes it makes in the background do not leak back to the caller — matching
the tree-walker's per-branch environment cloning.

### The `_` Placeholder

Use `_` (underscore) to control where the piped value is inserted. Instead of always going as the first argument, `_` lets you place it at a specific position:

```pipe
"hello"
    -- replace "hello" "l" "x"
    > replace _ "l" "x"
    > print

-- "hello" goes where _ is: replace("hello", "l", "x")
```

Without `_`, the value goes as the first argument. With `_`, the value goes exactly where `_` appears:

```pipe
[3, 1, 4, 1, 5]
    -- push([3, 1, 4, 1, 5], 9)
    > push _ 9
    > len _
    > print

-- output: 6
```

`_` can appear multiple times in a pipeline step — each occurrence receives the same piped value:

```pipe
"alpha"
    -- prints "alpha alpha"
    > print _ _
```

## File Processing Pipeline

A realistic example: reading a file, splitting into lines, filtering, sorting, and printing:

```pipe
not_empty: fn line
    (len line) > 0
has_pipe: fn line
    has line "pipe"

read_file "data/words.txt"
    > split _ "\n"
    > filter not_empty
    > filter has_pipe
    > sort
    > for line in _
        print line
```

Breaking this down step by step:

1. `read_file "data/words.txt"` — returns the file contents as a string
2. `split _ "\n"` — splits into a list of lines
3. `filter` — keeps only non-empty lines
4. `filter` — keeps only lines containing "pipe"
5. `sort` — sorts alphabetically
6. The result is stored and iterated with `for`

### Building a Data Analysis Pipeline

A multi-step number processing pipeline:

```pipe
is_positive: fn n
    (is_num n) && (n > 0)

read_file "data/numbers.csv"
    > split _ ","
    > map _ to_num
    > filter is_positive
    > sum _
    > print
```

## Horizontal Pipeline Syntax

Pipe also supports a single-line, left-to-right pipeline syntax using `>`:

```pipe
5 > double > square > print
```

This is equivalent to the vertical form but written on one line. It is useful for short chains:

```pipe
"hello" > upper > print
[1, 2, 3] > len > print
```

### `>` Ambiguity: Comparison vs Pipeline

The `>` symbol is overloaded — it means both "greater than" in comparisons and "pipeline" in pipeline expressions. Pipe resolves this ambiguity by context:

- In a **comparison context** (inside `if`, `while`, as operand to `&&`/`||`, etc.), `>` is the greater-than operator.
- In an **expression context** at the top level or pipeline line, `>` is the pipeline operator.

```pipe
-- comparison: greater than
if x > 10
  print "big"

-- pipeline: pass x to double
x > double
```

When the intent is ambiguous, use parentheses to disambiguate:

```pipe
-- comparison first, then pipeline
result: (x > 5) > double
-- pipeline on 5? unlikely, but parsed this way
result: x > (5 > double)
```

The rule of thumb: **`>` in condition position is comparison; `>` in expression position is pipeline**.

## Pipeline vs Functional Composition

Pipelines and nested function calls produce the same result, but pipelines are often more readable:

### Nested function calls:

```pipe
not_empty: fn x
    (len x) > 0

print (sort (map (filter (split (read_file "data.txt") "\n") not_empty) upper))
```

### The same with pipelines:

```pipe
not_empty: fn x
    (len x) > 0

read_file "data.txt"
    > split _ "\n"
    > filter not_empty
    > map _ upper
    > sort
    > print
```

The pipeline version reads top-to-bottom in the order operations happen. The nested version reads inside-out, which is harder to follow as the chain grows.

## Multi-Stage Pipeline with Conditional Logic

Pipelines can be combined with `if` expressions for conditional transformations:

```pipe
fn process_data raw
  cleaned: raw
    > strip
    > lower

  if (len cleaned) > 100
    cleaned > truncate 100
  else
    cleaned

data: read_file "input.txt"
    > process_data
    > print
```

Or use a conditional inside a pipeline step:

```pipe
fn keep_passing s
    if s >= 60
        true
    else
        print "dropping " ++ (to_str s)
        false

scores: [85, 92, 45, 78, 63, 99]
    > filter keep_passing
    > sort
    > print
```

## Design Principle

Pipe's pipeline design follows a simple principle: **data flows downward; transformations are applied in sequence**. Each `>` line receives the result of the previous step, applies one transformation, and passes the result to the next step.

This makes Pipe programs read like a description:

```pipe
is_error: fn line
    has line "ERROR"

read_file "log.txt"
    > split _ "\n"
    > filter is_error
    > count
    > print
```

"This program reads a log file, splits it into lines, filters for ERROR lines, counts them, and prints the count." The code structure directly mirrors the problem description.

Key points about pipelines:

- The **first line** provides the initial data.
- Each **`>` line** transforms the data sequentially.
- Each **`>>` line** starts a transformation in the background and returns a Future.
- **Futures resolve automatically** when the value is consumed (arithmetic, function args, concatenation, etc.).
- The **`_` placeholder** gives precise control over argument placement.
- Pipelines **compose naturally** — a pipeline can be the initial value of another pipeline.
- Pipeline results can be assigned to **variables** for reuse.
- Both **vertical** (multi-line) and **horizontal** (single-line) syntax are equivalent; choose based on readability.
