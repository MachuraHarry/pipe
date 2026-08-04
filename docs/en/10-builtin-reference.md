# 10. Built-in Function Reference

Pipe includes 115 built-in functions organized by category. This chapter documents each function with its signature, description, return type, and usage example.

---

## 10.1 IO & System (6 functions)

### `print`
**Signature:** `print(value ...)`
**Description:** Prints one or more values to stdout, separated by spaces and followed by a newline.
**Returns:** `nil`
```pipe
-- Hello
print "Hello"
-- The answer is 42
print "The answer is" 42
-- x: 10 y: 20
print "x:" x "y:" y
```

### `input`
**Signature:** `input(prompt)`
**Description:** Displays `prompt` (optional), reads a line from stdin, and returns it as a string.
**Returns:** `string`
```pipe
name: input "Enter your name: "
print "Hello, " ++ name
```

### `exec`
**Signature:** `exec(command)`
**Description:** Executes a system command via the shell and returns the combined stdout/stderr output.
**Returns:** `string`
```pipe
files: exec "ls -la"
print files

version: exec "git --version"
print version
```

### `env`
**Signature:** `env(name)`
**Description:** Returns the value of the environment variable `name`, or `nil` if not set.
**Returns:** `string` or `nil`
```pipe
home: env "HOME"
print "Home directory: " ++ home

path: env "PATH"
print path
```

### `sleep`
**Signature:** `sleep(ms)`
**Description:** Pauses execution for `ms` milliseconds.
**Returns:** `nil`
```pipe
-- wait 1 second
sleep 1000
print "Done waiting"

-- wait 0[5] seconds
sleep 500
```

### `go`
**Signature:** `go(fn, args...)`
**Description:** Runs `fn` asynchronously in a goroutine with the given arguments.
**Returns:** `nil`
```pipe
go print "concurrent"
print "main"
```

---

## 10.2 File System (17 functions)

### `read_file`
**Signature:** `read_file(path)`
**Description:** Reads the entire contents of a file and returns it as a string. Errors if the file does not exist.
**Returns:** `string`
```pipe
content: read_file "config.json"
print content
```

### `write_file`
**Signature:** `write_file(path, content)`
**Description:** Writes `content` to `path`, overwriting if it exists. Creates parent directories if needed.
**Returns:** `nil`
```pipe
write_file "output.txt" "Hello, world!"
```

### `append_file`
**Signature:** `append_file(path, content)`
**Description:** Appends `content` to the end of the file at `path`. Creates the file if it doesn't exist.
**Returns:** `nil`
```pipe
append_file "log.txt" "New log entry\n"
```

### `read_lines`
**Signature:** `read_lines(path)`
**Description:** Reads a file and returns its lines as a list of strings (without trailing newlines).
**Returns:** `list` of strings
```pipe
lines: read_lines "data.csv"
each lines fn line
    print "Line: " ++ line
```

### `file_exists`
**Signature:** `file_exists(path)`
**Description:** Returns `true` if the file or directory at `path` exists, `false` otherwise.
**Returns:** `boolean`
```pipe
if file_exists "config.json"
    print "Config found"
else
    print "Config missing"
```

### `file_delete`
**Signature:** `file_delete(path)`
**Description:** Deletes the file at `path`. Returns `true` on success.
**Returns:** `boolean`
```pipe
file_delete "temp.txt"
```

### `file_move`
**Signature:** `file_move(src, dst)`
**Description:** Moves or renames a file from `src` to `dst`.
**Returns:** `nil`
```pipe
file_move "old_name.txt" "new_name.txt"
```

### `file_copy`
**Signature:** `file_copy(src, dst)`
**Description:** Copies a file from `src` to `dst`.
**Returns:** `nil`
```pipe
file_copy "template.txt" "new_file.txt"
```

### `file_size`
**Signature:** `file_size(path)`
**Description:** Returns the size of the file in bytes.
**Returns:** `number`
```pipe
size: file_size "data.bin"
print "File size: " ++ size ++ " bytes"
```

### `file_type`
**Signature:** `file_type(path)`
**Description:** Returns `"file"`, `"dir"`, or `nil` if the path does not exist.
**Returns:** `string` or `nil`
```pipe
t: file_type "config.json"
if t == "dir"
    print "It's a directory"
```

### `list_dir`
**Signature:** `list_dir(path)`
**Description:** Returns a list of filenames in the directory at `path`.
**Returns:** `list` of strings
```pipe
files: list_dir "."
print_file: fn f
    print f
each files print_file
```

### `make_dir`
**Signature:** `make_dir(path)`
**Description:** Creates a new directory. Creates parent directories if `path` includes intermediate dirs.
**Returns:** `nil`
```pipe
make_dir "output/reports"
```

### `remove_dir`
**Signature:** `remove_dir(path)`
**Description:** Removes an empty directory. Fails if the directory is not empty.
**Returns:** `nil`
```pipe
remove_dir "output/temp"
```

### `path_join`
**Signature:** `path_join(base, part)`
**Description:** Joins two path components with the OS-appropriate separator.
**Returns:** `string`
```pipe
full: path_join "/home/user" "docs"
-- "/home/user/docs"
print full
```

### `path_base`
**Signature:** `path_base(path)`
**Description:** Returns the last component of a path (the filename or directory name).
**Returns:** `string`
```pipe
-- "file.txt"
path_base "/home/user/file.txt"
-- "user"
path_base "/home/user/"
```

### `path_dir`
**Signature:** `path_dir(path)`
**Description:** Returns the directory portion of a path, without the final component.
**Returns:** `string`
```pipe
-- "/home/user"
path_dir "/home/user/file.txt"
-- "."
path_dir "file.txt"
```

### `path_ext`
**Signature:** `path_ext(path)`
**Description:** Returns the file extension including the dot, or empty string if none.
**Returns:** `string`
```pipe
-- ".txt"
path_ext "file.txt"
-- ".gz"
path_ext "archive.tar.gz"
-- ""
path_ext "Makefile"
```

---

## 10.3 String (6 functions)

### `upper`
**Signature:** `upper(str)`
**Description:** Returns a copy of `str` with all characters converted to uppercase.
**Returns:** `string`
```pipe
-- "HELLO"
upper "hello"
-- "HELLO WORLD"
upper "Hello World"
```

