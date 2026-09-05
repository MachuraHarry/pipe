# 10. Built-in Function Reference

Pipe includes 232 built-in functions organized by category. This chapter documents each function with its signature, description, return type, and usage example.

---

## 10.1 IO & System (9 functions)

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

### `print_raw`
**Signature:** `print_raw(value ...)`
**Description:** Prints the string representation of one or more values to stdout with no separator, no trailing space and no added newline. Useful when the exact output bytes matter (e.g. emitting JSON or machine-readable data).
**Returns:** `nil`
```pipe
out: "{\"status\":\"ok\"}\n"
print_raw out
-- identical bytes to what was placed in out
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
**Description:** Executes a system command via the shell and returns a map with the combined stdout/stderr output, error message (empty on success) and exit status.
**Returns:** `map` with keys `output`, `error`, `status`
```pipe
result: exec "ls -la"
print (get result "output")
print (get result "status")

result: exec "git --version"
print (get result "output")
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


### `args`

**Signature:** `args()`

**Description:** Returns the CLI arguments passed to the script as a list of strings.

**Returns:** `list` of `string`
```pipe
-- pipe script.pipe hello world
args
-- ["hello", "world"]
```

### `read_stdin`

**Signature:** `read_stdin()`

**Description:** Reads the entire standard input and returns it as a string.

**Returns:** `string`
```pipe
-- echo "hello" | pipe script.pipe
data: read_stdin
print data   -- "hello\n"
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

### `spawn`
**Signature:** `spawn(fn, args...)`
**Description:** Launches `fn` in the background and returns a **Future** — a placeholder that resolves to the function's result. Use `await` to block until it completes.
**Returns:** `future`
```pipe
task: spawn long_computation 42
-- do other work...
result: await task
print result
```

### `await`
**Signature:** `await(future, timeout_ms?)`
**Description:** Blocks until a Future resolves and returns its value. Pass a timeout (in milliseconds) as the second argument to give up with an error instead of blocking forever. On a non-Future value it is a no-op.
**Returns:** the resolved value (or an error on timeout)
```pipe
task: spawn slow_op
result: await task        -- wait indefinitely
-- or with a timeout:
result: await task 5000   -- error after 5 s
```

### `chan`
**Signature:** `chan(capacity?)`
**Description:** Creates a channel for communicating between concurrent tasks. Unbuffered if no capacity is given; buffered with that many slots otherwise.
**Returns:** `channel`
```pipe
c: chan 3        -- buffered
u: chan          -- unbuffered
```

### `send`
**Signature:** `send(channel, value)`
**Description:** Sends a value on the channel, blocking until a receiver takes it. Errors if the channel is closed.
**Returns:** `nil`
```pipe
send c 42
```

### `recv`
**Signature:** `recv(channel)`
**Description:** Receives a value from the channel, blocking until one is available. Returns `nil` once the channel is closed and drained.
**Returns:** the received value (or `nil` on close)
```pipe
v: recv c
```

### `try_recv`
**Signature:** `try_recv(channel)`
**Description:** Receives a value without blocking. Returns `nil` if no value is available (or the channel is closed).
**Returns:** the received value or `nil`

### `try_send`
**Signature:** `try_send(channel, value)`
**Description:** Sends a value without blocking. Returns `false` if the channel is full or closed.
**Returns:** `bool`

### `close`
**Signature:** `close(channel)`
**Description:** Closes the channel. Sends afterwards error; receivers drain any buffered values and then return `nil`.
**Returns:** `nil`

### `chan_len`
**Signature:** `chan_len(channel)`
**Description:** Returns the number of values currently buffered in the channel.
**Returns:** `number`

### `chan_cap`
**Signature:** `chan_cap(channel)`
**Description:** Returns the capacity of the channel (0 for unbuffered).
**Returns:** `number`

### `mutex`
**Signature:** `mutex()`
**Description:** Creates a mutual exclusion lock for synchronizing concurrent tasks.
**Returns:** `mutex`

### `lock`
**Signature:** `lock(mutex)`
**Description:** Acquires the mutex, blocking until it is available.
**Returns:** `nil`

### `unlock`
**Signature:** `unlock(mutex)`
**Description:** Releases the mutex.
**Returns:** `nil`

### `try_lock`
**Signature:** `try_lock(mutex)`
**Description:** Acquires the mutex without blocking. Returns `false` if it is already held.
**Returns:** `bool`

### `semaphore`
**Signature:** `semaphore(count)`
**Description:** Creates a counting semaphore limiting concurrent access to a resource (e.g. a worker pool or a rate limit).
**Returns:** `semaphore`

### `acquire`
**Signature:** `acquire(semaphore)`
**Description:** Acquires the semaphore, blocking until a permit is available.
**Returns:** `nil`

### `release`
**Signature:** `release(semaphore)`
**Description:** Releases a permit back to the semaphore.
**Returns:** `nil`

### `try_acquire`
**Signature:** `try_acquire(semaphore)`
**Description:** Acquires the semaphore without blocking. Returns `false` if no permit is available.
**Returns:** `bool`

---

## 10.2 File System (23 functions)

### `read_file`
**Signature:** `read_file(path, mode?)`
**Description:** Reads the entire contents of a file. Default mode returns a string. Pass `"bytes"` as second argument to get raw bytes for binary files.
**Returns:** `string` or `bytes`
```pipe
content: read_file "config.json"
print content

data: read_file "image.png" "bytes"
print len data
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

### `file_open`
**Signature:** `file_open(path, mode)`
**Description:** Opens a file in random-access mode and returns a numeric handle. Modes: `"r"` (read), `"w"` (write, truncate), `"a"` (append), `"rw"` (read/write, keep), `"rw+"` (read/write, truncate). Respects the active sandbox profile.
**Returns:** `number` (handle)
```pipe
h: file_open "/tmp/data.bin" "rw"
```

### `file_close`
**Signature:** `file_close(handle)`
**Description:** Closes a file previously opened with `file_open`, releasing its handle.
**Returns:** `nil`
```pipe
file_close h
```

### `file_read`
**Signature:** `file_read(handle, offset, n)`
**Description:** Reads `n` bytes from `handle` starting at `offset` and returns them as `bytes`. Returns fewer bytes when past the end of the file.
**Returns:** `bytes`
```pipe
-- first 8 bytes
file_read h 0 8
```

### `file_write`
**Signature:** `file_write(handle, offset, data)`
**Description:** Writes `data` (bytes or string) to `handle` at `offset`, overwriting existing bytes. Returns the number of bytes written.
**Returns:** `number`
```pipe
file_write h 0 (to_bytes "0123456789")
```

### `file_truncate`
**Signature:** `file_truncate(handle, size)`
**Description:** Truncates the file to exactly `size` bytes.
**Returns:** `nil`
```pipe
file_truncate h 8
```

### `file_sync`
**Signature:** `file_sync(handle)`
**Description:** Flushes file data and metadata to disk (fsync).
**Returns:** `nil`
```pipe
file_sync h
```

---

## 10.3 String (9 functions)

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


### `repeat`

**Signature:** `repeat(str, count)`

**Description:** Repeats `str` `count` times. Much faster than while-loop string concatenation for large counts (O(n) vs O(n²)).

