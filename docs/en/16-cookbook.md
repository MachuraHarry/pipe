# Cookbook

Practical recipes demonstrating common programming patterns in Pipe. All examples are complete and runnable.

## 1. Fibonacci Numbers

```pipe
fib: fn n
  if n <= 1
    n
  else
    (fib n - 1) + (fib n - 2)

print fib 10
-- Output: 55
```

With memoization for performance:

```pipe
memo: {}
fib: fn n
  if n <= 1
    n
  else
    cached: get memo (to_str n)
    if not is_nil cached
      cached
    else
      result: (fib n - 1) + (fib n - 2)
      set memo (to_str n) result
      result

print fib 40
-- Output: 102334155
```

## 2. FizzBuzz

```pipe
for i in range 1 101
  out: ""
  if i % 3 == 0
    out: out ++ "Fizz"
  if i % 5 == 0
    out: out ++ "Buzz"
  if out == ""
    print i
  else
    print out
```

## 3. Prime Numbers Finder

```pipe
is_prime: fn n
  if n < 2
    false
  else
    result: true
    i: 2
    while i * i <= n and result
      if n % i == 0
        result: false
      i: i + 1
    result

primes: []
for n in range 2 101
  if is_prime n
    push primes n

print primes
-- Output: [2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59, 61, 67, 71, 73, 79, 83, 89, 97]
```

## 4. Factorial

```pipe
fact: fn n
  if n <= 1
    1
  else
    n * (fact n - 1)

print fact 5
-- Output: 120
```

Iterative version:

```pipe
fact: fn n
  result: 1
  for i in range 1 n + 1
    result: result * i
  result

print fact 5
-- Output: 120
```

## 5. Palindrome Checker

```pipe
is_palindrome: fn s
  s == (s > reverse_custom)

reverse_custom: fn s
  chars: split s ""
  result: ""
  i: (len chars) - 1
  while i >= 0
    result: result ++ (at chars i)
    i: i - 1
  result

print is_palindrome "racecar"
-- Output: true
print is_palindrome "hello"
-- Output: false
```

## 6. Celsius to Fahrenheit Converter

```pipe
c_to_f: fn c
  c * 9[0] / 5[0] + 32[0]

f_to_c: fn f
  (f - 32[0]) * 5[0] / 9[0]

print "0 C = " ++ (to_str c_to_f 0) ++ " F"
-- Output: 0 C = 32 F
print "100 C = " ++ (to_str c_to_f 100) ++ " F"
-- Output: 100 C = 212 F
print "32 F = " ++ (to_str f_to_c 32) ++ " C"
-- Output: 32 F = 0 C
```

## 7. Caesar Cipher

```pipe
caesar: fn text shift
  result: ""
  for ch in split text ""
    code: at ch 0
    result: result ++ shift_char code shift
  result

shift_char: fn ch shift
  if ch >= "A" and ch <= "Z"
    at "ABCDEFGHIJKLMNOPQRSTUVWXYZ" ((at ch 0) + shift) % 26
  else if ch >= "a" and ch <= "z"
    at "abcdefghijklmnopqrstuvwxyz" ((at ch 0) + shift) % 26
  else
    ch

print caesar "Hello World!" 3
-- Output: Khoor Zruog!
print caesar "Khoor Zruog!" -3
-- Output: Hello World!
```

## 8. List Sum

```pipe
nums: [1, 2, 3, 4, 5]

total: 0
for n in nums
  total: total + n

print total
-- Output: 15
```

Using `reduce`:

```pipe
nums: [1, 2, 3, 4, 5]
add: fn a b
    a + b
result: reduce nums add 0
print result
-- Output: 15
```

## 9. Text Statistics