### `lower`
**Signature:** `lower(str)`
**Description:** Returns a copy of `str` with all characters converted to lowercase.
**Returns:** `string`
```pipe
-- "hello"
lower "HELLO"
-- "hello world"
lower "Hello World"
```

### `trim`
**Signature:** `trim(str)`
**Description:** Returns a copy of `str` with leading and trailing whitespace removed.
**Returns:** `string`
```pipe
-- "hello"
trim "  hello  "
-- "indented"
trim "\t indented\n"
```

### `split`
**Signature:** `split(str, delimiter)`
**Description:** Splits `str` into a list of substrings separated by `delimiter`.
**Returns:** `list` of strings
```pipe
-- ["a", "b", "c"]
split "a,b,c" ","
-- ["one", "two", "three"]
split "one two three" " "
-- ["h", "e", "l", "l", "o"]
split "hello" ""
```

### `join`
**Signature:** `join(list, delimiter)`
**Description:** Joins list elements into a string with `delimiter` between each element.
**Returns:** `string`
```pipe
-- "a,b,c"
join (["a", "b", "c"]) ","
-- "one - two"
join (["one", "two"]) " - "
-- "123"
join ([1, 2, 3]) ""
```

### `contains`
**Signature:** `contains(haystack, needle)`
**Description:** For strings: checks if `needle` is a substring. For lists: checks if `needle` is an element.
**Returns:** `boolean`
```pipe
-- true
contains "hello world" "lo"
-- false
contains "hello world" "xyz"
-- true
contains ([1, 2, 3]) 2
-- false
contains (["a", "b"]) "c"
```

---

## 10.4 List (11 functions)

### `len`
**Signature:** `len(value)`
**Description:** Returns the length of a string (characters), list (elements), or map (keys).
**Returns:** `number`
```pipe
-- 5
len "hello"
-- 3
len ([1, 2, 3])
-- 1
len { a: 1 }
```

### `push`
**Signature:** `push(list, value)`
**Description:** Appends `value` to the end of `list`. **Mutates the list in place.**
**Returns:** The mutated `list`
```pipe
xs: [1, 2]
-- xs is now ([1, 2, 3])
push xs 3
-- xs is now ([1, 2, 3, "hello"])
push xs "hello"
```

### `pop`
**Signature:** `pop(list)`
**Description:** Removes and returns the last element of `list`. **Mutates the list in place.**
**Returns:** The removed element, or `nil` if empty
```pipe
xs: [1, 2, 3]
-- last = 3, xs is now ([1, 2])
last: pop xs
-- nil
pop []
```

### `at`
**Signature:** `at(collection, index)`
**Description:** Returns the element at 0-based `index` in a list or string. Returns `nil` if out of bounds.
**Returns:** `any` or `nil`
```pipe
-- 20
at ([10, 20, 30]) 1
-- "h"
at "hello" 0
-- nil
at ([1, 2]) 99
```

### `slice_list`
**Signature:** `slice_list(list, range)`
**Description:** Returns a sublist from `start` to `end` (exclusive). Uses `start..end` syntax.
**Returns:** `list`
```pipe
-- [10, 20, 30]
slice_list ([10, 20, 30, 40, 50]) (range 0 3)
-- [30, 40, 50]
slice_list ([10, 20, 30, 40, 50]) (range 2 5)
-- []
slice_list ([10, 20, 30]) (range 1 1)
```

### `sort`
**Signature:** `sort(list)`
**Description:** Sorts `list` in ascending order **in place**. For strings, sorts lexicographically. For numbers, sorts numerically.
**Returns:** The mutated `list`
```pipe
xs: [3, 1, 4, 1, 5]
-- xs is now ([1, 1, 3, 4, 5])
sort xs

ys: ["b", "a", "c"]
-- ys is now (["a", "b", "c"])
sort ys
```

### `range`
**Signature:** `range(start, end)`
**Description:** Creates a list of numbers from `start` (inclusive) to `end` (exclusive).
**Returns:** `list` of numbers
```pipe
-- [0, 1, 2, 3, 4]
range 0 5
-- [5, 6, 7, 8, 9]
range 5 10
-- []
range 0 0
```

### `map`
**Signature:** `map(list, fn)`
**Description:** Applies `fn` to each element and returns a new list of results. In VM mode, only built-in functions are accepted.
**Returns:** `list`
```pipe
double: fn x
    x * 2
map ([1, 2, 3]) double
    -- [2, 4, 6]

to_upper: fn s
    upper s
map (["a", "b"]) to_upper
    -- ["A", "B"]
```

### `filter`
**Signature:** `filter(list, fn)`
**Description:** Returns a new list containing only elements where `fn(element)` returns truthy. In VM mode, only built-in functions are accepted.
**Returns:** `list`
```pipe
above_two: fn x
    x > 2

filter ([1, 2, 3, 4]) above_two
    -- [3, 4]

identity: fn x
    x
filter ([0, 1, 0, 3]) identity
    -- [1, 3] (truthy elements)
```

### `reduce`
**Signature:** `reduce(list, fn, initial)`
**Description:** Accumulates a value by calling `fn(accumulator, element)` for each element. `initial` is the starting accumulator. In VM mode, only built-in functions are accepted.
**Returns:** `any`
```pipe
reduce ([1, 2, 3]) fn acc x
    -- 6
        acc + x  0
reduce ([2, 3, 4]) fn acc x
    -- 24
        acc * x  1
```

### `each`
**Signature:** `each(list, fn)`
**Description:** Calls `fn(element)` for each element in `list`. Used for side effects. Works with both built-in and user functions in all modes.
**Returns:** `nil`
```pipe
print_x: fn x
    print x
each ([1, 2, 3]) print_x
-- 1
-- 2
-- 3
```

---

## 10.5 Map (4 functions)

### `get`
**Signature:** `get(map, key)`
**Description:** Returns the value associated with `key` in `map`, or `nil` if key not found.
**Returns:** `any` or `nil`
```pipe
m: { name: "Alice", age: 30 }
-- "Alice"
get m "name"
-- nil
get m "country"
```

