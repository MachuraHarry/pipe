# 7. Data Structures

Pipe provides rich data structures — lists, maps, and strings — along with higher-order functions for functional-style data processing.

---

## 7.1 Lists

A list is an ordered, dynamically-sized sequence of values. Lists are heterogeneous: a single list can hold numbers, strings, maps, and even other lists.

### 7.1.1 Creating Lists

Lists are created with square brackets `[ ]`, elements separated by commas.

```pipe
let empty = []
let nums = [1, 2, 3, 4, 5]
let mixed = [42, "hello", true, nil, [1, 2]]
let nested = [[1, 2], [3, 4], [5, 6]]
```

### 7.1.2 Index Access

Pipe uses **0-based indexing**. Access elements with the `at` function or the built-in index syntax.

```pipe
let fruits = ["apple", "banana", "cherry"]

at fruits 0    # "apple"
at fruits 1    # "banana"
at fruits 2    # "cherry"
```

Negative indices are not supported. Accessing an out-of-bounds index returns `nil`.

```pipe
at fruits 99   # nil
at fruits -1   # nil (negative not supported)
```

### 7.1.3 Slicing with `start..end`

The `slice_list` function extracts a sublist using the `start..end` range syntax. The range is **inclusive of start, exclusive of end**.

```pipe
let nums = [10, 20, 30, 40, 50]

slice_list nums 0..3    # [10, 20, 30]
slice_list nums 1..4    # [20, 30, 40]
slice_list nums 2..5    # [30, 40, 50]
slice_list nums 0..5    # [10, 20, 30, 40, 50]  (full list)
slice_list nums 0..0    # []                      (empty slice)
```

If the range exceeds the list bounds, it is silently clamped:

```pipe
slice_list nums 3..99   # [40, 50]
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
let xs = [3, 1, 2]

len xs              # 3
push xs 4           # xs is now [3, 1, 2, 4]
pop xs              # returns 4, xs is now [3, 1, 2]
at xs 1             # 1
sort xs             # xs is now [1, 2, 3]

range 0 5           # [0, 1, 2, 3, 4]
range 5 10          # [5, 6, 7, 8, 9]
```

`range` is particularly useful for iteration:

```pipe
each range 0 10 fn(i) {
    print i
}
```

### 7.1.5 Higher-Order Functions

Pipe supports `map`, `filter`, `reduce`, and `each` for functional list processing. **In Tree-Walker mode**, these accept user-defined functions. **In Bytecode VM mode**, only built-in functions are accepted for `map`, `filter`, `reduce`.

#### `map(list, fn)`

Transforms each element, returning a new list.

```pipe
let double = fn(x) { x * 2 }
let nums = [1, 2, 3, 4, 5]

map nums double         # [2, 4, 6, 8, 10]

# Inline anonymous function
map nums fn(x) { x + 10 }   # [11, 12, 13, 14, 15]
```

#### `filter(list, fn)`

Returns a new list containing only elements for which the predicate is true.

```pipe
let is_even = fn(x) { x % 2 == 0 }
let nums = [1, 2, 3, 4, 5, 6]

filter nums is_even     # [2, 4, 6]

filter nums fn(x) { x > 3 }  # [4, 5, 6]
```

#### `reduce(list, fn, initial)`

Accumulates a value by applying the function to each element and the accumulator. The function receives `(accumulator, element)`.

```pipe
let add = fn(acc, x) { acc + x }
let nums = [1, 2, 3, 4, 5]

reduce nums add 0       # 15

# Product of all numbers
reduce nums fn(acc, x) { acc * x } 1   # 120
```

#### `each(list, fn)`

Iterates over every element, calling the function for its side effects. Returns `nil`.

```pipe
let print_item = fn(x) { print x }

each [1, 2, 3] print_item
# Output:
# 1
# 2
# 3

# With inline function
each ["a", "b", "c"] fn(x) {
    print "Item: " + x
}
```

**Important:** In Bytecode VM mode, `each` works with both built-in and user-defined functions. `map`, `filter`, and `reduce` only accept built-in functions in VM mode.

---

## 7.2 Maps

A map is an unordered collection of key-value pairs. Keys are strings; values can be any type.

### 7.2.1 Creating Maps

Maps are created with curly braces `{ }`, using `=` for key-value pairs.

```pipe
let person = {
    "name" = "Alice",
    "age" = 30,
    "city" = "New York"
}

let empty = {}

let nested = {
    "outer" = {
        "inner" = 42
    }
}
```

### 7.2.2 Accessing Values

Use the `get` function with key string:

```pipe
get person "name"       # "Alice"
get person "age"        # 30
get person "country"    # nil (key not found)
```

### 7.2.3 Dot-Notation Access

Pipe supports **dot-notation** as syntactic sugar for map access. `person.name` is equivalent to `get person "name"`.

```pipe
let person = { "name" = "Alice", "age" = 30 }

person.name     # "Alice"
person.age      # 30

# Nested access
let data = {
    "user" = {
        "profile" = {
            "email" = "alice@example.com"
        }
    }
}

data.user.profile.email     # "alice@example.com"
```

Dot-notation is read-only. Use `set` for modification.

### 7.2.4 Modifying Maps

The `set` function creates a new key-value pair or updates an existing one. It **mutates the map in place**.

```pipe
let person = { "name" = "Alice", "age" = 30 }

set person "city" "London"      # adds new key
set person "age" 31             # updates existing key

# person is now { "name" = "Alice", "age" = 31, "city" = "London" }
```

### 7.2.5 Keys and Values

```pipe
let m = { "a" = 1, "b" = 2, "c" = 3 }

keys m      # ["a", "b", "c"]
values m    # [1, 2, 3]
```

Note that map key ordering is not guaranteed; `keys` may return keys in any order.

---

## 7.3 Strings

Strings in Pipe are immutable sequences of characters. They support concatenation with `+` and character access via `at`.

### 7.3.1 Character Access

```pipe
let s = "Hello"

at s 0      # "H"
at s 1      # "e"
at s 4      # "o"
at s 99     # nil
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
upper "hello"           # "HELLO"
lower "WORLD"           # "world"
trim "  hi  "           # "hi"
split "a,b,c" ","       # ["a", "b", "c"]
join ["x", "y", "z"] "-"  # "x-y-z"
contains "hello world" "lo"  # true
len "hello"             # 5
```

---

## 7.4 `contains` for Strings and Lists

The `contains` function works on both strings and lists:

```pipe
# String containment (substring match)
contains "hello world" "world"    # true
contains "hello world" "xyz"      # false

# List containment (element match)
contains [1, 2, 3, 4] 3          # true
contains [1, 2, 3, 4] 99         # false
contains ["a", "b", "c"] "b"     # true
```

For lists, `contains` checks for exact element equality, not deep comparison of nested structures.

---

## 7.5 Practical Examples

### 7.5.1 Sum of a List

```pipe
let nums = [10, 20, 30, 40, 50]

# Using reduce
let total = reduce nums fn(acc, x) { acc + x } 0
print total     # 150

# Alternative: manual accumulation
let sum = fn(xs) {
    let total = 0
    each xs fn(x) {
        total = total + x
    }
    total
}
print sum nums  # 150
```

### 7.5.2 Iterative Processing

```pipe
let data = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]

# Get squares of even numbers
let even_squares = map
    (filter data fn(x) { x % 2 == 0 })
    fn(x) { x * x }

print even_squares      # [4, 16, 36, 64, 100]

# Chained with pipeline style
let result = pipe data
    | filter fn(x) { x > 3 }
    | map fn(x) { x * 2 }
    | sort

print result     # [8, 10, 12, 14, 16, 18, 20]
```

### 7.5.3 JSON Parsing

Pipe includes built-in JSON support via `parse_json` and `to_json`:

```pipe
let json_str = `{
    "users": [
        { "name": "Alice", "age": 30 },
        { "name": "Bob", "age": 25 }
    ],
    "count": 2
}`

let data = parse_json json_str

print data.count                    # 2
print data.users.0.name             # "Alice"  (dot-notation into list index)

# Transform the data
let names = map data.users fn(u) { u.name }
print names                         # ["Alice", "Bob"]

# Serialize back
print to_json data
```

### 7.5.4 Word Frequency Counter

```pipe
let text = "the cat and the dog and the bird"
let words = split text " "

# Build a frequency map
let freq = {}
each words fn(w) {
    let count = get freq w
    if count == nil {
        set freq w 1
    } {
        set freq w (count + 1)
    }
}

print keys freq     # ["the", "cat", "and", "dog", "bird"]
print freq.the      # 3
print freq.and      # 2
print freq.cat      # 1
```

### 7.5.5 Matrix Operations

```pipe
let matrix = [
    [1, 2, 3],
    [4, 5, 6],
    [7, 8, 9]
]

# Access element at row 1, col 2
at (at matrix 1) 2     # 6

# Sum of all elements
let flatten_sum = reduce matrix fn(acc, row) {
    acc + (reduce row fn(a, x) { a + x } 0)
} 0

print flatten_sum       # 45
```
