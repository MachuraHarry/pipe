# Getting Started with Pipe

## What is Pipe?

Pipe is an indentation-based pipeline scripting language written in Go. It compiles to a single statically-linked binary of approximately 10MB, with no external dependencies. The language centers around the pipeline operator (`>`), allowing data transformations to be expressed as a clear, top-to-bottom flow of operations.

Key characteristics:

- **Indentation-based**: blocks are defined by consistent whitespace, not braces or keywords
- **Pipeline-centric**: the `>` operator makes data flow the primary metaphor
- **Dual execution modes**: a tree-walking interpreter for development and a bytecode VM for performance
- **Single binary**: self-contained ~10MB executable
- **First-class functions**: closures, higher-order functions, and tail-call optimization
- **REPL included**: interactive read-eval-print loop for experimentation

## Prerequisites

To build Pipe from source, you need:

- **Go 1.21** or later
- **GNU Make**
- A Unix-like environment (Linux or macOS; Pipe has not been tested on Windows)

Check your Go version:

```sh
go version
```

## Installation

Clone the repository, build, and verify:

```sh
git clone https://github.com/pipe-lang/pipe.git
cd pipe
make build
./bin/pipe hello.pipe
```

The `make build` command produces `./bin/pipe` — the single binary that contains the interpreter, VM, and REPL.

To install Pipe system-wide (optional):

```sh
sudo cp ./bin/pipe /usr/local/bin/
```

## Hello World

Create a file named `hello.pipe`:

```pipe
print "Hello, World!"
```

Run it:

```sh
./bin/pipe hello.pipe
```

Output:

```
Hello, World!
```

## The REPL

Launch the interactive REPL by running `./bin/pipe` with no arguments:

```sh
./bin/pipe
```

The `>>> ` prompt accepts Pipe expressions. Multi-line input uses the `... ` continuation prompt.

```
>>> x: 42
>>> x * 2
84
>>> fn greet name
...   "Hello, " ++ name
...
>>> greet "Pipe"
Hello, Pipe
```

### REPL Commands

| Command | Alias | Description |
|---|---|---|
| `:quit` | `:q` | Exit the REPL |
| `:help` | `:h` | Display available commands |
| `:clear` | `:c` | Clear the screen |
| `:vm` | | Toggle VM execution mode for subsequent input |
| `:history` | | Show command history |
| `:!N` | | Re-execute history entry N (e.g. `:!3`) |

You can also exit with **Ctrl+D** (EOF) or enter a **blank line** to terminate multi-line input and evaluate the block.

## Execution Modes

Pipe has two execution backends:

### Tree-Walker (default)

```sh
./bin/pipe file.pipe       # interprets the AST directly
```

The tree-walker evaluates the abstract syntax tree by traversing nodes recursively. It is the default mode and is ideal for development and debugging.

### Virtual Machine (`-vm` flag)

```sh
./bin/pipe -vm file.pipe   # compiles to bytecode, then executes
```

The VM mode compiles the AST to a custom bytecode instruction set and executes it on a stack-based virtual machine. This provides better runtime performance, especially for compute-heavy or deeply recursive code.

### Comparison

| Aspect | Tree-Walker | VM |
|---|---|---|
| Start-up | Near-instant | Slightly slower (compilation) |
| Execution speed | Slower for long loops | Faster for hot paths |
| Debugging | Easier to trace | Opaque bytecode |
| Default | Yes | No |
| REPL toggle | Default mode | `:vm` to switch |

## CLI Argument Passing

Arguments passed on the command line after the filename are available inside the program via the built-in `args` list:

```sh
./bin/pipe script.pipe foo bar baz
```

Inside `script.pipe`:

```pipe
print args
-- Output: ["foo", "bar", "baz"]

for arg in args
  print arg
```

`args` contains the user-supplied arguments only — the script filename is not included.

## The `-h` Flag

Print help information:

```sh
./bin/pipe -h
```

Output shows the usage line, available flags, and a brief description of the language.

## A First Small Program

Here is a slightly longer example demonstrating variables, functions, pipelines, and loops:

```pipe
double: fn x
  x * 2

square: fn x
  x * x

nums: [1, 2, 3, 4, 5]

for n in nums
  result: n > double > square
  print n ++ " -> " ++ (to_str result)
```

Output:

```
1 -> 4
2 -> 16
3 -> 36
4 -> 64
5 -> 100
```

This program defines two functions (`double` and `square`), creates a list of numbers, and uses a `for-in` loop with a pipeline (`n > double > square`) to transform each number through sequential function application.