**Returns:** `string`
```pipe
repeat "abc" 3
-- "abcabcabc"

repeat "-" 10
-- "----------"
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

### `substring`
**Signature:** `substring(str, start, end)`
**Description:** Returns the substring of `str` from `start` (inclusive) to `end` (exclusive). Indexes are clamped to the string bounds.
**Returns:** `string`
```pipe
-- "el"
substring "hello" 1 3
-- "hello"
substring "hello" -5 99
```

### `index_of`
**Signature:** `index_of(str, needle)`
**Description:** Returns the 0-based index of the first occurrence of `needle` in `str`, or `-1` if not found.
**Returns:** `number`
```pipe
-- 6
index_of "hello world" "world"
-- -1
index_of "hello" "xyz"
```

---

## 10.4 List (13 functions)

### `len`
**Signature:** `len(value)`
**Description:** Returns the length of a string (characters), list (elements), map (keys), or bytes (bytes).
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

### `slice`
**Signature:** `slice(value, start, end)`
**Description:** Returns a slice of `value` from `start` (inclusive) to `end` (exclusive). Works on lists, strings, and bytes. Indexes are clamped.
**Returns:** `list`, `string`, or `bytes`
```pipe
-- [10, 20, 30]
slice ([10, 20, 30, 40, 50]) 0 3
-- "el"
slice "hello" 1 3
-- 0x656c
slice (to_bytes "hello") 1 3
```

### `sort`
**Signature:** `sort(list, comparator?)`
**Description:** Returns a new list sorted from `list`. Without a comparator: strings lexicographically, numbers numerically. With a comparator function `comparator(a, b)` returning truthy when `a` sorts before `b`, uses it for ordering.
**Returns:** A new `list`
```pipe
xs: [3, 1, 4, 1, 5]
-- [1, 1, 3, 4, 5] (xs unchanged)
sort xs

ys: ["b", "a", "c"]
-- ["a", "b", "c"]
sort ys

-- descending
sort [1, 2, 3] (fn a b: b < a)
```

### `sorted_by`
**Signature:** `sorted_by(list, keyFn)`
**Description:** Returns a new list sorted by the key that `keyFn(element)` returns for each element.
**Returns:** `list`
```pipe
-- ["a", "bb", "ccc"]
sorted_by ["ccc", "a", "bb"] (fn s: len s)
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

## 10.5 CSV (6 functions)

### `replace`
**Signature:** `replace(str, old, new)`
**Description:** Replaces the first occurrence of `old` with `new` in `str`.
**Returns:** `string`
```pipe
replace "hello world" "world" "pipe"
-- "hello pipe"
```

### `replace_all`
**Signature:** `replace_all(str, old, new)`
**Description:** Replaces all occurrences of `old` with `new` in `str`.
**Returns:** `string`
```pipe
replace_all "a,b,c" "," " | "
-- "a | b | c"
```

### `index_of`
**Signature:** `index_of(haystack, needle)`
**Description:** Returns the 0-based index of the first occurrence of `needle` in `haystack`. Works on strings and lists. Returns `-1` if not found.
**Returns:** `number`
```pipe
index_of "hello world" "world"   -- 6
index_of ["a", "b", "c"] "c"     -- 2
```

### `contains`
**Signature:** `contains(container, item)`
**Description:** Checks if `container` contains `item`. Works on strings (substring match), lists (element match via ValuesEqual), and maps (key existence).
**Returns:** `boolean`
```pipe
contains "hello world" "world"   -- true
contains ["a","b","c"] "d"       -- false
contains {a: 1, b: 2} "a"       -- true (map key check)
```

### `csv_parse`

**Signature:** `csv_parse(text)`

**Description:** Parses CSV-formatted text into a list of maps. The first row is treated as column headers. Handles multi-line CSV strings.

**Returns:** `list` of `map`
```pipe
data: csv_parse "name,age
Alice,30
Bob,25"

print (len data)                -- 2
print (get (at data 0) "name")  -- "Alice"
```

### `csv_format`

**Signature:** `csv_format(data, headers?)`

**Description:** Formats a list of maps into a CSV string. Optional second argument specifies column order (list of strings).

**Returns:** `string`
```pipe
rows: [{name: "Alice", age: 30}, {name: "Bob", age: 25}]
csv: csv_format rows
print csv
-- name,age
-- Alice,30
-- Bob,25
```


## 10.6 Map (6 functions)

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

### `map_delete`
**Signature:** `map_delete(map, key)`
**Description:** Removes `key` (and its value) from `map`. Does nothing if the key does not exist.
**Returns:** `nil`
```pipe
m: {a: 1, b: 2}
map_delete m "a"
-- m is now {b: 2}
```

### `unique`
**Signature:** `unique(list)`
**Description:** Returns a new list with duplicate elements removed. Preserves order of first occurrence. Uses type-aware deduplication (Integer and Float are distinct).
**Returns:** `list`
```pipe
unique [1, 2, 2, 3, 1]
-- [1, 2, 3]
```

---

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

## 10.7 Math (8 functions)

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

### `ceil`
**Signature:** `ceil(x)`
**Description:** Rounds `x` up to the nearest integer.
**Returns:** `number`
```pipe
ceil 3.2    -- 4
ceil 4.8    -- 5
ceil (-1.5) -- -1
```

### `floor`
**Signature:** `floor(x)`
**Description:** Rounds `x` down to the nearest integer.
**Returns:** `number`
```pipe
floor 3.2    -- 3
floor 4.8    -- 4
floor (-1.5) -- -2
```

---

## 10.8 Network & HTTP (8 functions)

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

### `http_request`
**Signature:** `http_request(method, url, headers?, body?)`
**Description:** Performs an HTTP request with custom method, URL, optional headers map and body. Returns a map with `status`, `headers`, and `body` keys.
**Returns:** `map` — `{status: number, headers: map, body: string}`
```pipe
h: {}
set h "Content-Type" "application/json"
r: http_request "POST" "https://api.example.com/data" h "{\"key\":\"value\"}"
print (get r "status")
print (get r "body")
```

### `http_server`
**Signature:** `http_server(addr, handler)`
**Description:** Starts an HTTP server on `addr` (e.g. `"0.0.0.0:8080"`). `handler` is a function `fn(req)` that receives a request map `{method, path, query, headers, body}` and must return a response map `{status, headers, body}`. The server runs in the background. Returns a server handle.
**Returns:** `server`
```pipe
fn handle req
    name: get req "path"
    {status: 200, headers: {}, body: "Hello " ++ name}

server: http_server "0.0.0.0:8080" handle
print "Server running on :8080"
sleep 60000
http_close server
```

### `http_close`
**Signature:** `http_close(server)`
**Description:** Shuts down an HTTP server and releases the port.
**Returns:** `nil`
```pipe
http_close server
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

## 10.9 TCP (9 functions)

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

### `tcp_connect_tls`
**Signature:** `tcp_connect_tls(host, port, servername?, insecure?)`
**Description:** Connects to a TLS-secured TCP server. `servername` defaults to `host` for SNI and certificate verification. `insecure` (bool) skips certificate verification (for self-signed certificates). Returns a connection handle.
**Returns:** `connection handle`
```pipe
-- MQTT broker with TLS on port 8883
conn: tcp_connect_tls "mqtt.example.com" 8883

-- With explicit servername for SNI
conn: tcp_connect_tls "192.168.1.100" 8883 "mqtt.example.com"

-- Skip certificate verification (self-signed certificates)
conn: tcp_connect_tls "192.168.1.100" 8883 "mqtt.example.com" true
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

### `tcp_read_bytes`
**Signature:** `tcp_read_bytes(conn, n)`
**Description:** Reads exactly `n` bytes from a TCP connection and returns them as bytes. Blocks until all bytes are available.
**Returns:** `bytes`
```pipe
-- Read exactly 1024 bytes (useful for MQTT binary payloads)
raw: tcp_read_bytes conn 1024
-- Access individual bytes: at raw 0 returns INTEGER (0-255)
first_byte: at raw 0
```