```pipe
text: "The quick brown fox jumps over the lazy dog"

-- Word count
words: split text " "
print "Words: " ++ (to_str len words)

-- Character count (excluding spaces)
chars: 0
for w in words
  chars: chars + (len w)
print "Characters (no spaces): " ++ (to_str chars)

-- Average word length
if len words > 0
  avg: chars / (len words)
  print "Average word length: " ++ (to_str avg)

-- Most frequent letter
freq: {}
for ch in split text ""
  if ch != " "
    count: get freq ch
    if is_nil count
      set freq ch 1
    else
      set freq ch count + 1

print "Letter frequencies: " ++ (to_json freq)
```

## 10. Calculator with Match

```pipe
calculate: fn a op b
  match op
    | "+" -> a + b
    | "-" -> a - b
    | "*" -> a * b
    | "/" -> a / b
    | "%" -> a % b
    | "**" -> pow a b
    | _ -> "unknown operator"

print calculate 10 "+" 5
-- Output: 15
print calculate 10 "/" 3
-- Output: 3
print calculate 2 "**" 8
-- Output: 256
```

## 11. HTTP Client

### GET request

```pipe
resp: http_get "https://api.github.com/repos/pipe-lang/pipe"
print "Status: " ++ (to_str resp.status)
print "Body length: " ++ (to_str len resp.body)
```

### POST request with JSON

```pipe
payload: to_json ({title: "Test", body: "Hello from Pipe", userId: "1"})
resp: http_post "https://jsonplaceholder.typicode.com/posts" payload
print "Status: " ++ (to_str resp.status)
data: parse_json resp.body
print "Created ID: " ++ (to_str data.id)
```

### JSON API

```pipe
resp: http_get_json "https://api.github.com/users/octocat"
print "Login: " ++ resp.login
print "Repos: " ++ (to_str resp.public_repos)
```

## 12. File Operations

### Read entire file

```pipe
content: read_file "data.txt"
print content
```

### Write to file

```pipe
write_file "output.txt" "Hello, Pipe!"
```

### Append to file

```pipe
append_file "log.txt" "line 1\n"
append_file "log.txt" "line 2\n"
```

### Read file line by line

```pipe
lines: read_lines "data.txt"
for line in lines
  if contains line "@"
    print line
```

### Check file existence

```pipe
if file_exists "config.json"
  print "Config file found"
else
  print "No config file"
```

## 13. Directory Operations

### Create directory

```pipe
make_dir "data/backups"
```

### List directory contents

```pipe
entries: list_dir "."
for entry in entries
  print entry
```

### Path utilities

```pipe
p: "docs/en/index.md"
print "Base: " ++ (path_base p)
-- Base: index.md
print "Dir:  " ++ (path_dir p)
-- Dir:  docs/en
print "Ext:  " ++ (path_ext p)
-- Ext:  .md
print "Join: " ++ (path_join "docs" "en" "index.md")
-- Join: docs/en/index.md
```

### Copy and move files

```pipe
file_copy "source.txt" "dest.txt"
file_move "old.txt" "new.txt"
```

### Cleanup

```pipe
file_delete "temp.txt"
remove_dir "old_data"
```

## 14. TCP Echo Server + Client

### Server

```pipe
print "Starting echo server on port 9999..."
ln: tcp_listen "127.0.0[1]" 9999

for i in range 5
  conn: tcp_accept ln
  msg: tcp_read conn
  print "Received: " ++ msg
  tcp_write conn "Echo: " ++ msg
  tcp_close conn

tcp_close ln
print "Server done"
```

### Client

```pipe
sleep 100

conn: tcp_connect "127.0.0[1]" 9999
tcp_write conn "Hello, Server!"
response: tcp_read conn
print response
tcp_close conn
```

## 15. JSON Configuration

### Write config

```text
config: {
  app_name: "MyApp",
  version: "1.0.0",
  debug: true,
  limits: {
    max_connections: 100,
    timeout_ms: 5000
  },
  allowed_hosts: ["localhost", "127.0.0[1]"]

json_str: to_json config
write_file "config.json" json_str
print "Config written"
```

### Read config

