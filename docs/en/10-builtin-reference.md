# 10. Built-in Function Reference

Pipe includes 81 built-in functions organized by category. This chapter documents each function with its signature, description, return type, and usage example.

---

## 10.1 IO & System (5 functions)

### `print`
**Signature:** `print(value ...)`
**Description:** Prints one or more values to stdout, separated by spaces and followed by a newline.
**Returns:** `nil`
```pipe
print "Hello"                          # Hello
print "The answer is" 42               # The answer is 42
print "x:" x "y:" y                    # x: 10 y: 20
```

### `input`
**Signature:** `input(prompt)`
**Description:** Displays `prompt` (optional), reads a line from stdin, and returns it as a string.
**Returns:** `string`
```pipe
let name = input "Enter your name: "
print "Hello, " + name
```

### `exec`
**Signature:** `exec(command)`
**Description:** Executes a system command via the shell and returns the combined stdout/stderr output.
**Returns:** `string`
```pipe
let files = exec "ls -la"
print files

let version = exec "git --version"
print version
```

### `env`
**Signature:** `env(name)`
**Description:** Returns the value of the environment variable `name`, or `nil` if not set.
**Returns:** `string` or `nil`
```pipe
let home = env "HOME"
print "Home directory: " + home

let path = env "PATH"
print path
```

### `sleep`
**Signature:** `sleep(ms)`
**Description:** Pauses execution for `ms` milliseconds.
**Returns:** `nil`
```pipe
sleep 1000      # wait 1 second
print "Done waiting"

sleep 500       # wait 0.5 seconds
```

---

## 10.2 File System (16 functions)

### `read_file`
**Signature:** `read_file(path)`
**Description:** Reads the entire contents of a file and returns it as a string. Errors if the file does not exist.
**Returns:** `string`
```pipe
let content = read_file "config.json"
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
let lines = read_lines "data.csv"
each lines fn(line) {
    print "Line: " + line
}
```

### `file_exists`
**Signature:** `file_exists(path)`
**Description:** Returns `true` if the file or directory at `path` exists, `false` otherwise.
**Returns:** `boolean`
```pipe
if file_exists "config.json" {
    print "Config found"
} {
    print "Config missing"
}
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
let size = file_size "data.bin"
print "File size: " + size + " bytes"
```

### `file_type`
**Signature:** `file_type(path)`
**Description:** Returns `"file"`, `"dir"`, or `nil` if the path does not exist.
**Returns:** `string` or `nil`
```pipe
let t = file_type "config.json"
if t == "dir" {
    print "It's a directory"
}
```

### `list_dir`
**Signature:** `list_dir(path)`
**Description:** Returns a list of filenames in the directory at `path`.
**Returns:** `list` of strings
```pipe
let files = list_dir "."
each files fn(f) { print f }
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
let full = path_join "/home/user" "docs"
print full      # "/home/user/docs"
```

### `path_base`
**Signature:** `path_base(path)`
**Description:** Returns the last component of a path (the filename or directory name).
**Returns:** `string`
```pipe
path_base "/home/user/file.txt"     # "file.txt"
path_base "/home/user/"             # "user"
```

### `path_dir`
**Signature:** `path_dir(path)`
**Description:** Returns the directory portion of a path, without the final component.
**Returns:** `string`
```pipe
path_dir "/home/user/file.txt"      # "/home/user"
path_dir "file.txt"                 # "."
```

### `path_ext`
**Signature:** `path_ext(path)`
**Description:** Returns the file extension including the dot, or empty string if none.
**Returns:** `string`
```pipe
path_ext "file.txt"     # ".txt"
path_ext "archive.tar.gz"  # ".gz"
path_ext "Makefile"     # ""
```

---

## 10.3 String (6 functions)

### `upper`
**Signature:** `upper(str)`
**Description:** Returns a copy of `str` with all characters converted to uppercase.
**Returns:** `string`
```pipe
upper "hello"       # "HELLO"
upper "Hello World" # "HELLO WORLD"
```

### `lower`
**Signature:** `lower(str)`
**Description:** Returns a copy of `str` with all characters converted to lowercase.
**Returns:** `string`
```pipe
lower "HELLO"       # "hello"
lower "Hello World" # "hello world"
```