### `tcp_set_read_timeout`
**Signature:** `tcp_set_read_timeout(conn, ms)`
**Description:** Sets a read deadline of `ms` milliseconds on a TCP connection. Subsequent reads error if no data arrives within the deadline. Pass `0` to clear the timeout.
**Returns:** `nil`
```pipe
tcp_set_read_timeout conn 5000
tcp_set_read_timeout conn 0
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

## 10.10 Regex (2 functions)

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

## 10.11 Date & Time (3 functions)

### `now`
**Signature:** `now()`
**Description:** Returns the current time as a Unix timestamp in seconds (floating point, includes fractional seconds).
**Returns:** `number`
```pipe
t: now
print "Current timestamp: " ++ t
-- e.g. 1700000000.123456
```

### `time_ms`
**Signature:** `time_ms()`
**Description:** Returns the current time as a Unix timestamp in milliseconds (integer). Use for fine-grained timing and benchmarks.
**Returns:** `number`
```pipe
start: time_ms
sleep 10
elapsed: time_ms - start
print "Elapsed: " ++ elapsed ++ " ms"
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

## 10.12 Random (2 functions)

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

## 10.13 Cryptography (8 functions)

> **Note:** Unlike `random` and `random_range` (which use pseudo-random `math/rand`), the `secure_*` functions use `crypto/rand` and are suitable for cryptographic purposes like key generation, tokens, and nonces.

### `secure_random`
**Signature:** `secure_random(byte_count)`
**Description:** Generates `byte_count` cryptographically secure random bytes, returned as a hex-encoded string. Maximum 1024 bytes per call.
**Returns:** `string`
```pipe
-- 32-character hex string (16 bytes)
key: secure_random 16
-- "a1b2c3d4e5f6789012345678abcdef01"
```

### `secure_random_int`
**Signature:** `secure_random_int()`
**Description:** Returns a cryptographically secure random 64-bit signed integer.
**Returns:** `number`
```pipe
token: secure_random_int
-- -8746123456789012345
```

### `secure_random_range`
**Signature:** `secure_random_range(min, max)`
**Description:** Returns a cryptographically secure random integer in the range `[min, max)`.
**Returns:** `number`
```pipe
-- random int between 100000 and 999999
code: secure_random_range 100000 1000000
```

### `secure_random_bytes`
**Signature:** `secure_random_bytes(byte_count)`
**Description:** Generates `byte_count` cryptographically secure random bytes. Returns raw bytes (not hex). For key material, IVs, and nonces. Max 1024 bytes.
**Returns:** `bytes`
```pipe
-- 32 raw bytes for AES-256 key
key: secure_random_bytes 32
```

### `encrypt`
**Signature:** `encrypt(key, plaintext[, associated_data])`
**Description:** Encrypts `plaintext` using AES-GCM with the given `key`. The key can be a raw string (16/24/32 bytes), a hex-encoded key from `secure_random`, or raw bytes from `secure_random_bytes`. A random nonce is generated and prepended to the ciphertext. Optional `associated_data` is authenticated but not encrypted (AEAD).
**Returns:** `string` (base64-encoded ciphertext)
```pipe
-- AES-256 encryption with hex key
key: secure_random 32
enc: encrypt key "Hello World"
-- "g+F+k0q...base64..."

-- With associated data (authenticated but not encrypted)
enc: encrypt key "Hello" "meta-info"
```

### `decrypt`
**Signature:** `decrypt(key, ciphertext[, associated_data])`
**Description:** Decrypts AES-GCM ciphertext. The key must match the one used for encryption. The nonce is extracted from the ciphertext. If the key is wrong or the data is tampered, returns an error.
**Returns:** `string`
```pipe
decrypt key enc           -- "Hello World"
decrypt wrong_key enc     -- ERROR: authentication failed
```

### `hmac_sha256`
**Signature:** `hmac_sha256(key, message)`
**Description:** Computes the HMAC-SHA256 of `message` using `key`. Returns a hex-encoded 32-byte MAC. Used for message authentication, API signing, JWT tokens.
**Returns:** `string` (64 hex chars)
```pipe
-- Generate MAC
sig: hmac_sha256 "secret-key" "Transfer 100 EUR"
-- "8f14e45f...64 hex chars..."

-- Verify (recompute and compare)
sig == (hmac_sha256 "secret-key" "Transfer 100 EUR")   -- true
```

### `hmac_sha512`
**Signature:** `hmac_sha512(key, message)`
**Description:** Computes the HMAC-SHA512 of `message` using `key`. Returns a hex-encoded 64-byte MAC. Higher security than HMAC-SHA256.
**Returns:** `string` (128 hex chars)
```pipe
sig: hmac_sha512 "key" "data"
```

---

## 10.14 Encoding (2 functions)

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

## 10.15 Hashing (4 functions)

### `sha256`

**Signature:** `sha256(text)`

**Description:** Computes the SHA-256 hash of `text` and returns it as a 64-character hexadecimal string.

**Returns:** `string`
```pipe
sha256 "hello"
-- 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
```

### `md5`

**Signature:** `md5(text)`

**Description:** Computes the MD5 hash of `text` and returns it as a 32-character hex string. Useful for fast non-cryptographic checksums.

**Returns:** `string`
```pipe
md5 "hello"
-- 5d41402abc4b2a76b9719d911017c592
```

### `sha1`

**Signature:** `sha1(text)`

**Description:** Computes the SHA-1 hash of `text` and returns it as a 40-character hex string. Used by git.

**Returns:** `string`
```pipe
sha1 "hello"
-- aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d
```

### `sha512`

**Signature:** `sha512(text)`

**Description:** Computes the SHA-512 hash of `text` and returns it as a 128-character hex string.

**Returns:** `string`
```pipe
sha512 "hello"
-- 9b71d224bd62f3785d96d46ad3ea3d73319bfbc2890caadae2dff72519673ca7...
```

---

## 10.16 Database — SQLite (module)