### `set`
**Signature:** `set(map, key, value)`
**Description:** Sets `key` to `value` in `map`. Creates new key or updates existing. **Mutates the map in place.**
**Returns:** The mutated `map`
```pipe
m: { name: "Alice" }
-- adds key
set m "age" 30
-- updates key
set m "name" "Bob"
-- m is now { name: "Bob", age: 30 }
```

### `keys`
**Signature:** `keys(map)`
**Description:** Returns a list of all keys in `map`. Order is not guaranteed.
**Returns:** `list` of strings
```pipe
m: { a: 1, b: 2, c: 3 }
-- ["a", "b", "c"] (order may vary)
keys m
```

### `values`
**Signature:** `values(map)`
**Description:** Returns a list of all values in `map`. Order corresponds to `keys` order.
**Returns:** `list`
```pipe
m: { a: 1, b: 2, c: 3 }
-- [1, 2, 3] (order corresponds to keys)
values m
```

---

## 10.6 Math (6 functions)

### `abs`
**Signature:** `abs(x)`
**Description:** Returns the absolute value of `x`.
**Returns:** `number`
```pipe
-- 5
abs -5
-- 42
abs 42
-- 0
abs 0
```

### `min`
**Signature:** `min(a, b)`
**Description:** Returns the smaller of `a` and `b`.
**Returns:** `number`
```pipe
-- 10
min 10 20
-- -5
min -5 0
-- 3[0]
min 3[14] 3[0]
```

### `max`
**Signature:** `max(a, b)`
**Description:** Returns the larger of `a` and `b`.
**Returns:** `number`
```pipe
-- 20
max 10 20
-- 0
max -5 0
-- 3[14]
max 3[14] 3[0]
```

### `pow`
**Signature:** `pow(base, exp)`
**Description:** Returns `base` raised to the power of `exp`.
**Returns:** `number`
```pipe
-- 1024
pow 2 10
-- 1.414... (square root)
pow 2 0[5]
-- 1000
pow 10 3
```

### `sqrt`
**Signature:** `sqrt(x)`
**Description:** Returns the square root of `x`.
**Returns:** `number`
```pipe
-- 10
sqrt 100
-- 1.414...
sqrt 2
-- 0
sqrt 0
```