### `trim`
**Signature:** `trim(str)`
**Description:** Returns a copy of `str` with leading and trailing whitespace removed.
**Returns:** `string`
```pipe
trim "  hello  "    # "hello"
trim "\t indented\n"  # "indented"
```

### `split`
**Signature:** `split(str, delimiter)`
**Description:** Splits `str` into a list of substrings separated by `delimiter`.
**Returns:** `list` of strings
```pipe
split "a,b,c" ","       # ["a", "b", "c"]
split "one two three" " "  # ["one", "two", "three"]
split "hello" ""        # ["h", "e", "l", "l", "o"]
```

### `join`
**Signature:** `join(list, delimiter)`
**Description:** Joins list elements into a string with `delimiter` between each element.
**Returns:** `string`
```pipe
join ["a", "b", "c"] ","    # "a,b,c"
join ["one", "two"] " - "   # "one - two"
join [1, 2, 3] ""           # "123"
```

### `contains`
**Signature:** `contains(haystack, needle)`
**Description:** For strings: checks if `needle` is a substring. For lists: checks if `needle` is an element.
**Returns:** `boolean`
```pipe
contains "hello world" "lo"     # true
contains "hello world" "xyz"    # false
contains [1, 2, 3] 2            # true
contains ["a", "b"] "c"         # false
```

---

## 10.4 List (11 functions)

### `len`
**Signature:** `len(value)`
**Description:** Returns the length of a string (characters), list (elements), or map (keys).
**Returns:** `number`
```pipe
len "hello"         # 5
len [1, 2, 3]       # 3
len { "a" = 1 }     # 1
```

### `push`
**Signature:** `push(list, value)`
**Description:** Appends `value` to the end of `list`. **Mutates the list in place.**
**Returns:** The mutated `list`
```pipe
let xs = [1, 2]
push xs 3           # xs is now [1, 2, 3]
push xs "hello"     # xs is now [1, 2, 3, "hello"]
```

### `pop`
**Signature:** `pop(list)`
**Description:** Removes and returns the last element of `list`. **Mutates the list in place.**
**Returns:** The removed element, or `nil` if empty
```pipe
let xs = [1, 2, 3]
let last = pop xs   # last = 3, xs is now [1, 2]
pop []              # nil
```

### `at`
**Signature:** `at(collection, index)`
**Description:** Returns the element at 0-based `index` in a list or string. Returns `nil` if out of bounds.
**Returns:** `any` or `nil`
```pipe
at [10, 20, 30] 1   # 20
at "hello" 0         # "h"
at [1, 2] 99         # nil
```

### `slice_list`
**Signature:** `slice_list(list, range)`
**Description:** Returns a sublist from `start` to `end` (exclusive). Uses `start..end` syntax.
**Returns:** `list`
```pipe
slice_list [10, 20, 30, 40, 50] 0..3    # [10, 20, 30]
slice_list [10, 20, 30, 40, 50] 2..5    # [30, 40, 50]
slice_list [10, 20, 30] 1..1             # []
```

### `sort`
**Signature:** `sort(list)`
**Description:** Sorts `list` in ascending order **in place**. For strings, sorts lexicographically. For numbers, sorts numerically.
**Returns:** The mutated `list`
```pipe
let xs = [3, 1, 4, 1, 5]
sort xs             # xs is now [1, 1, 3, 4, 5]

let ys = ["b", "a", "c"]
sort ys             # ys is now ["a", "b", "c"]
```

### `range`
**Signature:** `range(start, end)`
**Description:** Creates a list of numbers from `start` (inclusive) to `end` (exclusive).
**Returns:** `list` of numbers
```pipe
range 0 5           # [0, 1, 2, 3, 4]
range 5 10          # [5, 6, 7, 8, 9]
range 0 0           # []
```

### `map`
**Signature:** `map(list, fn)`
**Description:** Applies `fn` to each element and returns a new list of results. In VM mode, only built-in functions are accepted.
**Returns:** `list`
```pipe
map [1, 2, 3] fn(x) { x * 2 }       # [2, 4, 6]
map ["a", "b"] fn(s) { upper s }    # ["A", "B"]
```