```pipe
json_str: read_file "config.json"
config: parse_json json_str

print "App: " ++ config.app_name
print "Version: " ++ config.version
print "Max connections: " ++ (to_str config.limits.max_connections)
print "Hosts: " ++ (to_json config.allowed_hosts)
```

## 16. Regex Validation

### Email validation

```pipe
is_email: fn s
  regex_match "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$" s

print is_email "user@example.com"
-- Output: true
print is_email "not-valid"
-- Output: false
```

### Phone number masking

```pipe
mask_phone: fn phone
  regex_replace "\\d(?=\\d{4})" phone "*"

print mask_phone "1234567890"
-- Output: ******7890
```

### Extract numbers from string

```pipe
extract_numbers: fn s
  regex_replace "[^0-9]" s ""

print extract_numbers "Price: $42[99] - Code: 1234"
-- Output: 42991234
```

## 17. Date and Time Formatting

```pipe
ts: now
print "Unix timestamp: " ++ (to_str ts)

default_fmt: format_time ts
print "Default: " ++ default_fmt
-- Output: 2026-01-15 14:30:00

custom_fmt: format_time ts "2006-01-02"
print "Date only: " ++ custom_fmt

time_fmt: format_time ts "15:04:05"
print "Time only: " ++ time_fmt

full: format_time ts "Monday, January 2 2006"
print "Full: " ++ full
```

## 18. Random Numbers

### Dice roller

```text
dice: fn sides
  random_range 1 sides

roll_d6: (fn
  dice 6)

print "d6: " ++ (to_str roll_d6)
print "d20: " ++ (to_str dice 20)

roll_3d6: (fn
  (dice 6) + (dice 6) + (dice 6)
)

print "3d6: " ++ (to_str roll_3d6)
```

### Random float and selection

```pipe
print "Random float: " ++ (to_str random)

items: ["apple", "banana", "cherry", "date"]
idx: random_range 0 (len items)
print "Random pick: " ++ (at items idx)
```

## 19. Shell Command Execution

### Basic exec

```pipe
result: exec "echo Hello from Pipe"
print result.output
```

### Exit code and error handling

```pipe
result: exec "ls /nonexistent"
if result.status != 0
  print "Command failed: " ++ result.error
else
  print result.output
```

### Environment variables

```pipe
home: env "HOME"
print "Home: " ++ home

path: env "PATH"
print "PATH: " ++ path
```

### Sleep

```pipe
print "Starting..."
sleep 1000
print "1 second later"
```

## 20. Error Handling with Result Type

### Safe division

```pipe
safe_div: fn a b
  if b == 0
    Err "division by zero"
  else
    Ok (a / b)

result: safe_div 10 2
if is_ok result
  print "Result: " ++ (to_str unwrap result)
else
  print "Error: " ++ result.Err

result2: safe_div 10 0
print unwrap_or result2 "N/A"
-- Output: N/A
```

### Try-catch

```pipe
try
  content: read_file "missing.txt"
  print content
catch e
  print "Could not read file: " ++ e
```

### Chaining results

```pipe
parse_and_divide: fn a_str b_str
  a: to_num a_str
  b: to_num b_str
  if is_num a and is_num b
    safe_div a b
  else
    Err "invalid numbers"

r: parse_and_divide "42" "7"
if is_ok r
  print "Result: " ++ (to_str unwrap r)
```

## 21. Closures

### Function factory

```pipe
make_greeter: fn greeting
  fn name
    greeting ++ ", " ++ name ++ "!"

hello: make_greeter "Hello"
hi:    make_greeter "Hi"

print hello "Alice"
-- Output: Hello, Alice!
print hi "Bob"
-- Output: Hi, Bob!
```

### make_multiplier

```pipe
make_multiplier: fn factor
  fn x
    x * factor

double: make_multiplier 2
triple: make_multiplier 3

print double 5
-- Output: 10
print triple 5
-- Output: 15
```

### Counter

```text
make_counter: fn
    count: 0
    fn
        count: count + 1
        count

counter: make_counter
print counter
-- Output: 1
print counter
-- Output: 2
print counter
-- Output: 3
```