### `round`
**Signature:** `round(x)`
**Description:** Rounds `x` to the nearest integer. Half values round to the nearest even integer (banker's rounding).
**Returns:** `number`
```pipe
-- 3
round 3[14]
-- 4
round 3[5]
-- 2 (banker's rounding)
round 2[5]
-- 5
round 4[7]
```

---

## 10.7 Network & HTTP (5 functions)

### `http_get`
**Signature:** `http_get(url)`
**Description:** Performs an HTTP GET request to `url` and returns the response body as a string.
**Returns:** `string`
```pipe
body: http_get "https://api.example.com/data"
print body
```

### `http_post`
**Signature:** `http_post(url, body)`
**Description:** Performs an HTTP POST request to `url` with `body` as the request payload. Returns the response body.
**Returns:** `string`
```pipe
resp: http_post "https://api.example.com/submit" "{\"key\": \"value\"}"
print resp
```

### `http_get_json`
**Signature:** `http_get_json(url)`
**Description:** Performs an HTTP GET request to `url` and parses the response as JSON.
**Returns:** `any` (map, list, number, string, boolean, or nil)
```pipe
data: http_get_json "https://api.example.com/users"
print data.count
print_name: fn u
    print u.name
each data.users print_name
```

### `parse_json`
**Signature:** `parse_json(json_string)`
**Description:** Parses a JSON string into Pipe data structures (maps, lists, numbers, strings, booleans, nil).
**Returns:** `any` or `nil` on parse error
```pipe
obj: parse_json `{name: "Alice", scores: [95, 87, 92]}`
-- "Alice"
print obj.name
-- 87
print obj.scores[1]
```

### `to_json`
**Signature:** `to_json(value)`
**Description:** Serializes a Pipe value into a JSON string.
**Returns:** `string`
```pipe
data: { name: "Alice", age: 30 }
json_str: to_json data
-- {name: "Alice",age: 30}
print json_str
```

---

## 10.8 TCP (6 functions)

### `tcp_listen`
**Signature:** `tcp_listen(address)`
**Description:** Starts a TCP server listening on `address` (e.g., `":8080"` or `"localhost:3000"`). Returns a listener handle.
**Returns:** `listener handle`
```pipe
listener: tcp_listen ":8080"
print "Server listening on port 8080"
```

### `tcp_connect`
**Signature:** `tcp_connect(address)`
**Description:** Connects to a TCP server at `address` and returns a connection handle.
**Returns:** `connection handle`
```pipe
conn: tcp_connect "localhost:8080"
tcp_write conn "Hello, server!"
```

### `tcp_accept`
**Signature:** `tcp_accept(listener)`
**Description:** Accepts an incoming connection on a listener. Blocks until a client connects. Returns a connection handle.
**Returns:** `connection handle`
```pipe
conn: tcp_accept listener
msg: tcp_read conn 1024
print "Received: " ++ msg
```

### `tcp_read`
**Signature:** `tcp_read(conn, max_bytes)`
**Description:** Reads up to `max_bytes` from a TCP connection and returns the data as a string.
**Returns:** `string`
```pipe
data: tcp_read conn 4096
print data
```

### `tcp_write`
**Signature:** `tcp_write(conn, data)`
**Description:** Writes `data` to a TCP connection.
**Returns:** `nil`
```pipe
tcp_write conn "HTTP/1[1] 200 OK\r\n\r\nHello"
```

### `tcp_close`
**Signature:** `tcp_close(handle)`
**Description:** Closes a TCP connection or listener.
**Returns:** `nil`
```pipe
tcp_close conn
tcp_close listener
```

### TCP Server Example

```pipe
listener: tcp_listen ":9999"
print "Echo server on :9999"

each range 0 5 fn i
    conn: tcp_accept listener
    msg: tcp_read conn 1024
    print "Got: " ++ msg
    tcp_write conn "Echo: " ++ msg
    tcp_close conn

tcp_close listener
```

---

## 10.9 Regex (2 functions)

### `regex_match`
**Signature:** `regex_match(pattern, str)`
**Description:** Returns `true` if `str` matches the regex `pattern`, `false` otherwise.
**Returns:** `boolean`
```pipe
-- true
regex_match "^[a-z]+$" "hello"
-- false
regex_match "^[a-z]+$" "hello123"
-- true
regex_match "\\d{3}-\\d{4}" "555-1234"
```

### `regex_replace`
**Signature:** `regex_replace(pattern, replacement, str)`
**Description:** Replaces all occurrences of `pattern` in `str` with `replacement`.
**Returns:** `string`
```pipe
-- "hello-world"
regex_replace "\\s+" "-" "hello world"
-- "h*ll*"
regex_replace "[aeiou]" "*" "hello"
-- "abc###xyz"
regex_replace "\\d" "#" "abc123xyz"
```

---

## 10.10 Date & Time (2 functions)

### `now`
**Signature:** `now()`
**Description:** Returns the current time as a Unix timestamp in seconds (floating point, includes fractional seconds).
**Returns:** `number`
```pipe
t: now
print "Current timestamp: " ++ t
-- e.g. 1700000000.123456
```

### `format_time`
**Signature:** `format_time(timestamp, layout)`
**Description:** Formats a Unix timestamp into a human-readable string using Go's reference time for the layout. The reference time is `Mon Jan 2 15:04:05 MST 2006`, which is expressed as `01/02 03:04:05PM '06 -0700`.

**Format layout reference:**
| Pattern | Meaning | Example |
|---------|---------|---------|
| `2006` | 4-digit year | 2024 |
| `06` | 2-digit year | 24 |
| `01` | Month (2-digit) | 01-12 |
| `Jan` | Month (abbreviated) | Jan-Dec |
| `January` | Month (full) | January-December |
| `02` | Day (2-digit) | 01-31 |
| `Mon` | Weekday (abbreviated) | Mon-Sun |
| `Monday` | Weekday (full) | Monday-Sunday |
| `15` | Hour (24-hour) | 00-23 |
| `03` | Hour (12-hour) | 01-12 |
| `04` | Minute | 00-59 |
| `05` | Second | 00-59 |
| `PM` | AM/PM marker | AM or PM |
| `MST` | Timezone (abbrev) | EST, UTC |
| `-0700` | Timezone offset | -0500, +0000 |

**Returns:** `string`
```pipe
t: now

-- "2024-07-15"
format_time t "2006-01-02"
-- "2024-07-15 14:30:00"
format_time t "2006-01-02 15:04:05"
-- "Mon Jul 15 14:30:00 2024"
format_time t "Mon Jan 2 15:04:05 2006"
-- "02:30 PM"
format_time t "03:04 PM"
-- "Monday, July 15, 2024"
format_time t "Monday, January 2, 2006"
```

---

## 10.11 Random (2 functions)

### `random`
**Signature:** `random()`
**Description:** Returns a random floating-point number in the range `[0.0, 1.0)`.
**Returns:** `number`
```pipe
-- e.g. 0[734291]
r: random
print r
```

### `random_range`
**Signature:** `random_range(min, max)`
**Description:** Returns a random integer in the range `[min, max]` inclusive.
**Returns:** `number` (integer)
```pipe
-- 1, 2, 3, 4, 5, or 6
dice: random_range 1 6
-- 0 or 1
coin: random_range 0 1
```

---

## 10.12 Encoding (2 functions)

### `base64_encode`
**Signature:** `base64_encode(str)`
**Description:** Encodes a string to Base64.
**Returns:** `string`
```pipe
-- "aGVsbG8="
base64_encode "hello"
-- "UGlwZQ=="
base64_encode "Pipe"
```

### `base64_decode`
**Signature:** `base64_decode(str)`
**Description:** Decodes a Base64-encoded string.
**Returns:** `string`
```pipe
-- "hello"
base64_decode "aGVsbG8="
-- "Pipe"
base64_decode "UGlwZQ=="
```

---

## 10.13 Type Checks (6 functions)

### `type_of`
**Signature:** `type_of(value)`
**Description:** Returns a string indicating the type of `value`.
**Returns:** `string` — one of `"number"`, `"string"`, `"list"`, `"map"`, `"nil"`, `"function"`, `"boolean"`, `"result"`
```pipe
-- "number"
type_of 42
-- "string"
type_of "hello"
-- "list"
type_of ([1, 2])
-- "map"
type_of { a: 1 }
-- "nil"
type_of nil
type_of: (fn x
    x)-- "function"
-- "boolean"
type_of true
-- "result"
type_of (Ok 1)
```

### `is_num`
**Signature:** `is_num(value)`
**Description:** Returns `true` if `value` is a number.
**Returns:** `boolean`
```pipe
-- true
is_num 42
-- true
is_num 3[14]
-- false
is_num "42"
-- false
is_num nil
```

### `is_str`
**Signature:** `is_str(value)`
**Description:** Returns `true` if `value` is a string.
**Returns:** `boolean`
```pipe
-- true
is_str "hello"
-- false
is_str 42
-- true
is_str ""
```

### `is_list`
**Signature:** `is_list(value)`
**Description:** Returns `true` if `value` is a list.
**Returns:** `boolean`
```pipe
-- true
is_list ([1, 2])
-- true
is_list []
-- false
is_list "ab"
```

### `is_map`
**Signature:** `is_map(value)`
**Description:** Returns `true` if `value` is a map.
**Returns:** `boolean`
```pipe
-- true
is_map { a: 1 }
-- true
is_map {}
-- false
is_map ([1, 2])
```

### `is_nil`
**Signature:** `is_nil(value)`
**Description:** Returns `true` if `value` is `nil`.
**Returns:** `boolean`
```pipe
-- true
is_nil nil
-- false
is_nil 0
-- false
is_nil ""
-- false
is_nil false
```

---

## 10.14 Conversion (2 functions)

### `to_str`
**Signature:** `to_str(value)`
**Description:** Converts `value` to its string representation. Numbers, booleans, and nil all convert to strings.
**Returns:** `string`
```pipe
-- "42"
to_str 42
-- "3[14]"
to_str 3[14]
-- "true"
to_str true
-- "nil"
to_str nil
-- "[1, 2, 3]"
to_str ([1, 2, 3])
```

### `to_num`
**Signature:** `to_num(str)`
**Description:** Parses a string into a number. Returns `nil` if parsing fails.
**Returns:** `number` or `nil`
```pipe
-- 42
to_num "42"
-- 3[14]
to_num "3[14]"
-- nil
to_num "hello"
-- nil (only decimal)
to_num "0xFF"
```

---

## 10.15 Result Type (6 functions)

### `Ok`
**Signature:** `Ok(value)`
**Description:** Creates a successful `Result` containing `value`.
**Returns:** `Result` (Ok variant)
```pipe
r: Ok 42
-- true
is_ok r
-- 42
unwrap r
```

### `Err`
**Signature:** `Err(message)`
**Description:** Creates a failed `Result` containing the error `message` string.
**Returns:** `Result` (Err variant)
```pipe
r: Err "something went wrong"
-- true
is_err r
-- ERROR: called unwrap on an Err value
unwrap r
```

### `is_ok`
**Signature:** `is_ok(result)`
**Description:** Returns `true` if `result` is an `Ok` variant.
**Returns:** `boolean`
```pipe
-- true
is_ok (Ok 42)
-- false
is_ok (Err "oops")
```

### `is_err`
**Signature:** `is_err(result)`
**Description:** Returns `true` if `result` is an `Err` variant.
**Returns:** `boolean`
```pipe
-- false
is_err (Ok 42)
-- true
is_err (Err "oops")
```

### `unwrap`
**Signature:** `unwrap(result)`
**Description:** Returns the value inside `Ok`, or raises an error if called on `Err`.
**Returns:** `any`
```pipe
-- 42
unwrap (Ok 42)
-- ERROR: called unwrap on an Err value
unwrap (Err "fail")
```

### `unwrap_or`
**Signature:** `unwrap_or(result, default)`
**Description:** Returns the value inside `Ok`, or `default` if `result` is `Err`.
**Returns:** `any`
```pipe
-- 42
unwrap_or (Ok 42) 0
-- 0
unwrap_or (Err "x") 0
-- "hi"
unwrap_or (Ok "hi") ""
```
---

## 10.16 AI — Configuration (5 functions)

### `ai_provider`

**Signature:** `ai_provider(name)`

**Description:** Sets the AI provider to `name`. Supported: `"openai"`, `"anthropic"`, `"deepseek"`, `"ollama"`.

**Returns:** `string` (confirmation message)
```pipe
ai_provider "deepseek"
ai_model "deepseek-chat"
```

### `ai_model`

**Signature:** `ai_model(name)`

**Description:** Sets the model name for the current AI provider.

**Returns:** `string` (confirmation message)
```pipe
ai_model "gpt-4o"
ai_model "claude-3-5-sonnet-20241022"
```

### `ai_host`

**Signature:** `ai_host(url)`

**Description:** Sets a custom API host URL. Useful for local proxies or self-hosted endpoints.

**Returns:** `string` (confirmation message)
```pipe
-- Ollama
ai_host "http://localhost:11434/v1"
```

### `ai_timeout`

**Signature:** `ai_timeout(seconds)`

**Description:** Sets the AI request timeout in seconds.

**Returns:** `nil`
```pipe
ai_timeout 30
```

### `ai_cache`

**Signature:** `ai_cache(option)`

**Description:** Controls AI response caching. Cached responses are stored in memory and expire after the configured TTL (default 10 minutes). The cache key is a SHA-256 hash of provider + model + system prompt + user prompt.

`option` can be:
- `true` / `"on"` — enable cache (default TTL: 10 minutes)
- `false` / `"off"` — disable cache
- A number — enable cache with custom TTL in minutes
- `"clear"` — flush all cached entries
- `"stats"` — return hit/miss counts

**Returns:** `string` (status message or stats)

```pipe
-- Enable with default 10 min TTL
ai_cache on

-- Enable with 30 minute TTL
ai_cache 30

-- Check cache statistics
ai_cache "stats"
    > print

-- Disable caching
ai_cache off

-- Clear all cached entries
ai_cache "clear"
```

**Example:**
```pipe
ai_provider "deepseek"
ai_cache 10

-- API call (miss)
ask "What is the capital of France?"
    > print

-- Instant response from cache (hit)
ask "What is the capital of France?"
    > print
```

---

## 10.17 AI — Low-level Chat (2 functions)

### `ai_chat`

**Signature:** `ai_chat(system_prompt, user_prompt)`

**Description:** Sends a chat request with system and user prompts. Returns the assistant's response.

**Returns:** `string`
```pipe
ai_chat "You are a helpful assistant" "What is Pipe?"
```

### `ai_chat_json`

**Signature:** `ai_chat_json(system_prompt, user_prompt)`

**Description:** Like `ai_chat`, but parses the response as JSON and returns the parsed value.

**Returns:** `any` (parsed JSON — map, list, number, string, boolean, or nil)
```pipe
data: ai_chat_json "Return JSON" "List 3 colors as JSON array"
-- ["red", "green", "blue"]
```

---

## 10.18 AI — Streaming (1 function)

### `ai_stream`

**Signature:** `ai_stream(system_prompt, user_prompt)`

**Description:** Sends a chat request and streams the response token-by-token to stdout in real time. Returns the full accumulated response.

**Returns:** `string`
```pipe
full: ai_stream "Explain" "How does AI work?"
-- tokens print as they arrive
```

---

## 10.19 AI — High-level Convenience (7 functions)

### `summarize`

**Signature:** `summarize(text)`

**Description:** Summarizes the given text in 2-3 sentences.

**Returns:** `string`
```pipe
summarize "Long article text here..."
```

### `translate`

**Signature:** `translate(text, target_language)`

**Description:** Translates `text` into `target_language`.

**Returns:** `string`
```pipe
translate "Hello world" "German"
```

### `classify`

**Signature:** `classify(text, categories)`

**Description:** Classifies `text` into exactly one of the given categories. `categories` can be a string (comma-separated) or a list.

**Returns:** `string` (the chosen category)
```pipe
classify "The app crashes on submit" (["bug", "feature", "question"])
```

### `extract`

**Signature:** `extract(text, schema)`

**Description:** Extracts structured data from `text` according to a schema description. Returns parsed JSON.

**Returns:** `any` (parsed JSON)
```pipe
data: extract "Alice is 30 and lives in Paris" "Extract name, age, city as JSON"
```

### `generate`

**Signature:** `generate(prompt)`

**Description:** Generates text from a single prompt (no system message).

**Returns:** `string`
```pipe
generate "Write a haiku about programming"
```

### `generate_json`

**Signature:** `generate_json(instruction, schema)`

**Description:** Generates structured JSON data matching a schema description using AI. The model is instructed to respond with valid JSON only — no markdown, no explanation.

**Returns:** `any` — parsed JSON as native Pipe types (map, list, number, string, boolean)

```pipe
users: generate_json "Create 3 fake users" "name: string, email: string, age: number"

-- Access fields
first: at users 0
(get first "name")
    > print
(get first "email")
    > print
```

**Example — generate config:**
```pipe
config: generate_json "Define a web server config" "host: string, port: number, debug: boolean"
print (get config "host")
print (get config "port")
```

### `ask`

**Signature:** `ask(question)`

**Description:** Answers a single question conversationally.

**Returns:** `string`
```pipe
ask "What is the meaning of life?"
```

---

## 10.20 AI — Parallel (3 functions)

### `ai_batch`

**Signature:** `ai_batch(system_prompt, items)`

**Description:** Processes a list of items in parallel with the same system prompt. Each item is formatted into the user message.

**Returns:** `list` of `string` (one response per item)
```pipe
items: ["Summarize: Go", "Summarize: Rust", "Summarize: Zig"]
results: ai_batch "Explain briefly" items
```

### `ai_parallel`

**Signature:** `ai_parallel(concurrency, system_prompt, items)`

**Description:** Like `ai_batch` but with explicit `concurrency` limit (max parallel requests).

**Returns:** `list` of `string`
```pipe
ai_parallel 2 "Translate to French" (["Hello", "Goodbye"])
```

### `ai_rate_limit`

**Signature:** `ai_rate_limit(calls_per_second)`

**Description:** Limits the rate of AI calls for subsequent parallel/batch operations.

**Returns:** `nil`
```pipe
-- max 5 calls per second
ai_rate_limit 5
```

---

## 10.21 AI — Tool Calling (2 functions)

### `ai_tool`

**Signature:** `ai_tool(name, description, parameters, function)`

**Description:** Registers a tool that the AI can call. `parameters` is a JSON schema map. `function` is called with the tool arguments.

**Returns:** `nil`
```pipe
ai_tool "get_weather" "Get weather for a city" { city: { type: "string" } } (fn args
        http_get "https://wttr.in/" ++ get args "city")
```

### `ai_with_tools`

**Signature:** `ai_with_tools(system_prompt, user_prompt, max_rounds?)`

**Description:** Sends a chat request with tool-calling enabled. The AI can invoke registered tools to answer the query. `max_rounds` defaults to 5.

**Returns:** `string`
```pipe
ai_with_tools "You have weather data" "What is the weather in Berlin?"
```

---

## 10.22 AI — Agents (3 functions)

### `agent`

**Signature:** `agent(name, system_prompt)`

**Description:** Creates a stateful agent with the given name and system prompt. Agents maintain their own conversation history across calls. Multiple agents can coexist independently.

**Returns:** `string` (confirmation message)

```pipe
agent "helper" "You are a helpful assistant. Keep answers short."
agent "poet" "You are a poet. Respond in rhyming couplets."
```

### `agent_ask`

**Signature:** `agent_ask(name, message)`

**Description:** Sends a message to the named agent. The agent remembers all previous messages (conversation history), allowing for multi-turn conversations with context.

**Returns:** `string` (agent response)

```pipe
agent_ask "helper" "What is the capital of France?"
    > print

-- Agent remembers context from previous message
agent_ask "helper" "What's its population?"
    > print
```

### `agent_clear`

**Signature:** `agent_clear(name)`

**Description:** Clears the conversation history of the named agent, keeping its system prompt. The agent will not remember previous messages.

**Returns:** `string` (confirmation message)

```pipe
agent_clear "helper"
```

**Example — multi-agent conversation:**
```pipe
ai_provider "deepseek"
agent "de" "Du bist ein deutscher Assistent."
agent "eng" "You are an English assistant."

agent_ask "de" "Was ist die Hauptstadt von Frankreich?"
    > print

agent_ask "eng" "What is the capital of France?"
    > print
```

---

## 10.23 AI — Embeddings (5 functions)

### `embed`

**Signature:** `embed(text)`

**Description:** Converts text into an embedding vector (list of floats).

**Returns:** `list` of `float`
```pipe
vec: embed "Pipe programming language"
```

### `embed_batch`

**Signature:** `embed_batch(items)`

**Description:** Embeds multiple texts in parallel.

**Returns:** `list` of `list` of `float`
```pipe
vecs: embed_batch (["Hello", "World", "Pipe"])
```

### `cosine_sim`

**Signature:** `cosine_sim(vec_a, vec_b)`

**Description:** Computes the cosine similarity between two embedding vectors.

**Returns:** `float`
```pipe
a: embed "cat"
b: embed "dog"
cosine_sim a b
```

### `dot_product`

**Signature:** `dot_product(vec_a, vec_b)`

**Description:** Computes the dot product of two vectors.

**Returns:** `float`
```pipe
dot_product (embed "a") (embed "b")
```

### `nearest`

**Signature:** `nearest(query_vec, doc_vectors, k)`

**Description:** Finds the `k` nearest neighbors to `query_vec` among `doc_vectors` using cosine similarity.

**Returns:** `list` of `integer` (indices of nearest neighbors)
```pipe
docs: embed_batch (["cat", "dog", "bird", "fish"])
q: embed "pet"
-- [1, 0] (dog, cat)
nearest q docs 2
```

---

## 10.24 AI — Search (1 function)

### `web_search`

**Signature:** `web_search(query)`

**Description:** Searches the web using DuckDuckGo's free Instant Answer API. No API key required. Returns a list of result maps, each with `title`, `snippet`, and `url` keys.

**Returns:** `list` of `map` (each map has keys: `title`, `snippet`, `url`)

```pipe
results: web_search "Go programming language"

first: at results 0
(get first "title")
    > print
(get first "snippet")
    > print
```

**Example — RAG with web search:**
```pipe
ai_provider "deepseek"

results: web_search "How does a quantum computer work?"

context: ""
for r in results
    context: context ++ (get r "title") ++ "\n" ++ (get r "snippet") ++ "\n---\n"

ask ("Context:\n" ++ context ++ "\nQuestion: Explain quantum computing simply")
    > print
```

---

## 10.25 Sandbox (3 functions)

### `sandbox_profile`
**Signature:** `sandbox_profile(name)`
**Description:** Selects a sandbox profile (`none`, `strict`, `noexec`, `isolated`, `networked`).
**Returns:** `string`
```pipe
sandbox_profile "strict"
```

### `set_sandbox`
**Signature:** `set_sandbox(profile)`
**Description:** Sets the active sandbox from a profile map or name.
**Returns:** `string`
```pipe
set_sandbox ({type: "strict", write: false})
```

### `with_sandbox`
**Signature:** `with_sandbox(profile, fn)`
**Description:** Runs `fn` under the given sandbox profile, then restores the previous one.
**Returns:** `any`
```pipe
with_sandbox "noexec" (fn
    print "isolated")
```

---

## 10.26 Test Assertions (6 functions)

**Note:** `test` blocks and assert builtins are available in all execution modes, but are designed for use with `pipe -test`.

### `assert`

**Signature:** `assert(condition)`

**Description:** Asserts that a value is truthy.

**Returns:** `nil` on success, `ERROR` on failure
```pipe
assert (2 + 2) == 4
assert true
```

### `assert_eq`

**Signature:** `assert_eq(expected, actual)`

**Description:** Asserts that two values are equal.

**Returns:** `nil` on success, `ERROR` on failure
```pipe
assert_eq (2 + 2) 4
assert_eq "hello" ("hel" ++ "lo")
```

### `assert_not_eq`

**Signature:** `assert_not_eq(unexpected, actual)`

**Description:** Asserts that two values are not equal.

**Returns:** `nil` on success, `ERROR` on failure
```pipe
assert_not_eq 5 6
assert_not_eq "cat" "dog"
```

### `assert_lt`

**Signature:** `assert_lt(a, b)`

**Description:** Asserts that `a < b` (numeric comparison).

**Returns:** `nil` on success, `ERROR` on failure
```pipe
assert_lt 3 5
assert_lt (-5) 0
```

### `assert_gt`

**Signature:** `assert_gt(a, b)`

**Description:** Asserts that `a > b` (numeric comparison).

**Returns:** `nil` on success, `ERROR` on failure
```pipe
assert_gt 10 3
assert_gt 0 (-5)
```

### `assert_error`

**Signature:** `assert_error(fn)`

**Description:** Asserts that calling `fn()` returns an error. Pass a zero-argument function.

**Returns:** `nil` on success, `ERROR` on failure
```pipe
failing: (fn
    read_file "nonexistent")
assert_error failing
```

### `test` — test block

**Description:** Groups assertions into a named test. Inside `pipe -test`, each test block prints `PASS name` or `FAIL name (reason)`. A failing assertion stops only its own test block.

```pipe
test "addition"
    assert_eq (2 + 2) 4
    assert_lt 3 5
```

---

## 10.27 Summary Table

### IO & System (6)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 1 | `print` | `print(value ...)` | `nil` |
| 2 | `input` | `input(prompt)` | `string` |
| 3 | `exec` | `exec(command)` | `string` |
| 4 | `env` | `env(name)` | `string` or `nil` |
| 5 | `sleep` | `sleep(ms)` | `nil` |
| 6 | `go` | `go(fn, args...)` | `nil` |

### File System (17)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 7 | `read_file` | `read_file(path)` | `string` |
| 8 | `write_file` | `write_file(path, content)` | `nil` |
| 9 | `append_file` | `append_file(path, content)` | `nil` |
| 10 | `read_lines` | `read_lines(path)` | `list` of strings |
| 11 | `file_exists` | `file_exists(path)` | `boolean` |
| 12 | `file_delete` | `file_delete(path)` | `boolean` |
| 13 | `file_move` | `file_move(src, dst)` | `nil` |
| 14 | `file_copy` | `file_copy(src, dst)` | `nil` |
| 15 | `file_size` | `file_size(path)` | `number` |
| 16 | `file_type` | `file_type(path)` | `string` or `nil` |
| 17 | `list_dir` | `list_dir(path)` | `list` of strings |
| 18 | `make_dir` | `make_dir(path)` | `nil` |
| 19 | `remove_dir` | `remove_dir(path)` | `nil` |
| 20 | `path_join` | `path_join(base, part)` | `string` |
| 21 | `path_base` | `path_base(path)` | `string` |
| 22 | `path_dir` | `path_dir(path)` | `string` |
| 23 | `path_ext` | `path_ext(path)` | `string` |

### String (6)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 24 | `upper` | `upper(str)` | `string` |
| 25 | `lower` | `lower(str)` | `string` |
| 26 | `trim` | `trim(str)` | `string` |
| 27 | `split` | `split(str, delimiter)` | `list` of strings |
| 28 | `join` | `join(list, delimiter)` | `string` |
| 29 | `contains` | `contains(haystack, needle)` | `boolean` |

### List (11)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 30 | `len` | `len(value)` | `number` |
| 31 | `push` | `push(list, value)` | The mutated `list` |
| 32 | `pop` | `pop(list)` | The removed element, or `nil` if empty |
| 33 | `at` | `at(collection, index)` | `any` or `nil` |
| 34 | `slice_list` | `slice_list(list, range)` | `list` |
| 35 | `sort` | `sort(list)` | The mutated `list` |
| 36 | `range` | `range(start, end)` | `list` of numbers |
| 37 | `map` | `map(list, fn)` | `list` |
| 38 | `filter` | `filter(list, fn)` | `list` |
| 39 | `reduce` | `reduce(list, fn, initial)` | `any` |
| 40 | `each` | `each(list, fn)` | `nil` |

### Map (4)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 41 | `get` | `get(map, key)` | `any` or `nil` |
| 42 | `set` | `set(map, key, value)` | The mutated `map` |
| 43 | `keys` | `keys(map)` | `list` of strings |
| 44 | `values` | `values(map)` | `list` |

### Math (6)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 45 | `abs` | `abs(x)` | `number` |
| 46 | `min` | `min(a, b)` | `number` |
| 47 | `max` | `max(a, b)` | `number` |
| 48 | `pow` | `pow(base, exp)` | `number` |
| 49 | `sqrt` | `sqrt(x)` | `number` |
| 50 | `round` | `round(x)` | `number` |

### Network & HTTP (5)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 51 | `http_get` | `http_get(url)` | `string` |
| 52 | `http_post` | `http_post(url, body)` | `string` |
| 53 | `http_get_json` | `http_get_json(url)` | `any` (map, list, number, string, boolean, or nil) |
| 54 | `parse_json` | `parse_json(json_string)` | `any` or `nil` on parse error |
| 55 | `to_json` | `to_json(value)` | `string` |

### TCP (6)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 56 | `tcp_listen` | `tcp_listen(address)` | `listener handle` |
| 57 | `tcp_connect` | `tcp_connect(address)` | `connection handle` |
| 58 | `tcp_accept` | `tcp_accept(listener)` | `connection handle` |
| 59 | `tcp_read` | `tcp_read(conn, max_bytes)` | `string` |
| 60 | `tcp_write` | `tcp_write(conn, data)` | `nil` |
| 61 | `tcp_close` | `tcp_close(handle)` | `nil` |

### Regex (2)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 62 | `regex_match` | `regex_match(pattern, str)` | `boolean` |
| 63 | `regex_replace` | `regex_replace(pattern, replacement, str)` | `string` |

### Date & Time (2)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 64 | `now` | `now()` | `number` |
| 65 | `format_time` | `format_time(timestamp, layout)` |  |

### Random (2)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 66 | `random` | `random()` | `number` |
| 67 | `random_range` | `random_range(min, max)` | `number` (integer) |

### Encoding (2)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 68 | `base64_encode` | `base64_encode(str)` | `string` |
| 69 | `base64_decode` | `base64_decode(str)` | `string` |

### Type Checks (6)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 70 | `type_of` | `type_of(value)` | `string` — one of `"number"`, `"string"`, `"list"`, `"map"`, `"nil"`, `"function"`, `"boolean"`, `"result"` |
| 71 | `is_num` | `is_num(value)` | `boolean` |
| 72 | `is_str` | `is_str(value)` | `boolean` |
| 73 | `is_list` | `is_list(value)` | `boolean` |
| 74 | `is_map` | `is_map(value)` | `boolean` |
| 75 | `is_nil` | `is_nil(value)` | `boolean` |

### Conversion (2)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 76 | `to_str` | `to_str(value)` | `string` |
| 77 | `to_num` | `to_num(str)` | `number` or `nil` |

### Result Type (6)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 78 | `Ok` | `Ok(value)` | `Result` (Ok variant) |
| 79 | `Err` | `Err(message)` | `Result` (Err variant) |
| 80 | `is_ok` | `is_ok(result)` | `boolean` |
| 81 | `is_err` | `is_err(result)` | `boolean` |
| 82 | `unwrap` | `unwrap(result)` | `any` |
| 83 | `unwrap_or` | `unwrap_or(result, default)` | `any` |

### AI — Configuration (5)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 84 | `ai_provider` | `ai_provider(name)` |  |
| 85 | `ai_model` | `ai_model(name)` |  |
| 86 | `ai_host` | `ai_host(url)` |  |
| 87 | `ai_timeout` | `ai_timeout(seconds)` |  |
| 88 | `ai_cache` | `ai_cache(option)` | `string` |

### AI — Low-level Chat (2)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 89 | `ai_chat` | `ai_chat(system_prompt, user_prompt)` |  |
| 90 | `ai_chat_json` | `ai_chat_json(system_prompt, user_prompt)` |  |

### AI — Streaming (1)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 91 | `ai_stream` | `ai_stream(system_prompt, user_prompt)` |  |

### AI — High-level Convenience (7)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 92 | `summarize` | `summarize(text)` |  |
| 93 | `translate` | `translate(text, target_language)` |  |
| 94 | `classify` | `classify(text, categories)` |  |
| 95 | `extract` | `extract(text, schema)` |  |
| 96 | `generate` | `generate(prompt)` |  |
| 97 | `generate_json` | `generate_json(instruction, schema)` |  |
| 98 | `ask` | `ask(question)` |  |

### AI — Parallel (3)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 99 | `ai_batch` | `ai_batch(system_prompt, items)` |  |
| 100 | `ai_parallel` | `ai_parallel(concurrency, system_prompt, items)` |  |
| 101 | `ai_rate_limit` | `ai_rate_limit(calls_per_second)` |  |

### AI — Tool Calling (2)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 102 | `ai_tool` | `ai_tool(name, description, parameters, function)` |  |
| 103 | `ai_with_tools` | `ai_with_tools(system_prompt, user_prompt, max_rounds?)` |  |

### AI — Agents (3)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 104 | `agent` | `agent(name, system_prompt)` | `string` |
| 105 | `agent_ask` | `agent_ask(name, message)` | `string` |
| 106 | `agent_clear` | `agent_clear(name)` | `string` |

### AI — Embeddings (5)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 107 | `embed` | `embed(text)` |  |
| 108 | `embed_batch` | `embed_batch(items)` |  |
| 109 | `cosine_sim` | `cosine_sim(vec_a, vec_b)` |  |
| 110 | `dot_product` | `dot_product(vec_a, vec_b)` |  |
| 111 | `nearest` | `nearest(query_vec, doc_vectors, k)` |  |

### AI — Search (1)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 112 | `web_search` | `web_search(query)` | `list` |

### Sandbox (3)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 113 | `sandbox_profile` | `sandbox_profile(name)` | `string` |
| 114 | `set_sandbox` | `set_sandbox(profile)` | `string` |
| 115 | `with_sandbox` | `with_sandbox(profile, fn)` | `any` |

### Test Assertions (6)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 116 | `assert` | `assert(condition)` |  |
| 117 | `assert_eq` | `assert_eq(expected, actual)` |  |
| 118 | `assert_not_eq` | `assert_not_eq(unexpected, actual)` |  |
| 119 | `assert_lt` | `assert_lt(a, b)` |  |
| 120 | `assert_gt` | `assert_gt(a, b)` |  |
| 121 | `assert_error` | `assert_error(fn)` |  |

---

**Total: 121 built-in functions**