### `filter`
**Signature:** `filter(list, fn)`
**Description:** Returns a new list containing only elements where `fn(element)` returns truthy. In VM mode, only built-in functions are accepted.
**Returns:** `list`
```pipe
filter [1, 2, 3, 4] fn(x) { x > 2 }        # [3, 4]
filter [0, 1, 0, 3] fn(x) { x }             # [1, 3] (truthy elements)
```

### `reduce`
**Signature:** `reduce(list, fn, initial)`
**Description:** Accumulates a value by calling `fn(accumulator, element)` for each element. `initial` is the starting accumulator. In VM mode, only built-in functions are accepted.
**Returns:** `any`
```pipe
reduce [1, 2, 3] fn(acc, x) { acc + x } 0      # 6
reduce [2, 3, 4] fn(acc, x) { acc * x } 1      # 24
```

### `each`
**Signature:** `each(list, fn)`
**Description:** Calls `fn(element)` for each element in `list`. Used for side effects. Works with both built-in and user functions in all modes.
**Returns:** `nil`
```pipe
each [1, 2, 3] fn(x) { print x }
# 1
# 2
# 3
```

---

## 10.5 Map (4 functions)

### `get`
**Signature:** `get(map, key)`
**Description:** Returns the value associated with `key` in `map`, or `nil` if key not found.
**Returns:** `any` or `nil`
```pipe
let m = { "name" = "Alice", "age" = 30 }
get m "name"        # "Alice"
get m "country"     # nil
```

### `set`
**Signature:** `set(map, key, value)`
**Description:** Sets `key` to `value` in `map`. Creates new key or updates existing. **Mutates the map in place.**
**Returns:** The mutated `map`
```pipe
let m = { "name" = "Alice" }
set m "age" 30      # adds key
set m "name" "Bob"  # updates key
# m is now { "name" = "Bob", "age" = 30 }
```

### `keys`
**Signature:** `keys(map)`
**Description:** Returns a list of all keys in `map`. Order is not guaranteed.
**Returns:** `list` of strings
```pipe
let m = { "a" = 1, "b" = 2, "c" = 3 }
keys m              # ["a", "b", "c"] (order may vary)
```

### `values`
**Signature:** `values(map)`
**Description:** Returns a list of all values in `map`. Order corresponds to `keys` order.
**Returns:** `list`
```pipe
let m = { "a" = 1, "b" = 2, "c" = 3 }
values m            # [1, 2, 3] (order corresponds to keys)
```

---

## 10.6 Math (6 functions)

### `abs`
**Signature:** `abs(x)`
**Description:** Returns the absolute value of `x`.
**Returns:** `number`
```pipe
abs -5          # 5
abs 42          # 42
abs 0           # 0
```

### `min`
**Signature:** `min(a, b)`
**Description:** Returns the smaller of `a` and `b`.
**Returns:** `number`
```pipe
min 10 20       # 10
min -5 0        # -5
min 3.14 3.0    # 3.0
```

### `max`
**Signature:** `max(a, b)`
**Description:** Returns the larger of `a` and `b`.
**Returns:** `number`
```pipe
max 10 20       # 20
max -5 0        # 0
max 3.14 3.0    # 3.14
```

### `pow`
**Signature:** `pow(base, exp)`
**Description:** Returns `base` raised to the power of `exp`.
**Returns:** `number`
```pipe
pow 2 10        # 1024
pow 2 0.5       # 1.414... (square root)
pow 10 3        # 1000
```

### `sqrt`
**Signature:** `sqrt(x)`
**Description:** Returns the square root of `x`.
**Returns:** `number`
```pipe
sqrt 100        # 10
sqrt 2          # 1.414...
sqrt 0          # 0
```

