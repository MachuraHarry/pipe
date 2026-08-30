# 7. Data Structures

Pipe provides rich data structures — lists, maps, strings, and structs — along with higher-order functions for functional-style data processing.

---

## 7.1 Lists

A list is an ordered, dynamically-sized sequence of values. Lists are heterogeneous: a single list can hold numbers, strings, maps, and even other lists.

### 7.1.1 Creating Lists

Lists are created with square brackets `[ ]`, elements separated by commas.

```pipe
empty: []
nums: [1, 2, 3, 4, 5]
mixed: [42, "hello", true, nil, [1, 2]]
nested: [[1, 2], [3, 4], [5, 6]]
```

### 7.1.2 Index Access

Pipe uses **0-based indexing**. Access elements with the `at` function or the built-in index syntax.

```pipe
fruits: ["apple", "banana", "cherry"]

-- "apple"
at fruits 0
-- "banana"
at fruits 1
-- "cherry"
at fruits 2
```

Negative indices are not supported. Accessing an out-of-bounds index returns `nil`.

```pipe
-- nil
at fruits 99
-- nil (negative not supported)
at fruits -1
```

### 7.1.3 Slicing with `start..end`

The `slice_list` function extracts a sublist using the `start..end` range syntax. The range is **inclusive of start, exclusive of end**.

```pipe
nums: [10, 20, 30, 40, 50]

-- [10, 20, 30]
slice_list nums (range 0 3)
-- [20, 30, 40]
slice_list nums (range 1 4)
-- [30, 40, 50]
slice_list nums (range 2 5)
-- [10, 20, 30, 40, 50]  (full list)
slice_list nums (range 0 5)
-- []                      (empty slice)
slice_list nums (range 0 0)
```

If the range exceeds the list bounds, it is silently clamped:

```pipe
-- [40, 50]
slice_list nums (range 3 99)
```

### 7.1.4 List Operations

| Function | Signature | Description | Returns |
|----------|-----------|-------------|---------|
| `len` | `len(list)` | Number of elements | number |
| `push` | `push(list, value)` | Append value to end, mutate | list |
| `pop` | `pop(list)` | Remove and return last element | any |
| `at` | `at(list, index)` | Element at index (0-based) | any or nil |
| `sort` | `sort(list)` | Sort in-place ascending, mutate | list |
| `range` | `range(start, end)` | Create [start, start+1, ..., end-1] | list |
| `slice_list` | `slice_list(list, 0..3)` | Sublist by range | list |

```pipe
xs: [3, 1, 2]

-- 3
len xs
-- xs is now ([3, 1, 2, 4])
push xs 4
-- returns 4, xs is now ([3, 1, 2])
pop xs
-- 1
at xs 1
-- xs is now ([1, 2, 3])
sort xs

-- [0, 1, 2, 3, 4]
range 0 5
-- [5, 6, 7, 8, 9]
range 5 10
```

`range` is particularly useful for iteration:

```pipe
each range 0 10 fn i
    print i
```

### 7.1.5 Higher-Order Functions

Pipe supports `map`, `filter`, `reduce`, and `each` for functional list processing. **In Tree-Walker mode**, these accept user-defined functions. **In Bytecode VM mode**, only built-in functions are accepted for `map`, `filter`, `reduce`.

#### `map(list, fn)`

Transforms each element, returning a new list.

```pipe
double: (fn x
    x * 2)
nums: [1, 2, 3, 4, 5]

-- [2, 4, 6, 8, 10]
map nums double

-- Inline anonymous function
add_ten: fn x
    x + 10
-- [11, 12, 13, 14, 15]
map nums add_ten
```

#### `filter(list, fn)`

Returns a new list containing only elements for which the predicate is true.

```pipe
is_even: (fn x
    x % 2 == 0)
nums: [1, 2, 3, 4, 5, 6]

-- [2, 4, 6]
filter nums is_even

above_three: fn x
    x > 3

-- [4, 5, 6]
filter nums above_three
```

#### `reduce(list, fn, initial)`

Accumulates a value by applying the function to each element and the accumulator. The function receives `(accumulator, element)`.

```pipe
add: (fn acc x
    acc + x)
nums: [1, 2, 3, 4, 5]

-- 15
reduce nums add 0

-- Product of all numbers
reduce nums fn acc x
    -- 120
        acc * x  1
```

