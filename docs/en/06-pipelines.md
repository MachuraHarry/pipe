# Pipelines

## The Central Language Feature

The pipeline is the defining feature of Pipe. It allows data to flow through a sequence of transformations, expressed as a top-to-bottom chain of operations. This design makes data processing scripts read like a description of the data's journey rather than a sequence of nested function calls.

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
[1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
> filter (fn x x % 2 == 0)
> map (fn x x * 3)
> sort
> print

-- filter even numbers, multiply by 3, sort, print
-- output: [6, 12, 18, 24, 30]
```

The piped value (`[1, 2, ...]`) is inserted as the first argument to `filter`. The anonymous function is the second argument.

### The `_` Placeholder

Use `_` (underscore) to control where the piped value is inserted. Instead of always going as the first argument, `_` lets you place it at a specific position:

```pipe
"hello"
> replace _ "l" "x"       -- replace "hello" "l" "x"
> print

-- "hello" goes where _ is: replace("hello", "l", "x")
```

Without `_`, the value goes as the first argument. With `_`, the value goes exactly where `_` appears:

```pipe
[3, 1, 4, 1, 5]
> push _ 9              -- push([3, 1, 4, 1, 5], 9)
> len _
> print

-- output: 6
```

`_` can appear multiple times in a pipeline step — each occurrence receives the same piped value:

```pipe
"alpha"
> print _ _              -- prints "alpha alpha"
```

## File Processing Pipeline

A realistic example: reading a file, splitting into lines, filtering, sorting, and printing:

```pipe
read_file "data/words.txt"
> split _ "\n"
> filter (fn line (len line) > 0)     -- remove blank lines
> filter (fn line (has line "pipe"))
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
read_file "data/numbers.csv"
> split _ ","
> map _ to_num
> filter (fn n (is_num n) && (n > 0))
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
if x > 10        -- comparison: greater than
  print "big"

x > double       -- pipeline: pass x to double
```

When the intent is ambiguous, use parentheses to disambiguate:

```pipe
result: (x > 5) > double    -- comparison first, then pipeline
result: x > (5 > double)    -- pipeline on 5? unlikely, but parsed this way
```

The rule of thumb: **`>` in condition position is comparison; `>` in expression position is pipeline**.

## Pipeline vs Functional Composition

Pipelines and nested function calls produce the same result, but pipelines are often more readable:

### Nested function calls:

```pipe
print (sort (map (filter (split (read_file "data.txt") "\n") (fn x (len x) > 0)) upper))
```

### The same with pipelines:

```pipe
read_file "data.txt"
> split _ "\n"
> filter (fn x (len x) > 0)
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
scores: [85, 92, 45, 78, 63, 99]
> filter (fn s
    if s >= 60
      true
    else
      print "dropping " ++ (to_str s)
      false)
> sort
> print
```

## Design Principle

Pipe's pipeline design follows a simple principle: **data flows downward; transformations are applied in sequence**. Each `>` line receives the result of the previous step, applies one transformation, and passes the result to the next step.

This makes Pipe programs read like a description:

```pipe
read_file "log.txt"
> split _ "\n"
> filter (fn line (has line "ERROR"))
> count
> print
```

"This program reads a log file, splits it into lines, filters for ERROR lines, counts them, and prints the count." The code structure directly mirrors the problem description.

Key points about pipelines:

- The **first line** provides the initial data.
- Each **`>` line** transforms the data from above.
- The **`_` placeholder** gives precise control over argument placement.
- Pipelines **compose naturally** — a pipeline can be the initial value of another pipeline.
- Pipeline results can be assigned to **variables** for reuse.
- Both **vertical** (multi-line) and **horizontal** (single-line) syntax are equivalent; choose based on readability.