### `round`
**Signature:** `round(x)`
**Description:** Rounds `x` to the nearest integer. Half values round to the nearest even integer (banker's rounding).
**Returns:** `number`
```pipe
round 3.14      # 3
round 3.5       # 4
round 2.5       # 2 (banker's rounding)
round 4.7       # 5
```

---

## 10.7 Network & HTTP (5 functions)

### `http_get`
**Signature:** `http_get(url)`
**Description:** Performs an HTTP GET request to `url` and returns the response body as a string.
**Returns:** `string`
```pipe
let body = http_get "https://api.example.com/data"
print body
```

### `http_post`
**Signature:** `http_post(url, body)`
**Description:** Performs an HTTP POST request to `url` with `body` as the request payload. Returns the response body.
**Returns:** `string`
```pipe
let resp = http_post "https://api.example.com/submit" "{\"key\": \"value\"}"
print resp
```

### `http_get_json`
**Signature:** `http_get_json(url)`
**Description:** Performs an HTTP GET request to `url` and parses the response as JSON.
**Returns:** `any` (map, list, number, string, boolean, or nil)
```pipe
let data = http_get_json "https://api.example.com/users"
print data.count
each data.users fn(u) { print u.name }
```

### `parse_json`
**Signature:** `parse_json(json_string)`
**Description:** Parses a JSON string into Pipe data structures (maps, lists, numbers, strings, booleans, nil).
**Returns:** `any` or `nil` on parse error
```pipe
let obj = parse_json `{"name": "Alice", "scores": [95, 87, 92]}`
print obj.name          # "Alice"
print obj.scores.1      # 87
```

### `to_json`
**Signature:** `to_json(value)`
**Description:** Serializes a Pipe value into a JSON string.
**Returns:** `string`
```pipe
let data = { "name" = "Alice", "age" = 30 }
let json_str = to_json data
print json_str          # {"name":"Alice","age":30}
```

---

## 10.8 TCP (6 functions)

### `tcp_listen`
**Signature:** `tcp_listen(address)`
**Description:** Starts a TCP server listening on `address` (e.g., `":8080"` or `"localhost:3000"`). Returns a listener handle.
**Returns:** `listener handle`
```pipe
let listener = tcp_listen ":8080"
print "Server listening on port 8080"
```

### `tcp_connect`
**Signature:** `tcp_connect(address)`
**Description:** Connects to a TCP server at `address` and returns a connection handle.
**Returns:** `connection handle`
```pipe
let conn = tcp_connect "localhost:8080"
tcp_write conn "Hello, server!"
```

### `tcp_accept`
**Signature:** `tcp_accept(listener)`
**Description:** Accepts an incoming connection on a listener. Blocks until a client connects. Returns a connection handle.
**Returns:** `connection handle`
```pipe
let conn = tcp_accept listener
let msg = tcp_read conn 1024
print "Received: " + msg
```

### `tcp_read`
**Signature:** `tcp_read(conn, max_bytes)`
**Description:** Reads up to `max_bytes` from a TCP connection and returns the data as a string.
**Returns:** `string`
```pipe
let data = tcp_read conn 4096
print data
```

### `tcp_write`
**Signature:** `tcp_write(conn, data)`
**Description:** Writes `data` to a TCP connection.
**Returns:** `nil`
```pipe
tcp_write conn "HTTP/1.1 200 OK\r\n\r\nHello"
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
let listener = tcp_listen ":9999"
print "Echo server on :9999"

each range 0 5 fn(i) {
    let conn = tcp_accept listener
    let msg = tcp_read conn 1024
    print "Got: " + msg
    tcp_write conn "Echo: " + msg
    tcp_close conn
}

tcp_close listener
```

---

## 10.9 Regex (2 functions)

### `regex_match`
**Signature:** `regex_match(pattern, str)`
**Description:** Returns `true` if `str` matches the regex `pattern`, `false` otherwise.
**Returns:** `boolean`
```pipe
regex_match "^[a-z]+$" "hello"      # true
regex_match "^[a-z]+$" "hello123"   # false
regex_match "\\d{3}-\\d{4}" "555-1234"  # true
```

### `regex_replace`
**Signature:** `regex_replace(pattern, replacement, str)`
**Description:** Replaces all occurrences of `pattern` in `str` with `replacement`.
**Returns:** `string`
```pipe
regex_replace "\\s+" "-" "hello world"         # "hello-world"
regex_replace "[aeiou]" "*" "hello"            # "h*ll*"
regex_replace "\\d" "#" "abc123xyz"            # "abc###xyz"
```

---

## 10.10 Date & Time (2 functions)

### `now`
**Signature:** `now()`
**Description:** Returns the current time as a Unix timestamp in seconds (floating point, includes fractional seconds).
**Returns:** `number`
```pipe
let t = now()
print "Current timestamp: " + t
# e.g. 1700000000.123456
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
let t = now()

format_time t "2006-01-02"              # "2024-07-15"
format_time t "2006-01-02 15:04:05"     # "2024-07-15 14:30:00"
format_time t "Mon Jan 2 15:04:05 2006" # "Mon Jul 15 14:30:00 2024"
format_time t "03:04 PM"                # "02:30 PM"
format_time t "Monday, January 2, 2006" # "Monday, July 15, 2024"
```

---

## 10.11 Random (2 functions)

### `random`
**Signature:** `random()`
**Description:** Returns a random floating-point number in the range `[0.0, 1.0)`.
**Returns:** `number`
```pipe
let r = random()        # e.g. 0.734291
print r
```

### `random_range`
**Signature:** `random_range(min, max)`
**Description:** Returns a random integer in the range `[min, max]` inclusive.
**Returns:** `number` (integer)
```pipe
let dice = random_range 1 6        # 1, 2, 3, 4, 5, or 6
let coin = random_range 0 1        # 0 or 1
```

---

## 10.12 Encoding (2 functions)

### `base64_encode`
**Signature:** `base64_encode(str)`
**Description:** Encodes a string to Base64.
**Returns:** `string`
```pipe
base64_encode "hello"       # "aGVsbG8="
base64_encode "Pipe"        # "UGlwZQ=="
```

### `base64_decode`
**Signature:** `base64_decode(str)`
**Description:** Decodes a Base64-encoded string.
**Returns:** `string`
```pipe
base64_decode "aGVsbG8="   # "hello"
base64_decode "UGlwZQ=="   # "Pipe"
```

---

## 10.13 Type Checks (6 functions)

### `type_of`
**Signature:** `type_of(value)`
**Description:** Returns a string indicating the type of `value`.
**Returns:** `string` — one of `"number"`, `"string"`, `"list"`, `"map"`, `"nil"`, `"function"`, `"boolean"`, `"result"`
```pipe
type_of 42              # "number"
type_of "hello"         # "string"
type_of [1, 2]          # "list"
type_of { "a" = 1 }     # "map"
type_of nil             # "nil"
type_of fn(x) { x }     # "function"
type_of true            # "boolean"
type_of (Ok 1)          # "result"
```

### `is_num`
**Signature:** `is_num(value)`
**Description:** Returns `true` if `value` is a number.
**Returns:** `boolean`
```pipe
is_num 42       # true
is_num 3.14     # true
is_num "42"     # false
is_num nil      # false
```

### `is_str`
**Signature:** `is_str(value)`
**Description:** Returns `true` if `value` is a string.
**Returns:** `boolean`
```pipe
is_str "hello"  # true
is_str 42       # false
is_str ""       # true
```

### `is_list`
**Signature:** `is_list(value)`
**Description:** Returns `true` if `value` is a list.
**Returns:** `boolean`
```pipe
is_list [1, 2]  # true
is_list []      # true
is_list "ab"    # false
```

### `is_map`
**Signature:** `is_map(value)`
**Description:** Returns `true` if `value` is a map.
**Returns:** `boolean`
```pipe
is_map { "a" = 1 }  # true
is_map {}           # true
is_map [1, 2]       # false
```

### `is_nil`
**Signature:** `is_nil(value)`
**Description:** Returns `true` if `value` is `nil`.
**Returns:** `boolean`
```pipe
is_nil nil      # true
is_nil 0        # false
is_nil ""       # false
is_nil false    # false
```

---

## 10.14 Conversion (2 functions)

### `to_str`
**Signature:** `to_str(value)`
**Description:** Converts `value` to its string representation. Numbers, booleans, and nil all convert to strings.
**Returns:** `string`
```pipe
to_str 42           # "42"
to_str 3.14         # "3.14"
to_str true         # "true"
to_str nil          # "nil"
to_str [1, 2, 3]   # "[1, 2, 3]"
```

### `to_num`
**Signature:** `to_num(str)`
**Description:** Parses a string into a number. Returns `nil` if parsing fails.
**Returns:** `number` or `nil`
```pipe
to_num "42"         # 42
to_num "3.14"       # 3.14
to_num "hello"      # nil
to_num "0xFF"       # nil (only decimal)
```

---

## 10.15 Result Type (6 functions)

### `Ok`
**Signature:** `Ok(value)`
**Description:** Creates a successful `Result` containing `value`.
**Returns:** `Result` (Ok variant)
```pipe
let r = Ok 42
is_ok r     # true
unwrap r    # 42
```

### `Err`
**Signature:** `Err(message)`
**Description:** Creates a failed `Result` containing the error `message` string.
**Returns:** `Result` (Err variant)
```pipe
let r = Err "something went wrong"
is_err r    # true
unwrap r    # ERROR: called unwrap on an Err value
```

### `is_ok`
**Signature:** `is_ok(result)`
**Description:** Returns `true` if `result` is an `Ok` variant.
**Returns:** `boolean`
```pipe
is_ok (Ok 42)       # true
is_ok (Err "oops")  # false
```

### `is_err`
**Signature:** `is_err(result)`
**Description:** Returns `true` if `result` is an `Err` variant.
**Returns:** `boolean`
```pipe
is_err (Ok 42)      # false
is_err (Err "oops") # true
```

### `unwrap`
**Signature:** `unwrap(result)`
**Description:** Returns the value inside `Ok`, or raises an error if called on `Err`.
**Returns:** `any`
```pipe
unwrap (Ok 42)          # 42
unwrap (Err "fail")     # ERROR: called unwrap on an Err value
```

### `unwrap_or`
**Signature:** `unwrap_or(result, default)`
**Description:** Returns the value inside `Ok`, or `default` if `result` is `Err`.
**Returns:** `any`
```pipe
unwrap_or (Ok 42) 0      # 42
unwrap_or (Err "x") 0    # 0
unwrap_or (Ok "hi") ""   # "hi"
```

---

## 10.16 Concurrency (1 function)

### `go`
**Signature:** `go(fn)`
**Description:** Launches `fn` as a concurrent goroutine. The function runs in parallel with the main program. **Tree-Walker mode only.** Not available in Bytecode VM.

**Returns:** `nil`
```pipe
go fn() {
    sleep 1000
    print "Hello from goroutine!"
}

print "Main continues immediately..."
sleep 2000
print "Main done"

# Output (order may vary):
# Main continues immediately...
# Hello from goroutine!
# Main done
```

**Important limitations:**
- Goroutines share the same address space; there is no built-in synchronization mechanism.
- Use with caution when accessing shared mutable state.
- Only works in Tree-Walker interpreter mode.

---

## 10.17 Summary Table

### IO & System (5)
| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 1 | `print` | `print(values...)` | `nil` |
| 2 | `input` | `input(prompt)` | `string` |
| 3 | `exec` | `exec(command)` | `string` |
| 4 | `env` | `env(name)` | `string` or `nil` |
| 5 | `sleep` | `sleep(ms)` | `nil` |

### File System (16)
| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 6 | `read_file` | `read_file(path)` | `string` |
| 7 | `write_file` | `write_file(path, content)` | `nil` |
| 8 | `append_file` | `append_file(path, content)` | `nil` |
| 9 | `read_lines` | `read_lines(path)` | `list` |
| 10 | `file_exists` | `file_exists(path)` | `boolean` |
| 11 | `file_delete` | `file_delete(path)` | `boolean` |
| 12 | `file_move` | `file_move(src, dst)` | `nil` |
| 13 | `file_copy` | `file_copy(src, dst)` | `nil` |
| 14 | `file_size` | `file_size(path)` | `number` |
| 15 | `file_type` | `file_type(path)` | `string` or `nil` |
| 16 | `list_dir` | `list_dir(path)` | `list` |
| 17 | `make_dir` | `make_dir(path)` | `nil` |
| 18 | `remove_dir` | `remove_dir(path)` | `nil` |
| 19 | `path_join` | `path_join(base, part)` | `string` |
| 20 | `path_base` | `path_base(path)` | `string` |
| 21 | `path_dir` | `path_dir(path)` | `string` |
| 22 | `path_ext` | `path_ext(path)` | `string` |

### String (6)
| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 23 | `upper` | `upper(str)` | `string` |
| 24 | `lower` | `lower(str)` | `string` |
| 25 | `trim` | `trim(str)` | `string` |
| 26 | `split` | `split(str, delim)` | `list` |
| 27 | `join` | `join(list, delim)` | `string` |
| 28 | `contains` | `contains(haystack, needle)` | `boolean` |

### List (11)
| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 29 | `len` | `len(value)` | `number` |
| 30 | `push` | `push(list, value)` | `list` |
| 31 | `pop` | `pop(list)` | `any` or `nil` |
| 32 | `at` | `at(collection, index)` | `any` or `nil` |
| 33 | `slice_list` | `slice_list(list, range)` | `list` |
| 34 | `sort` | `sort(list)` | `list` |
| 35 | `range` | `range(start, end)` | `list` |
| 36 | `map` | `map(list, fn)` | `list` |
| 37 | `filter` | `filter(list, fn)` | `list` |
| 38 | `reduce` | `reduce(list, fn, initial)` | `any` |
| 39 | `each` | `each(list, fn)` | `nil` |

### Map (4)
| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 40 | `get` | `get(map, key)` | `any` or `nil` |
| 41 | `set` | `set(map, key, value)` | `map` |
| 42 | `keys` | `keys(map)` | `list` |
| 43 | `values` | `values(map)` | `list` |

### Math (6)
| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 44 | `abs` | `abs(x)` | `number` |
| 45 | `min` | `min(a, b)` | `number` |
| 46 | `max` | `max(a, b)` | `number` |
| 47 | `pow` | `pow(base, exp)` | `number` |
| 48 | `sqrt` | `sqrt(x)` | `number` |
| 49 | `round` | `round(x)` | `number` |

### Network & HTTP (5)
| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 50 | `http_get` | `http_get(url)` | `string` |
| 51 | `http_post` | `http_post(url, body)` | `string` |
| 52 | `http_get_json` | `http_get_json(url)` | `any` |
| 53 | `parse_json` | `parse_json(json_str)` | `any` or `nil` |
| 54 | `to_json` | `to_json(value)` | `string` |

### TCP (6)
| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 55 | `tcp_listen` | `tcp_listen(address)` | handle |
| 56 | `tcp_connect` | `tcp_connect(address)` | handle |
| 57 | `tcp_accept` | `tcp_accept(listener)` | handle |
| 58 | `tcp_read` | `tcp_read(conn, max)` | `string` |
| 59 | `tcp_write` | `tcp_write(conn, data)` | `nil` |
| 60 | `tcp_close` | `tcp_close(handle)` | `nil` |

### Regex (2)
| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 61 | `regex_match` | `regex_match(pattern, str)` | `boolean` |
| 62 | `regex_replace` | `regex_replace(pattern, repl, str)` | `string` |

### Date & Time (2)
| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 63 | `now` | `now()` | `number` |
| 64 | `format_time` | `format_time(ts, layout)` | `string` |

### Random (2)
| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 65 | `random` | `random()` | `number` |
| 66 | `random_range` | `random_range(min, max)` | `number` |

### Encoding (2)
| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 67 | `base64_encode` | `base64_encode(str)` | `string` |
| 68 | `base64_decode` | `base64_decode(str)` | `string` |

### Type Checks (6)
| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 69 | `type_of` | `type_of(value)` | `string` |
| 70 | `is_num` | `is_num(value)` | `boolean` |
| 71 | `is_str` | `is_str(value)` | `boolean` |
| 72 | `is_list` | `is_list(value)` | `boolean` |
| 73 | `is_map` | `is_map(value)` | `boolean` |
| 74 | `is_nil` | `is_nil(value)` | `boolean` |

### Conversion (2)
| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 75 | `to_str` | `to_str(value)` | `string` |
| 76 | `to_num` | `to_num(str)` | `number` or `nil` |

### Result Type (6)
| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 77 | `Ok` | `Ok(value)` | `Result` |
| 78 | `Err` | `Err(message)` | `Result` |
| 79 | `is_ok` | `is_ok(result)` | `boolean` |
| 80 | `is_err` | `is_err(result)` | `boolean` |
| 81 | `unwrap` | `unwrap(result)` | `any` |
| 82 | `unwrap_or` | `unwrap_or(result, default)` | `any` |

### Concurrency (1)
| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 83 | `go` | `go(fn)` | `nil` |

---

**Total: 83 built-in functions** (note: `contains` and `len` serve double roles for strings and lists, counted once each)