#### `each(list, fn)`

Iterates over every element, calling the function for its side effects. Returns `nil`.

```pipe
print_item: (fn x
    print x)

each ([1, 2, 3]) print_item
-- Output:
-- 1
-- 2
-- 3

-- With inline function
each (["a", "b", "c"]) fn x
    print "Item: " + x
```

**Important:** In Bytecode VM mode, `each` works with both built-in and user-defined functions. `map`, `filter`, and `reduce` only accept built-in functions in VM mode.

---

## 7.2 Maps

A map is an unordered collection of key-value pairs. Keys are strings; values can be any type.

### 7.2.1 Creating Maps

Maps are created with curly braces `{ }`, using `=` for key-value pairs.

```pipe
person: {name: "Alice", age: 30, city: "New York"}

empty: {}

nested: {outer: {inner: 42}}
```

### 7.2.2 Accessing Values

Use the `get` function with key string:

```pipe
-- "Alice"
get person "name"
-- 30
get person "age"
-- nil (key not found)
get person "country"
```

### 7.2.3 Dot-Notation Access

Pipe supports **dot-notation** as syntactic sugar for map access. `person.name` is equivalent to `get person "name"`.

```pipe
person: {name: "Alice", age: 30}

-- "Alice"
person.name
-- 30
person.age

-- Nested access
data: {user: {profile: {email: "alice@example.com"}}}

-- "alice@example.com"
data.user.profile.email
```

Dot-notation is read-only. Use `set` for modification.

### 7.2.4 Modifying Maps

The `set` function creates a new key-value pair or updates an existing one. It **mutates the map in place**.

```pipe
person: { name: "Alice", age: 30 }

-- adds new key
set person "city" "London"
-- updates existing key
set person "age" 31

-- person is now { name: "Alice", age: 31, city: "London" }
```

### 7.2.5 Keys and Values

```pipe
m: { a: 1, b: 2, c: 3 }

-- ["a", "b", "c"]
keys m
-- [1, 2, 3]
values m
```

`keys` and `values` follow the map's key order. Map literals keep their declaration order, so `keys m` returns the keys in the order you wrote them. Maps built programmatically (for example HTTP headers or MCP tool arguments) are sorted alphabetically by key for determinism. Here `keys m` returns `["a", "b", "c"]` and `values m` returns `[1, 2, 3]`.

---

## 7.3 Strings

Strings in Pipe are immutable sequences of characters. They support concatenation with `+` and character access via `at`.

### 7.3.1 Character Access

```pipe
s: "Hello"

-- "H"
at s 0
-- "e"
at s 1
-- "o"
at s 4
-- nil
at s 99
```

### 7.3.2 String Operations

| Function | Signature | Description |
|----------|-----------|-------------|
| `upper` | `upper(str)` | Convert to uppercase |
| `lower` | `lower(str)` | Convert to lowercase |
| `trim` | `trim(str)` | Remove leading/trailing whitespace |
| `split` | `split(str, delim)` | Split into list by delimiter |
| `join` | `join(list, delim)` | Join list elements with delimiter |
| `contains` | `contains(str, substr)` | Check if substring exists |
| `len` | `len(str)` | Number of characters |

```pipe
-- "HELLO"
upper "hello"
-- "world"
lower "WORLD"
-- "hi"
trim "  hi  "
-- ["a", "b", "c"]
split "a,b,c" ","
-- "x-y-z"
join (["x", "y", "z"]) "-"
-- true
contains "hello world" "lo"
-- 5
len "hello"
```

---

## 7.4 `contains` for Strings and Lists

The `contains` function works on both strings and lists:

```pipe
-- String containment (substring match)
-- true
contains "hello world" "world"
-- false
contains "hello world" "xyz"

-- List containment (element match)
-- true
contains ([1, 2, 3, 4]) 3
-- false
contains ([1, 2, 3, 4]) 99
-- true
contains (["a", "b", "c"]) "b"
```

For lists, `contains` checks for exact element equality, not deep comparison of nested structures.

---

## 7.5 Practical Examples

### 7.5.1 Sum of a List

```pipe
nums: [10, 20, 30, 40, 50]

-- Using reduce
add_fn: fn acc x
    acc + x
total: reduce nums add_fn 0
-- 150
print total

-- Alternative: manual accumulation
sum: fn xs
    total: 0
    for x in xs
        total: total + x
    total
-- 150
print sum nums
```