The `db_open`, `db_close`, `db_exec`, and `db_query` builtins have been replaced by a native Pipe module available in the [`pipe-modules`](https://github.com/MachuraHarry/pipe-modules) registry. The external `modernc.org/sqlite` dependency has been removed. The binary is now ~8 MB, dependency-free.

Install with `pipe -get sqlite`, then import via `import "sqlite"`:

```pipe
-- Classic API
import "sqlite"
h: db_open ":memory:"
db_exec h "CREATE TABLE tasks (id INTEGER PRIMARY KEY, title TEXT, priority TEXT)"
db_exec h "INSERT INTO tasks VALUES (1, 'Fix bug', 'high'), (2, 'Update docs', 'medium')"
rows: db_query h "SELECT * FROM tasks WHERE priority = 'high'"
for row in rows
    print (row.title)
db_close h

-- Pipeline API (composable via > operator)
fn is_high row
    (row.priority) == "high"
q h "SELECT * FROM tasks" > filter is_high > each print
```

**Supported SQL:** CREATE TABLE, INSERT (single + multi-value), UPDATE, DELETE, SELECT with WHERE, GROUP BY + aggregates (COUNT, SUM, AVG, MIN, MAX), ORDER BY, LIMIT/OFFSET, JOIN (INNER, LEFT, RIGHT), DISTINCT, BEGIN/COMMIT/ROLLBACK. SQL is case-insensitive.

**Pipeline helpers:** `q` / `exec` (short aliases), `row_get` / `row_eq` / `row_ne` (filter predicates). Demo: `examples/sqlite_pipeline.pipe`.

Persistence uses a paged binary format with CRC32 checksums; `":memory:"` for in-memory databases.

See the [SQLite Module](26-sqlite-module.md) chapter for architecture details and benchmarks (Pipe vs Python vs Lua).

> **Note:** TV mode is fully functional — all operations work (CREATE, INSERT, SELECT, WHERE, GROUP BY, UPDATE, DELETE). VM mode has a known compiler bug with large module imports — individual operations work but complex dispatch (db_exec → exec_insert) fails. See the [SQLite Module](26-sqlite-module.md) chapter.

| Function | Signature | Description |
|----------|-----------|-------------|
| `db_open` | `db_open(path)` | Opens a database file or `":memory:"`, returns handle |
| `db_close` | `db_close(handle)` | Closes database and persists changes to disk |
| `db_exec` | `db_exec(handle, sql)` | Executes DDL/DML, returns affected row count |
| `db_query` | `db_query(handle, sql)` | Executes SELECT, returns list of row maps |
| `q` | `q(handle, sql)` | Shorthand for `db_query` (pipeline-friendly) |
| `exec` | `exec(handle, sql)` | Shorthand for `db_exec` |
| `row_get` | `row_get(row, key)` | Nil-safe field access from a row map |
| `row_eq` | `row_eq(row, key, val)` | Predicate: row[key] == val (for `filter`) |
| `row_ne` | `row_ne(row, key, val)` | Predicate: row[key] != val (for `filter`) |


## 10.17 Type Checks (6 functions)

### `type_of`
**Signature:** `type_of(value)`
**Description:** Returns a string indicating the type of `value`.
**Returns:** `string` — one of `"number"`, `"string"`, `"list"`, `"map"`, `"nil"`, `"function"`, `"boolean"`, `"result"`, `"STRUCT"`
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
-- "STRUCT"
struct P: x, y
type_of (P)
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

## 10.18 Conversion (2 functions)

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

## 10.19 Result Type (6 functions)

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

## 10.20 AI — Configuration (6 functions)

### `ai_provider`

**Signature:** `ai_provider(name, {model, host, timeout, thinking, effort}?)`

**Description:** Sets the AI provider to `name`. Supported: `"openai"`, `"anthropic"`, `"deepseek"`, `"ollama"`. An optional config block overrides the provider's cheap-and-fast default model, the API host, and/or the request timeout in one call. For DeepSeek only, `thinking` (bool) toggles V4 thinking mode and `effort` (`"low"`, `"medium"`, `"high"`, `"xhigh"`, `"max"`, `"none"`) sets the reasoning effort; using them with another provider returns an error.

**Returns:** `string` (confirmation message)
```pipe
ai_provider "deepseek"
ai_provider "deepseek" {model: "deepseek-v4-flash", timeout: 120}
ai_provider "deepseek" {model: "deepseek-v4-pro", thinking: true, effort: "max"}
ai_model "deepseek-v4-flash"
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

### `ai_set_key`

**Signature:** `ai_set_key(provider, key)`

**Description:** Sets the API key for the given provider in memory. Use when environment variables are not available (browser, CI/CD). Supported providers: `"openai"`, `"deepseek"`, `"anthropic"`.

**Returns:** `string` (confirmation message)
```pipe
ai_set_key "deepseek" "sk-your-key-here"
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

## 10.21 AI — Low-level Chat (2 functions)

### `ai_chat`

**Signature:** `ai_chat(system_prompt, user_prompt[, max_tokens])`

**Description:** Sends a chat request with system and user prompts. Returns the assistant's response. The optional third argument limits the response to `max_tokens` tokens.

**Returns:** `string`
```pipe
ai_chat "You are a helpful assistant" "What is Pipe?"
```

### `ai_chat_json`

**Signature:** `ai_chat_json(system_prompt, user_prompt[, max_tokens])`

**Description:** Like `ai_chat`, but parses the response as JSON and returns the parsed value. The optional third argument limits the response to `max_tokens` tokens.

**Returns:** `any` (parsed JSON — map, list, number, string, boolean, or nil)
```pipe
data: ai_chat_json "Return JSON" "List 3 colors as JSON array"
-- ["red", "green", "blue"]
```

---

## 10.22 AI — Streaming (1 function)

### `ai_stream`

**Signature:** `ai_stream(system_prompt, user_prompt)`

**Description:** Sends a chat request and streams the response token-by-token to stdout in real time. Returns the full accumulated response.

**Returns:** `string`
```pipe
full: ai_stream "Explain" "How does AI work?"
-- tokens print as they arrive
```

---

## 10.23 AI — High-level Convenience (8 functions)

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

### `generate_json`

**Signature:** `generate_json(instruction, schema)`

**Description:** Generates structured JSON data matching a schema description using AI. Responds with valid JSON only — no markdown, no explanation. Returns parsed JSON as native Pipe types.

**Returns:** `any` — parsed JSON as map, list, number, string, or boolean
```pipe
users: generate_json "Create 2 fake users" "name: string, age: number"
for user in users
    print (get user "name")
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

## 10.24 AI — Parallel (3 functions)

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

## 10.25 AI — Tool Calling (2 functions)

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

## 10.26 AI — Agents (4 functions)

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

### `try_ai_log`

**Signature:** `try_ai_log()`

**Description:** Returns the log of all `try_ai` fix attempts. Each entry is a map with keys: `time` (Unix timestamp), `code` (error code), `original` (expression), `fixed` (fix expression), `attempt` (1-3), `success` (boolean).

**Returns:** `list` of `map`
```pipe
log: try_ai_log
for entry in log
    print (get entry "code") ++ ": " ++ (get entry "original") ++ " → " ++ (get entry "fixed")
```


## 10.27 AI — Embeddings (5 functions)

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

## 10.28 AI — Search (2 functions)

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


### `wiki_search`

**Signature:** `wiki_search(query)`

**Description:** Searches Wikipedia (supports CORS for browser/WASM). Free, no API key required. Returns list of maps with `title`, `snippet`, and `url`.

**Returns:** `list` of `map`
```pipe
results: wiki_search "quantum computing"
for r in results
    print (get r "title")
```

---

## 10.29 Sandbox (5 functions)

### `sandbox_profile`
**Signature:** `sandbox_profile(name)`
**Description:** Selects a sandbox profile (`none`, `strict`, `noexec`, `isolated`, `networked`). Defines a named profile with `sandbox_profile "name" {fs: ..., network: ..., exec: ..., ai: ...}`. While a restricted profile is active, only profiles that are no more permissive (a subset) can be registered.
**Returns:** `string`
```pipe
sandbox_profile "strict"
```

### `set_sandbox`
**Signature:** `set_sandbox(profile)`
**Description:** Sets the active sandbox from a profile map or name. From a non-`none` profile only equal-or-more-restrictive profiles are reachable (the sandbox ratchets down; `none` is unreachable once any other profile is active).
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

### `audit_log`
**Signature:** `audit_log()`
**Description:** Returns the audit trail of the active profile as a list of maps. Only populated when the profile was defined with `audit_log: true`.
**Returns:** `list` of `{time, event, detail, profile}`
```pipe
sandbox_profile "audited" {fs: "full", network: true, exec: false, ai: true, audit_log: true}
set_sandbox "audited"

http_get "https://example.com"
for entry in audit_log
    print entry.event ++ " -> " ++ entry.detail
-- -> http_get -> https://example.com
```

### `budget_spent`
**Signature:** `budget_spent()`
**Description:** Returns the total AI cost (in USD) recorded for the active profile. Used together with the `budget` key to monitor or enforce spending limits.
**Returns:** `number`
```pipe
sandbox_profile "budgeted" {fs: "full", network: false, exec: false, ai: true, budget: 0.01}
set_sandbox "budgeted"

ask "Hello"
print (budget_spent)
-- -> 0.000079 (approx.)
```

---

## 10.30 AI — Cost Tracking (4 functions)

### `ai_cost`
**Signature:** `ai_cost()`
**Description:** Returns the cumulative cost metrics of the current run as a map. `cache_hits`/`cache_misses` are cumulative prompt-token counts reported by the *provider itself* (e.g. DeepSeek's automatic server-side prompt-prefix caching, via `usage.prompt_cache_hit_tokens`/`prompt_cache_miss_tokens`) — not pipe's own local response cache (see [`ai_cache`](#ai_cache) below, a separate mechanism with its own `"stats"` counters). Providers that don't report prompt-cache usage always show 0/0. Pass the string `"reset"` to zero out all metrics.
**Returns:** `map` of `{cost_usd, calls, cache_hits, cache_misses}`
```pipe
print (ai_cost)
-- -> {cache_hits: 6400, calls: 2, cost_usd: 0.00012, cache_misses: 120}

ai_cost "reset"   -- reset all metrics
```

### `ai_tokens`
**Signature:** `ai_tokens()`
**Description:** Returns the total number of tokens consumed by all AI calls in the current run.
**Returns:** `number`
```pipe
print (ai_tokens)
```

### `ai_cache_hits`
**Signature:** `ai_cache_hits()`
**Description:** Returns the cumulative number of prompt tokens the provider served from its own server-side cache (same number as `ai_cost`'s `cache_hits`). Zero for providers that don't report this.
**Returns:** `number`

### `ai_cache_misses`
**Signature:** `ai_cache_misses()`
**Description:** Returns the cumulative number of prompt tokens the provider did *not* serve from its own server-side cache (same number as `ai_cost`'s `cache_misses`).
**Returns:** `number`
```pipe
print (ai_cache_hits)   -- provider prompt-cache hit tokens so far
print (ai_cache_misses) -- provider prompt-cache miss tokens so far
```

---

## 10.31 Test Assertions (9 functions)

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

### `assert_near`

**Signature:** `assert_near(expected, actual[, epsilon])`

**Description:** Asserts that `expected` and `actual` differ by at most `epsilon` (default `1e-6`). Use for floating-point comparisons.

**Returns:** `nil` on success, `ERROR` on failure
```pipe
assert_near 3.1415926 3.14159 0.00001
```

### `assert_contains`

**Signature:** `assert_contains(container, item)`

**Description:** Asserts that a string contains the substring, a list contains the element, or a map contains the string key.

**Returns:** `nil` on success, `ERROR` on failure
```pipe
assert_contains "hello world" "world"
assert_contains ([1, 2, 3]) 2
m: {name: "pipe"}
assert_contains m "name"
```

### `test` — test block

**Description:** Groups assertions into a named test. Inside `pipe -test`, each test block prints `PASS name` or `FAIL name (reason)`. A failing assertion stops only its own test block.

```pipe
test "addition"
    assert_eq (2 + 2) 4
    assert_lt 3 5
```

`test setup` and `test teardown` define file-level hooks: the setup block runs before all tests, the teardown block after all tests (even when a test fails). Hooks are silent and share the environment with the tests.

---

## 10.32 Bytes & Binary (15 functions)

### `to_bytes`
**Signature:** `to_bytes(value)`
**Description:** Converts a string to its UTF-8 bytes, or a list of integers (0-255) to bytes. Returns bytes unchanged.
**Returns:** `bytes`
```pipe
-- 0x4869
to_bytes "Hi"
-- 0x01ff
to_bytes [1, 255]
```

### `from_bytes`
**Signature:** `from_bytes(value)`
**Description:** Converts bytes to a string, decoding them as UTF-8. Returns strings unchanged.
**Returns:** `string`
```pipe
-- "Hi"
from_bytes (to_bytes "Hi")
```

### `bytes_append`
**Signature:** `bytes_append(bytes, chunk, ...)`
**Description:** Appends one or more chunks (bytes or strings) to `bytes`.
**Returns:** `bytes`
```pipe
-- 0x0102
bytes_append (to_bytes [1]) (to_bytes [2])
```

### `bytes_to_int`
**Signature:** `bytes_to_int(bytes, offset?, n?)`
**Description:** Interprets `n` big-endian bytes (max 8) starting at `offset` as an unsigned integer. Defaults to the whole `bytes`.
**Returns:** `number`
```pipe
-- 0x0102 = 258
bytes_to_int (to_bytes [1, 2]) 0 2
```

### `int_to_bytes`
**Signature:** `int_to_bytes(value, n?)`
**Description:** Encodes a non-negative integer as big-endian bytes (minimal length, or exactly `n` bytes if given).
**Returns:** `bytes`
```pipe
-- 0x0102
int_to_bytes 258 2
```

### `bytes_compare`
**Signature:** `bytes_compare(a, b)`
**Description:** Lexicographically compares two byte sequences. Negative if `a < b`, 0 if equal, positive if `a > b`.
**Returns:** `number`
```pipe
-- -1
bytes_compare (to_bytes [1]) (to_bytes [2])
```

### `hex_encode`
**Signature:** `hex_encode(bytes)`
**Description:** Encodes bytes as a lowercase hexadecimal string.
**Returns:** `string`
```pipe
-- "4869"
hex_encode (to_bytes "Hi")
```

### `hex_decode`
**Signature:** `hex_decode(str)`
**Description:** Decodes a hexadecimal string into bytes.
**Returns:** `bytes`
```pipe
-- 0x4869
hex_decode "4869"
```

### `bit_and`
**Signature:** `bit_and(a, b)`
**Description:** Bitwise AND of two integers.
**Returns:** `number`
```pipe
-- 2
bit_and 6 3
```

### `bit_or`
**Signature:** `bit_or(a, b)`
**Description:** Bitwise OR of two integers.
**Returns:** `number`
```pipe
-- 7
bit_or 6 3
```

### `bit_xor`
**Signature:** `bit_xor(a, b)`
**Description:** Bitwise XOR of two integers.
**Returns:** `number`
```pipe
-- 5
bit_xor 6 3
```

### `bit_not`
**Signature:** `bit_not(a)`
**Description:** Bitwise NOT of an integer (two's complement).
**Returns:** `number`
```pipe
-- -6
bit_not 5
```

### `bit_lshift`
**Signature:** `bit_lshift(a, n)`
**Description:** Bitwise left-shift of `a` by `n` positions. `n` must be 0-63.
**Returns:** `number`
```pipe
-- 1024
bit_lshift 1 10
```

### `bit_rshift`
**Signature:** `bit_rshift(a, n)`
**Description:** Bitwise right-shift of `a` by `n` positions. `n` must be 0-63.
**Returns:** `number`
```pipe
-- 16
bit_rshift 256 4
```

### `crc32`
**Signature:** `crc32(value)`
**Description:** Computes the IEEE CRC-32 checksum of a string or bytes.
**Returns:** `number`
```pipe
-- 907060870
crc32 "hello"
```

---

## 10.33 MCP — Model Context Protocol (13 functions)

Pipe implements the Model Context Protocol both as a **Server** (exposing tools, resources and prompts to external clients such as Claude Desktop) and as a **Client** (consuming external MCP servers in `ai_with_tools`). Transports: stdio and Streamable HTTP. See [Chapter 25](25-mcp.md) for the full guide.

### `mcp_server`

**Signature:** `mcp_server(name, version)`

**Description:** Creates an MCP server. Automatically bridges all `ai_tool`-registered functions as MCP tools and all registered resources/prompts.

**Returns:** `nil`
```pipe
mcp_server "Pipe Agent" "1.0.0"
```

### `mcp_serve_stdio`

**Signature:** `mcp_serve_stdio`

**Description:** Starts the server on stdin/stdout (blocking). For Claude Desktop, Cursor, etc.

**Returns:** `nil`
```pipe
mcp_server "Pipe Agent" "1.0.0"
mcp_serve_stdio
```

### `mcp_serve_sse`

**Signature:** `mcp_serve_sse(addr)`

**Description:** Starts a Streamable HTTP server on `addr` (e.g. `:9090`, blocking). Clients connect via `POST` + SSE; sessions are managed via `Mcp-Session-Id`.

**Returns:** `nil`
```pipe
mcp_server "Pipe Agent" "1.0.0"
mcp_serve_sse ":9090"
```

### `mcp_tools`

**Signature:** `mcp_tools`

**Description:** Lists all registered tools (local + remote) as a list of Maps with `name`, `description` and `source`.

**Returns:** `list`
```pipe
print "All tools: " ++ (to_str (len (mcp_tools)))
```

### `mcp_resource`

**Signature:** `mcp_resource(uri, name, mime, read_fn)`

**Description:** Registers a static resource. `read_fn(uri)` is called with the requested URI and returns the resource text.

**Returns:** `nil`
```pipe
fn read_docs uri
    "Documentation for " ++ uri

mcp_resource "docs://pipe" "Pipe Docs" "text/markdown" read_docs
```

### `mcp_resource_template`

**Signature:** `mcp_resource_template(template, name, mime, read_fn)`

**Description:** Registers a URI-template resource, e.g. `file:///{path}`. `read_fn(uri)` is called with the concrete URI of any matching request.

**Returns:** `nil`
```pipe
fn read_file uri
    content: read_file (replace uri "file:///" "")
    content

mcp_resource_template "file:///{path}" "File" "text/plain" read_file
```

### `mcp_prompt`

**Signature:** `mcp_prompt(name, description, args_map, build_fn)`

**Description:** Registers a prompt template. `args_map` maps argument names to a description (string) or to a Map with `description` and optional `required` (default `true`). `build_fn(args)` returns the rendered prompt text.

**Returns:** `nil`
```pipe
fn build_summary args
    "Please summarize: " ++ (get args "text")

mcp_prompt "summarize" "Summarize text" {text: "The text"} build_summary
```

### `mcp_resources`

**Signature:** `mcp_resources`

**Description:** Lists all resources (local registrations + remote clients) as a list of Maps with `uri`, `name`, `mimeType`, `description` and `source`.

**Returns:** `list`
```pipe
print (mcp_resources)
```

### `mcp_read_resource`

**Signature:** `mcp_read_resource(uri)`

**Description:** Reads a resource (static or template match) from the local registries, the local server, or a connected MCP client. Works standalone without a running server.

**Returns:** `string`
```pipe
print (mcp_read_resource "docs://pipe")
```

### `mcp_prompts`

**Signature:** `mcp_prompts`

**Description:** Lists all prompts (local registrations + remote clients) as a list of Maps with `name`, `description` and `source`.

**Returns:** `list`
```pipe
print (mcp_prompts)
```

### `mcp_prompt_get`

**Signature:** `mcp_prompt_get(name, args?)`

**Description:** Renders a prompt from the local registries, the local server, or a connected MCP client. Missing required arguments are rejected with an error.

**Returns:** `string`
```pipe
print (mcp_prompt_get "summarize" {text: "A long article ..."})
```

### `mcp_use_stdio`

**Signature:** `mcp_use_stdio(command, args..., env?, alias?)`

**Description:** Spawns a subprocess and connects to it as an MCP server via stdio. Discovers its tools and registers them in the tool registry with a `mcp0_`, `mcp1_`, ... prefix. Also discovers resources and prompts if advertised. Gated by the active sandbox profile's `exec` policy (blocked unless `exec: true`). The optional `alias` names the tool prefix (e.g. `alias "github"` produces `github_list_issues`); it follows the env map, so pass `{}` when no env vars are needed. An alias must be a valid identifier and must not collide with an existing client.

**Returns:** `string` (confirmation message)
```pipe
mcp_use_stdio "npx" "-y" "@modelcontextprotocol/server-everything"

mcp_use_stdio "npx" "-y" "@modelcontextprotocol/server-github" {GITHUB_TOKEN: (env "GITHUB_TOKEN")} "github"
```

### `mcp_use_sse`

**Signature:** `mcp_use_sse(url, alias?)`

**Description:** Connects to a Streamable HTTP MCP server via POST + SSE (session-managed). Registers its tools with a `mcp2_`, `mcp3_`, ... prefix, or a custom prefix when the optional `alias` is given (e.g. `alias "github"` produces `github_list_issues`). Gated by the active sandbox profile's `network` policy: the URL and every subsequent request (including redirects) are checked against the profile's `network_whitelist`.

**Returns:** `string` (confirmation message)
```pipe
mcp_use_sse "http://localhost:9090/"

mcp_use_sse "http://localhost:9090/" "github"
```

---

## 10.34 Summary Table

### IO & System (9)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 1 | `print` | `print(value ...)` | `nil` |
| 1a | `print_raw` | `print_raw(value ...)` | `nil` |
| 2 | `input` | `input(prompt)` | `string` |
| 3 | `exec` | `exec(command)` | `string` |
| 4 | `env` | `env(name)` | `string` or `nil` |
| 5 | `sleep` | `sleep(ms)` | `nil` |
| 6 | `args` | `args()` | `list` |
| 7 | `read_stdin` | `read_stdin()` | `string` |
| 8 | `go` | `go(fn, args...)` | `nil` |
| 8a | `spawn` | `spawn(fn, args...)` | `future` |
| 8b | `await` | `await(future, timeout_ms?)` | `any` |

### Concurrency (18)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| c1 | `chan` | `chan(capacity?)` | `channel` |
| c2 | `send` | `send(channel, value)` | `nil` |
| c3 | `recv` | `recv(channel)` | `any` or `nil` |
| c4 | `try_recv` | `try_recv(channel)` | `any` or `nil` |
| c5 | `try_send` | `try_send(channel, value)` | `bool` |
| c6 | `close` | `close(channel)` | `nil` |
| c7 | `chan_len` | `chan_len(channel)` | `number` |
| c8 | `chan_cap` | `chan_cap(channel)` | `number` |
| c9 | `mutex` | `mutex()` | `mutex` |
| c10 | `lock` | `lock(mutex)` | `nil` |
| c11 | `unlock` | `unlock(mutex)` | `nil` |
| c12 | `try_lock` | `try_lock(mutex)` | `bool` |
| c13 | `semaphore` | `semaphore(count)` | `semaphore` |
| c14 | `acquire` | `acquire(semaphore)` | `nil` |
| c15 | `release` | `release(semaphore)` | `nil` |
| c16 | `try_acquire` | `try_acquire(semaphore)` | `bool` |

### File System (23)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 9 | `read_file` | `read_file(path, mode?)` | `string` or `bytes` |
| 10 | `write_file` | `write_file(path, content)` | `nil` |
| 11 | `append_file` | `append_file(path, content)` | `nil` |
| 12 | `read_lines` | `read_lines(path)` | `list` of strings |
| 13 | `file_exists` | `file_exists(path)` | `boolean` |
| 14 | `file_delete` | `file_delete(path)` | `boolean` |
| 15 | `file_move` | `file_move(src, dst)` | `nil` |
| 16 | `file_copy` | `file_copy(src, dst)` | `nil` |
| 17 | `file_size` | `file_size(path)` | `number` |
| 18 | `file_type` | `file_type(path)` | `string` or `nil` |
| 19 | `list_dir` | `list_dir(path)` | `list` of strings |
| 20 | `make_dir` | `make_dir(path)` | `nil` |
| 21 | `remove_dir` | `remove_dir(path)` | `nil` |
| 22 | `path_join` | `path_join(base, part)` | `string` |
| 23 | `path_base` | `path_base(path)` | `string` |
| 24 | `path_dir` | `path_dir(path)` | `string` |
| 25 | `path_ext` | `path_ext(path)` | `string` |
| 26 | `file_open` | `file_open(path, mode)` | `number` (handle) |
| 27 | `file_close` | `file_close(handle)` | `nil` |
| 28 | `file_read` | `file_read(handle, offset, n)` | `bytes` |
| 29 | `file_write` | `file_write(handle, offset, data)` | `number` |
| 30 | `file_truncate` | `file_truncate(handle, size)` | `nil` |
| 31 | `file_sync` | `file_sync(handle)` | `nil` |

### String (11)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 32 | `upper` | `upper(str)` | `string` |
| 33 | `lower` | `lower(str)` | `string` |
| 34 | `trim` | `trim(str)` | `string` |
| 35 | `split` | `split(str, delimiter)` | `list` of strings |
| 36 | `join` | `join(list, delimiter)` | `string` |
| 37 | `repeat` | `repeat(str, count)` | `string` |
| 37a | `replace` | `replace(str, old, new)` | `string` |
| 37b | `replace_all` | `replace_all(str, old, new)` | `string` |
| 38 | `contains` | `contains(container, item)` | `boolean` (string, list, or map) |
| 39 | `substring` | `substring(str, start, end)` | `string` |
| 40 | `index_of` | `index_of(haystack, needle)` | `number` (string or list) |

### List (14)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 41 | `len` | `len(value)` | `number` |
| 42 | `push` | `push(list, value)` | The mutated `list` |
| 43 | `pop` | `pop(list)` | The removed element, or `nil` if empty |
| 44 | `at` | `at(collection, index)` | `any` or `nil` |
| 45 | `slice_list` | `slice_list(list, range)` | `list` |
| 46 | `slice` | `slice(value, start, end)` | `list`, `string`, or `bytes` |
| 47 | `sort` | `sort(list, comparator?)` | A new `list` |
| 48 | `sorted_by` | `sorted_by(list, keyFn)` | `list` |
| 49 | `range` | `range(start, end)` | `list` of numbers |
| 50 | `map` | `map(list, fn)` | `list` |
| 51 | `filter` | `filter(list, fn)` | `list` |
| 52 | `reduce` | `reduce(list, fn, initial)` | `any` |
| 53 | `each` | `each(list, fn)` | `nil` |
| 53a | `unique` | `unique(list)` | `list` with duplicates removed |

### Bytes & Binary (15)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 54 | `to_bytes` | `to_bytes(value)` | `bytes` |
| 55 | `from_bytes` | `from_bytes(value)` | `string` |
| 56 | `bytes_append` | `bytes_append(bytes, chunk, ...)` | `bytes` |
| 57 | `bytes_to_int` | `bytes_to_int(bytes, offset?, n?)` | `number` |
| 58 | `int_to_bytes` | `int_to_bytes(value, n?)` | `bytes` |
| 59 | `bytes_compare` | `bytes_compare(a, b)` | `number` |
| 60 | `hex_encode` | `hex_encode(bytes)` | `string` |
| 61 | `hex_decode` | `hex_decode(str)` | `bytes` |
| 62 | `bit_and` | `bit_and(a, b)` | `number` |
| 63 | `bit_or` | `bit_or(a, b)` | `number` |
| 64 | `bit_xor` | `bit_xor(a, b)` | `number` |
| 65 | `bit_not` | `bit_not(a)` | `number` |
| 66 | `bit_lshift` | `bit_lshift(a, n)` | `number` |
| 67 | `bit_rshift` | `bit_rshift(a, n)` | `number` |
| 68 | `crc32` | `crc32(value)` | `number` |

### CSV (2)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 69 | `csv_parse` | `csv_parse(text)` | `list` |
| 70 | `csv_format` | `csv_format(data)` | `string` |

### Map (5)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 71 | `get` | `get(map, key)` | `any` or `nil` |
| 72 | `set` | `set(map, key, value)` | The mutated `map` |
| 73 | `keys` | `keys(map)` | `list` of strings |
| 74 | `values` | `values(map)` | `list` |
| 74a | `map_delete` | `map_delete(map, key)` | `nil` |

### Math (8)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 75 | `abs` | `abs(x)` | `number` |
| 76 | `min` | `min(a, b)` | `number` |
| 77 | `max` | `max(a, b)` | `number` |
| 78 | `pow` | `pow(base, exp)` | `number` |
| 79 | `sqrt` | `sqrt(x)` | `number` |
| 80 | `round` | `round(x)` | `number` |
| 80a | `ceil` | `ceil(x)` | `number` |
| 80b | `floor` | `floor(x)` | `number` |

### Network & HTTP (6)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 81 | `http_get` | `http_get(url)` | `string` |
| 82 | `http_post` | `http_post(url, body)` | `string` |
| 83 | `http_get_json` | `http_get_json(url)` | `any` (map, list, number, string, boolean, or nil) |
| 83a | `http_request` | `http_request(method, url, headers?, body?)` | `map` — `{status, headers, body}` |
| 84 | `parse_json` | `parse_json(json_string)` | `any` or `nil` on parse error |
| 85 | `to_json` | `to_json(value)` | `string` |

### TCP (9)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 86 | `tcp_listen` | `tcp_listen(address)` | `listener handle` |
| 87 | `tcp_connect` | `tcp_connect(address)` | `connection handle` |
| 88 | `tcp_connect_tls` | `tcp_connect_tls(host, port, servername?, insecure?)` | `connection handle` |
| 89 | `tcp_accept` | `tcp_accept(listener)` | `connection handle` |
| 90 | `tcp_read` | `tcp_read(conn, max_bytes)` | `string` |
| 91 | `tcp_read_bytes` | `tcp_read_bytes(conn, n)` | `bytes` |
| 92 | `tcp_set_read_timeout` | `tcp_set_read_timeout(conn, ms)` | `nil` |
| 93 | `tcp_write` | `tcp_write(conn, data)` | `nil` |
| 94 | `tcp_close` | `tcp_close(handle)` | `nil` |

### Regex (2)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 94 | `regex_match` | `regex_match(pattern, str)` | `boolean` |
| 95 | `regex_replace` | `regex_replace(pattern, replacement, str)` | `string` |

### Date & Time (3)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 96 | `now` | `now()` | `number` |
| 94a | `time_ms` | `time_ms()` | `number` |
| 95 | `format_time` | `format_time(timestamp, layout)` | `string` |

### Random (2)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 96 | `random` | `random()` | `number` |
| 97 | `random_range` | `random_range(min, max)` | `number` (integer) |

### Cryptography (8)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 98 | `secure_random` | `secure_random(byte_count)` | `string` (hex) |
| 99 | `secure_random_int` | `secure_random_int()` | `number` |
| 100 | `secure_random_range` | `secure_random_range(min, max)` | `number` (integer) |
| 101 | `secure_random_bytes` | `secure_random_bytes(byte_count)` | `bytes` |
| 102 | `encrypt` | `encrypt(key, plaintext[, ad])` | `string` (base64) |
| 103 | `decrypt` | `decrypt(key, ciphertext[, ad])` | `string` |
| 104 | `hmac_sha256` | `hmac_sha256(key, message)` | `string` (64 hex) |
| 105 | `hmac_sha512` | `hmac_sha512(key, message)` | `string` (128 hex) |

### Encoding (2)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 101 | `base64_encode` | `base64_encode(str)` | `string` |
| 99 | `base64_decode` | `base64_decode(str)` | `string` |

### Hashing (4)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 100 | `sha256` | `sha256(text)` | `string` |
| 101 | `md5` | `md5(text)` | `string` |
| 102 | `sha1` | `sha1(text)` | `string` |
| 103 | `sha512` | `sha512(text)` | `string` |

### Database (9) — module (see `pipe -get sqlite`)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| — | `db_open` | `db_open(path)` | *module* |
| — | `db_close` | `db_close(handle)` | *module* |
| — | `db_exec` | `db_exec(handle, sql)` | *module* |
| — | `db_query` | `db_query(handle, sql)` | *module* |
| — | `q` | `q(handle, sql)` | *module* |
| — | `exec` | `exec(handle, sql)` | *module* |
| — | `row_get` | `row_get(row, key)` | *module* |
| — | `row_eq` | `row_eq(row, key, val)` | *module* |
| — | `row_ne` | `row_ne(row, key, val)` | *module* |

### Type Checks (6)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 108 | `type_of` | `type_of(value)` | `string` — one of `"number"`, `"string"`, `"list"`, `"map"`, `"bytes"`, `"nil"`, `"function"`, `"boolean"`, `"result"` |
| 109 | `is_num` | `is_num(value)` | `boolean` |
| 110 | `is_str` | `is_str(value)` | `boolean` |
| 111 | `is_list` | `is_list(value)` | `boolean` |
| 112 | `is_map` | `is_map(value)` | `boolean` |
| 113 | `is_nil` | `is_nil(value)` | `boolean` |

### Conversion (2)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 114 | `to_str` | `to_str(value)` | `string` |
| 115 | `to_num` | `to_num(str)` | `number` or `nil` |

### Result Type (6)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 116 | `Ok` | `Ok(value)` | `Result` (Ok variant) |
| 117 | `Err` | `Err(message)` | `Result` (Err variant) |
| 118 | `is_ok` | `is_ok(result)` | `boolean` |
| 119 | `is_err` | `is_err(result)` | `boolean` |
| 120 | `unwrap` | `unwrap(result)` | `any` |
| 121 | `unwrap_or` | `unwrap_or(result, default)` | `any` |

### AI — Configuration (6)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 122 | `ai_provider` | `ai_provider(name, {model, host, timeout, thinking, effort}?)` | `string` |
| 123 | `ai_model` | `ai_model(name)` | `string` |
| 124 | `ai_set_key` | `ai_set_key(provider, key)` | `string` |
| 125 | `ai_host` | `ai_host(url)` | `string` |
| 126 | `ai_cache` | `ai_cache(option)` | `string` |
| 127 | `ai_timeout` | `ai_timeout(seconds)` | `nil` |

### AI — Low-level Chat (2)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 128 | `ai_chat` | `ai_chat(system_prompt, user_prompt)` | `string` |
| 129 | `ai_chat_json` | `ai_chat_json(system_prompt, user_prompt)` | `any` |

### AI — Streaming (1)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 130 | `ai_stream` | `ai_stream(system_prompt, user_prompt)` | `string` |

### AI — High-level Convenience (7)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 131 | `summarize` | `summarize(text)` | `string` |
| 132 | `generate_json` | `generate_json(instruction, schema)` | `any` |
| 133 | `translate` | `translate(text, target_language)` | `string` |
| 134 | `classify` | `classify(text, categories)` | `string` |
| 135 | `extract` | `extract(text, schema)` | `any` |
| 136 | `generate` | `generate(prompt)` | `string` |
| 137 | `ask` | `ask(question)` | `string` |

### AI — Parallel (3)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 138 | `ai_batch` | `ai_batch(system_prompt, items)` | `list` |
| 139 | `ai_parallel` | `ai_parallel(concurrency, system_prompt, items)` | `list` |
| 140 | `ai_rate_limit` | `ai_rate_limit(calls_per_second)` | `nil` |

### AI — Tool Calling (2)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 141 | `ai_tool` | `ai_tool(name, description, parameters, function)` | `string` |
| 142 | `ai_with_tools` | `ai_with_tools(system_prompt, user_prompt, max_rounds?)` | `string` |

### AI — Agents (3)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 143 | `agent` | `agent(name, system_prompt)` | `string` |
| 144 | `agent_ask` | `agent_ask(name, message)` | `string` |
| 145 | `agent_clear` | `agent_clear(name)` | `string` |

### AI — Embeddings (5)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 146 | `embed` | `embed(text)` | `list` of numbers |
| 147 | `embed_batch` | `embed_batch(items)` | `list` |
| 148 | `cosine_sim` | `cosine_sim(vec_a, vec_b)` | `number` |
| 149 | `dot_product` | `dot_product(vec_a, vec_b)` | `number` |
| 150 | `nearest` | `nearest(query_vec, doc_vectors, k)` | `list` |

### AI — Search (2)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 151 | `web_search` | `web_search(query)` | `list` |
| 152 | `wiki_search` | `wiki_search(query)` | `list` |

### AI — Cost Tracking (5)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 153 | `ai_cost` | `ai_cost()` | `map` — `{cost_usd, calls, cache_hits, cache_misses}` |
| 154 | `ai_tokens` | `ai_tokens()` | `number` |
| 155 | `ai_cache_hits` | `ai_cache_hits()` | `number` |
| 156 | `ai_cache_misses` | `ai_cache_misses()` | `number` |
| 157 | `try_ai_log` | `try_ai_log()` | `list` |

### Sandbox (5)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 158 | `sandbox_profile` | `sandbox_profile(name)` | `string` |
| 159 | `set_sandbox` | `set_sandbox(profile)` | `string` |
| 160 | `with_sandbox` | `with_sandbox(profile, fn)` | `any` |
| 161 | `audit_log` | `audit_log()` | `list` |
| 162 | `budget_spent` | `budget_spent()` | `number` |

### Test Assertions (8)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 163 | `assert` | `assert(condition)` | `nil` |
| 164 | `assert_eq` | `assert_eq(expected, actual)` | `nil` |
| 165 | `assert_not_eq` | `assert_not_eq(unexpected, actual)` | `nil` |
| 166 | `assert_lt` | `assert_lt(a, b)` | `nil` |
| 167 | `assert_gt` | `assert_gt(a, b)` | `nil` |
| 168 | `assert_error` | `assert_error(fn)` | `nil` |
| 169 | `assert_near` | `assert_near(expected, actual[, epsilon])` | `nil` |
| 170 | `assert_contains` | `assert_contains(container, item)` | `nil` |

### MCP (13)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 171 | `mcp_server` | `mcp_server(name, version)` | `nil` |
| 172 | `mcp_serve_stdio` | `mcp_serve_stdio` | `nil` |
| 173 | `mcp_serve_sse` | `mcp_serve_sse(addr)` | `nil` |
| 174 | `mcp_tools` | `mcp_tools` | `list` |
| 175 | `mcp_resource` | `mcp_resource(uri, name, mime, read_fn)` | `nil` |
| 176 | `mcp_resource_template` | `mcp_resource_template(template, name, mime, read_fn)` | `nil` |
| 177 | `mcp_prompt` | `mcp_prompt(name, description, args_map, build_fn)` | `nil` |
| 178 | `mcp_resources` | `mcp_resources` | `list` |
| 179 | `mcp_read_resource` | `mcp_read_resource(uri)` | `string` |
| 180 | `mcp_prompts` | `mcp_prompts` | `list` |
| 181 | `mcp_prompt_get` | `mcp_prompt_get(name, args?)` | `string` |
| 182 | `mcp_use_stdio` | `mcp_use_stdio(command, args...)` | `string` |
| 183 | `mcp_use_sse` | `mcp_use_sse(url)` | `string` |

### AI — Swarms (3)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 184 | `swarm_agent` | `swarm_agent(name, config)` | `bool` |
| 185 | `ai_swarm` | `ai_swarm(task, entry_agent, max_rounds?)` | `string` |
| 186 | `ai_swarm_trace` | `ai_swarm_trace(task, entry_agent, max_rounds?)` | `map` |

### AI — Vision (1)

| # | Function | Signature | Returns |
|---|----------|-----------|---------|
| 187 | `ai_vision` | `ai_vision(image, prompt, max_tokens?)` | `string` |

---

**Total: 233 built-in functions**