## 22. Binary Search

```pipe
binary_search: fn arr target
  low: 0
  high: (len arr) - 1

  while low <= high
    mid: (low + high) / 2
    guess: at arr mid
    if guess == target
      return (to_str target) ++ " found at index " ++ (to_str mid)
    if guess < target
      low: mid + 1
    else
      high: mid - 1

  (to_str target) ++ " not found"

nums: [1, 3, 5, 7, 9, 11, 13, 15, 17]
print binary_search nums 7
-- Output: 7 found at index 3
print binary_search nums 10
-- Output: 10 not found
```

## 23. Enum + Match

```pipe
Red: 0
Green: 1
Blue: 2
Yellow: 3

describe: fn c
  match c
    | "Red" -> "The color of passion"
    | "Green" -> "The color of nature"
    | "Blue" -> "The color of the sky"
    | "Yellow" -> "The color of sunshine"
    | _ -> "Unknown color"

print describe "Red"
-- Output: The color of passion
print describe "Blue"
-- Output: The color of the sky
```

Using enum values for control flow:

```pipe
Pending: 0
Processing: 1
Complete: 2
Failed: 3

handle: fn status
  match status
    | "Pending" -> "Waiting..."
    | "Processing" -> "In progress"
    | "Complete" -> "Done!"
    | "Failed" -> "Error occurred"
    | _ -> "Invalid status"

print handle "Pending"
```

Pipeline with match:

```pipe
status: "Pending"
status
  > handle
  > upper
  > print
-- Output: WAITING...
```

## 24. Import & Module System

### File `math_utils.pipe`

```pipe
export fn square x
  x * x

export fn cube x
  x * x * x

export fn is_even x
  x % 2 == 0
```

### File `main.pipe`

```pipe
import "math_utils.pipe"

print square 4
-- Output: 16
print cube 3
-- Output: 27
print is_even 7
-- Output: false

for i in range 1 6
  print (to_str i) ++ "^2 = " ++ (to_str square i)
```

### Import with namespace alias

```pipe
import "math_utils.pipe" as math

print math.square 5
print math.cube 4
```

### Re-exporting

```pipe
-- advanced.pipe
import "math_utils.pipe"

export fn power_of_two x
  square x
```

## 25. Defer for Cleanup

### File resource management

```pipe
process_file: fn path
  print "Opening " ++ path
  content: read_file path
  defer print "Closing " ++ path
  defer print "Cleanup done"

  print "Processing..."
  print "Content: " ++ content

  print "Returning result"

process_file "data.txt"
```

Output:
```
Opening data.txt
Processing...
Content: <file content>
Returning result
Cleanup done
Closing data.txt
```

### Timing with defer

```pipe
timed: fn
    print "Working..."
    sleep 500
    print "Operation completed at: " ++ (format_time (now))

print "---"
timed
  print "Still working..."
  sleep 500

timed_fn
```

### Multiple cleanup operations

```text
-- defer runs after the function body
cleanup: fn
    print "Connection closed"

  print "Performing operation..."
  result: {status: "ok"}

  print "Operation result: " ++ result.status

### SQLite Database

```pipe
import "sqlite.pipe"

h: db_open ":memory:"

db_exec h "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, score INTEGER)"
db_exec h "INSERT INTO users VALUES (1, 'Alice', 95), (2, 'Bob', 72)"
db_exec h "INSERT INTO users VALUES (3, 'Charlie', 88)"

-- All users
rows: db_query h "SELECT * FROM users"
for row in rows
  print "#" ++ (to_str (get row "id")) ++ " " ++ (get row "name")

-- Filtered
high: db_query h "SELECT name, score FROM users WHERE score > 80"

-- Aggregate
stats: db_query h "SELECT AVG(score) as avg_score FROM users"

db_close h
```

---

Note: `defer` expressions execute in LIFO order (last-in, first-out). The last `defer` declared is the first to execute when the scope exits.