### 7.5.2 Iterative Processing

```pipe
data: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]

-- Get squares of even numbers
is_even: fn x
    x % 2 == 0
square: fn x
    x * x

evens: filter data is_even
even_squares: map evens square

-- [4, 16, 36, 64, 100]
print even_squares

above_three: fn x
    x > 3
double: fn x
    x * 2

-- Chained with pipeline style
data
    > filter above_three
    > map double
    > sort
    > print
```

### 7.5.3 JSON Parsing

Pipe includes built-in JSON support via `parse_json` and `to_json`:

```pipe
json_str: `{
    users: [
        { name: "Alice", age: 30 },
        { name: "Bob", age: 25 }
    ],
    count: 2
}`

data: parse_json json_str

-- 2
print data.count
-- "Alice"  (dot-notation into list index)
first_user: at data.users 0
print first_user.name

-- Transform the data
get_name: fn u
    u.name
names: map data.users get_name
-- ["Alice", "Bob"]
print names

-- Serialize back
print to_json data
```

### 7.5.4 Word Frequency Counter

```pipe
text: "the cat and the dog and the bird"
words: split text " "

-- Build a frequency map
freq: {}
each words fn w
    count: get freq w
    if count == nil
        set freq w 1
    else
        set freq w (count + 1)

-- ["the", "cat", "and", "dog", "bird"]
print keys freq
-- 3
print freq.the
-- 2
print freq.and
-- 1
print freq.cat
```

### 7.5.5 Matrix Operations

```pipe
matrix: [[1, 2, 3], [4, 5, 6], [7, 8, 9]]

-- Access element at row 1, col 2
-- 6
at (at matrix 1) 2

-- Sum of all elements
sum_row: fn acc row
    row_sum: fn a x
        a + x
    acc + (reduce row row_sum 0)
flatten_sum: reduce matrix sum_row 0

-- 45
print flatten_sum
```

## 7.6 Structs

Structs are user-defined record types that group related named fields into a single value. Unlike maps, structs have a fixed set of fields defined at compile time, and fields are accessed via dot notation.

### 7.6.1 Defining Structs

Structs are defined with the `struct` keyword. There are two forms:

**Block form** — fields listed in an indented block, one per line. Each field can have an optional default value:

```pipe
struct Point
    x
    y

struct Person
    name: "Unknown"
    age: 0
    active: true
```

**Inline form** — compact, comma-separated field names. No default values:

```pipe
struct Point: x, y
struct Color: red, green, blue, alpha
```

Fields are stored in definition order. Default values are evaluated at struct definition time.

### 7.6.2 Creating Instances

Struct instances are created by calling the struct name as a constructor with space-separated positional arguments matching the field order:

```pipe
p: Point 10 20
person: Person "Alice" 30 true
c: Color 255 128 0 255
```

Arguments are applied positionally. Any fields not provided use their default value (if one was specified) or remain unset.

### 7.6.3 Field Access

Fields are accessed with dot notation:

```pipe
p.x        -- 10
person.name  -- "Alice"
person.age   -- 30
```

Dot access is checked at runtime:
- Accessing a field that doesn't exist on the struct produces an error
- Dot access on non-struct, non-map values produces an error

```pipe
p.z        -- ERROR: struct Point has no field 'z'
42.x       -- ERROR: cannot use .x on INTEGER

-- Dot access also works on maps (existing behavior):
m: {a: 1, b: 2}
m.a        -- 1
```

### 7.6.4 Structs vs Maps

| Feature | Struct | Map |
|---------|--------|-----|
| Field set | Fixed at definition | Dynamic, can add/remove keys |
| Access | `p.x` (dot notation) | `m.x` or `get m "x"` |
| Creation | `Point 1 2` (positional) | `{x: 1, y: 2}` |
| Type identity | Named (Point, Person, etc.) | Anonymous (`MAP`) |
| Default values | Per-field, defined once | No built-in defaults |
| Use case | Fixed-schema data records | Dynamic key-value data, ad-hoc structures |

Use structs when you know the shape of your data ahead of time (e.g., a 2D point, a configuration record, API response shape). Use maps when keys are determined at runtime (e.g., parsing unknown JSON, caching named results).

