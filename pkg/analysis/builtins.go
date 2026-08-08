package analysis

// BuiltinDoc describes one builtin function for IntelliSense purposes.
type BuiltinDoc struct {
	Name        string
	Signature   string
	Params      []Param
	ReturnType  string
	Description string
	Category    string
}

// Param is a single named function parameter.
type Param struct {
	Name string
	Type string
}

// Categories used for completion grouping / sorting.
const (
	CatIO       = "IO & System"
	CatFile     = "File System"
	CatString   = "String"
	CatCSV      = "CSV"
	CatList     = "List"
	CatMap      = "Map"
	CatMath     = "Math"
	CatNet      = "Network & HTTP"
	CatTCP      = "TCP"
	CatServer   = "HTTP Server"
	CatRegex    = "Regex"
	CatDate     = "Date & Time"
	CatRandom   = "Random"
	CatCrypto   = "Cryptography"
	CatEncode   = "Encoding"
	CatHash     = "Hashing"
	CatBytes    = "Bytes & Binary"
	CatType     = "Type Checks"
	CatConv     = "Conversion"
	CatResult   = "Result"
	CatAIConf   = "AI Configuration"
	CatAIChat   = "AI Chat"
	CatAIHi     = "AI High-level"
	CatAIStream = "AI Streaming"
	CatAIPar    = "AI Parallel"
	CatAITool   = "AI Tool Calling"
	CatAIAgent  = "AI Agents"
	CatAIEmbed  = "AI Embeddings"
	CatAISearch = "AI Search"
	CatSandbox  = "Sandbox"
	CatTest     = "Test Assertions"
)

func p(name, typ string) Param { return Param{Name: name, Type: typ} }

// builtinDocs is the source of truth for builtin signatures and documentation.
// Signatures and arities are verified against pkg/object.Builtins and the
// builtin implementations; descriptions come from docs/en/10-builtin-reference.md.
var builtinDocs = []BuiltinDoc{
	// ---- IO & System ----
	{Name: "print", Signature: "print(value...)", Params: []Param{p("value", "any")}, ReturnType: "nil",
		Description: "Prints one or more values to stdout, separated by spaces and followed by a newline.", Category: CatIO},
	{Name: "input", Signature: "input(prompt?)", Params: []Param{p("prompt", "string")}, ReturnType: "string",
		Description: "Displays the optional prompt, reads a line from stdin, and returns it as a string.", Category: CatIO},
	{Name: "exec", Signature: "exec(command)", Params: []Param{p("command", "string")}, ReturnType: "string",
		Description: "Executes a system command via the shell and returns the combined stdout/stderr output.", Category: CatIO},
	{Name: "env", Signature: "env(name)", Params: []Param{p("name", "string")}, ReturnType: "string",
		Description: "Returns the value of the environment variable name, or an empty string if unset.", Category: CatIO},
	{Name: "sleep", Signature: "sleep(ms)", Params: []Param{p("ms", "number")}, ReturnType: "nil",
		Description: "Pauses execution for ms milliseconds.", Category: CatIO},
	{Name: "args", Signature: "args()", Params: nil, ReturnType: "list",
		Description: "Returns CLI arguments passed to the script as a list of strings.", Category: CatIO},
	{Name: "read_stdin", Signature: "read_stdin()", Params: nil, ReturnType: "string",
		Description: "Reads the entire standard input and returns it as a string.", Category: CatIO},

	// ---- File System ----
	{Name: "read_file", Signature: "read_file(path)", Params: []Param{p("path", "string")}, ReturnType: "string",
		Description: "Reads the entire contents of a file and returns it as a string.", Category: CatFile},
	{Name: "write_file", Signature: "write_file(path, content)", Params: []Param{p("path", "string"), p("content", "string")}, ReturnType: "nil",
		Description: "Writes content to path, overwriting if it exists. Creates parent directories if needed.", Category: CatFile},
	{Name: "append_file", Signature: "append_file(path, content)", Params: []Param{p("path", "string"), p("content", "string")}, ReturnType: "nil",
		Description: "Appends content to the end of the file at path. Creates the file if it doesn't exist.", Category: CatFile},
	{Name: "read_lines", Signature: "read_lines(path)", Params: []Param{p("path", "string")}, ReturnType: "list",
		Description: "Reads a file and returns its lines as a list of strings (without trailing newlines).", Category: CatFile},
	{Name: "file_exists", Signature: "file_exists(path)", Params: []Param{p("path", "string")}, ReturnType: "boolean",
		Description: "Returns true if the file or directory at path exists, false otherwise.", Category: CatFile},
	{Name: "file_delete", Signature: "file_delete(path)", Params: []Param{p("path", "string")}, ReturnType: "boolean",
		Description: "Deletes the file at path. Returns true on success.", Category: CatFile},
	{Name: "file_move", Signature: "file_move(src, dst)", Params: []Param{p("src", "string"), p("dst", "string")}, ReturnType: "nil",
		Description: "Moves or renames a file from src to dst.", Category: CatFile},
	{Name: "file_copy", Signature: "file_copy(src, dst)", Params: []Param{p("src", "string"), p("dst", "string")}, ReturnType: "nil",
		Description: "Copies a file from src to dst.", Category: CatFile},
	{Name: "file_size", Signature: "file_size(path)", Params: []Param{p("path", "string")}, ReturnType: "number",
		Description: "Returns the size of the file in bytes.", Category: CatFile},
	{Name: "file_type", Signature: "file_type(path)", Params: []Param{p("path", "string")}, ReturnType: "string",
		Description: "Returns \"file\", \"dir\", or nil if the path does not exist.", Category: CatFile},
	{Name: "list_dir", Signature: "list_dir(path)", Params: []Param{p("path", "string")}, ReturnType: "list",
		Description: "Returns a list of filenames in the directory at path.", Category: CatFile},
	{Name: "make_dir", Signature: "make_dir(path)", Params: []Param{p("path", "string")}, ReturnType: "nil",
		Description: "Creates a new directory, including parent directories.", Category: CatFile},
	{Name: "remove_dir", Signature: "remove_dir(path)", Params: []Param{p("path", "string")}, ReturnType: "nil",
		Description: "Removes an empty directory. Fails if the directory is not empty.", Category: CatFile},
	{Name: "path_join", Signature: "path_join(base, part)", Params: []Param{p("base", "string"), p("part", "string")}, ReturnType: "string",
		Description: "Joins two path components with the OS-appropriate separator.", Category: CatFile},
	{Name: "path_base", Signature: "path_base(path)", Params: []Param{p("path", "string")}, ReturnType: "string",
		Description: "Returns the last component of a path (the filename or directory name).", Category: CatFile},
	{Name: "path_dir", Signature: "path_dir(path)", Params: []Param{p("path", "string")}, ReturnType: "string",
		Description: "Returns the directory portion of a path, without the final component.", Category: CatFile},
	{Name: "path_ext", Signature: "path_ext(path)", Params: []Param{p("path", "string")}, ReturnType: "string",
		Description: "Returns the file extension including the dot, or empty string if none.", Category: CatFile},
	{Name: "file_open", Signature: "file_open(path, mode)", Params: []Param{p("path", "string"), p("mode", "string")}, ReturnType: "number",
		Description: "Opens a file in random-access mode and returns a handle. mode: \"r\", \"w\", \"a\", \"rw\", \"rw+\".", Category: CatFile},
	{Name: "file_close", Signature: "file_close(handle)", Params: []Param{p("handle", "number")}, ReturnType: "nil",
		Description: "Closes a file opened with file_open, releasing its handle.", Category: CatFile},
	{Name: "file_read", Signature: "file_read(handle, offset, n)", Params: []Param{p("handle", "number"), p("offset", "number"), p("n", "number")}, ReturnType: "bytes",
		Description: "Reads n bytes from handle starting at offset and returns them as bytes. Fewer if past EOF.", Category: CatFile},
	{Name: "file_write", Signature: "file_write(handle, offset, data)", Params: []Param{p("handle", "number"), p("offset", "number"), p("data", "bytes|string")}, ReturnType: "number",
		Description: "Writes data to handle at offset. Returns the number of bytes written.", Category: CatFile},
	{Name: "file_truncate", Signature: "file_truncate(handle, size)", Params: []Param{p("handle", "number"), p("size", "number")}, ReturnType: "nil",
		Description: "Truncates the file to size bytes.", Category: CatFile},
	{Name: "file_sync", Signature: "file_sync(handle)", Params: []Param{p("handle", "number")}, ReturnType: "nil",
		Description: "Flushes file data to disk (fsync).", Category: CatFile},

	// ---- String ----
	{Name: "upper", Signature: "upper(str)", Params: []Param{p("str", "string")}, ReturnType: "string",
		Description: "Returns a copy of str with all characters converted to uppercase.", Category: CatString},
	{Name: "lower", Signature: "lower(str)", Params: []Param{p("str", "string")}, ReturnType: "string",
		Description: "Returns a copy of str with all characters converted to lowercase.", Category: CatString},
	{Name: "trim", Signature: "trim(str)", Params: []Param{p("str", "string")}, ReturnType: "string",
		Description: "Returns a copy of str with leading and trailing whitespace removed.", Category: CatString},
	{Name: "split", Signature: "split(str, delimiter)", Params: []Param{p("str", "string"), p("delimiter", "string")}, ReturnType: "list",
		Description: "Splits str into a list of substrings separated by delimiter.", Category: CatString},
	{Name: "join", Signature: "join(list, delimiter)", Params: []Param{p("list", "list"), p("delimiter", "string")}, ReturnType: "string",
		Description: "Joins list elements into a string with delimiter between each element.", Category: CatString},
	{Name: "contains", Signature: "contains(haystack, needle)", Params: []Param{p("haystack", "string|list"), p("needle", "any")}, ReturnType: "boolean",
		Description: "For strings: checks if needle is a substring. For lists: checks if needle is an element.", Category: CatString},
	{Name: "repeat", Signature: "repeat(str, count)", Params: []Param{p("str", "string"), p("count", "number")}, ReturnType: "string",
		Description: "Repeats str count times. Much faster than while-loop concat for large counts.", Category: CatString},
	{Name: "replace", Signature: "replace(str, old, new)", Params: []Param{p("str", "string"), p("old", "string"), p("new", "string")}, ReturnType: "string",
		Description: "Replaces the first occurrence of old with new in str.", Category: CatString},
	{Name: "replace_all", Signature: "replace_all(str, old, new)", Params: []Param{p("str", "string"), p("old", "string"), p("new", "string")}, ReturnType: "string",
		Description: "Replaces all occurrences of old with new in str.", Category: CatString},
	{Name: "index_of", Signature: "index_of(haystack, needle)", Params: []Param{p("haystack", "string|list"), p("needle", "any")}, ReturnType: "number",
		Description: "Returns the index of needle in haystack, or -1 if not found. String find or list element search.", Category: CatString},

	// ---- CSV ----
	{Name: "csv_parse", Signature: "csv_parse(text)", Params: []Param{p("text", "string")}, ReturnType: "list",
		Description: "Parses CSV text into a list of maps. First row is treated as headers.", Category: CatCSV},
	{Name: "csv_format", Signature: "csv_format(data, headers?)", Params: []Param{p("data", "list"), p("headers", "list")}, ReturnType: "string",
		Description: "Formats a list of maps into a CSV string. Optional headers list for column order.", Category: CatCSV},

	// ---- List ----
	{Name: "len", Signature: "len(value)", Params: []Param{p("value", "string|list|map|bytes")}, ReturnType: "number",
		Description: "Returns the length of a string (characters), list (elements), map (keys), or bytes (bytes).", Category: CatList},
	{Name: "push", Signature: "push(list, value...)", Params: []Param{p("list", "list"), p("value", "any")}, ReturnType: "list",
		Description: "Appends value(s) to the end of list. Mutates the list in place and returns it.", Category: CatList},
	{Name: "pop", Signature: "pop(list)", Params: []Param{p("list", "list")}, ReturnType: "any",
		Description: "Removes and returns the last element of list. Returns nil if empty.", Category: CatList},
	{Name: "at", Signature: "at(collection, index)", Params: []Param{p("collection", "list|string"), p("index", "number")}, ReturnType: "any",
		Description: "Returns the element at 0-based index in a list or string. Returns nil if out of bounds.", Category: CatList},
	{Name: "slice_list", Signature: "slice_list(list, start, end)", Params: []Param{p("list", "list"), p("start", "number"), p("end", "number")}, ReturnType: "list",
		Description: "Returns a sublist from start to end (exclusive). The x[a..b] syntax works on lists, strings and bytes.", Category: CatList},
	{Name: "slice", Signature: "slice(value, start, end)", Params: []Param{p("value", "list|string|bytes"), p("start", "number"), p("end", "number")}, ReturnType: "any",
		Description: "Returns a slice of value from start to end (exclusive). Indexes are clamped.", Category: CatList},
	{Name: "sort", Signature: "sort(list, comparator?)", Params: []Param{p("list", "list"), p("comparator", "function")}, ReturnType: "list",
		Description: "Sorts list. Numbers numerically, strings lexicographically, or via comparator(a, b) that returns truthy if a sorts before b.", Category: CatList},
	{Name: "sorted_by", Signature: "sorted_by(list, keyFn)", Params: []Param{p("list", "list"), p("keyFn", "function")}, ReturnType: "list",
		Description: "Returns a new list sorted by the key that keyFn(element) returns for each element.", Category: CatList},
	{Name: "range", Signature: "range(start?, end?, step?)", Params: []Param{p("start", "number"), p("end", "number"), p("step", "number")}, ReturnType: "list",
		Description: "Creates a list of numbers. range(n) gives 0..n, range(a, b) gives a..b, range(a, b, s) with step s.", Category: CatList},
	{Name: "map", Signature: "map(list, fn)", Params: []Param{p("list", "list"), p("fn", "function")}, ReturnType: "list",
		Description: "Applies fn to each element and returns a new list of results.", Category: CatList},
	{Name: "filter", Signature: "filter(list, fn)", Params: []Param{p("list", "list"), p("fn", "function")}, ReturnType: "list",
		Description: "Returns a new list containing only elements where fn(element) returns truthy.", Category: CatList},
	{Name: "reduce", Signature: "reduce(list, fn, initial)", Params: []Param{p("list", "list"), p("fn", "function"), p("initial", "any")}, ReturnType: "any",
		Description: "Accumulates a value by calling fn(accumulator, element) for each element.", Category: CatList},
	{Name: "each", Signature: "each(list, fn)", Params: []Param{p("list", "list"), p("fn", "function")}, ReturnType: "nil",
		Description: "Calls fn(element) for each element in list. Used for side effects.", Category: CatList},
	{Name: "unique", Signature: "unique(list)", Params: []Param{p("list", "list")}, ReturnType: "list",
		Description: "Returns a new list with duplicate elements removed. Preserves order of first occurrence.", Category: CatList},

	// ---- Map ----
	{Name: "get", Signature: "get(map, key)", Params: []Param{p("map", "map"), p("key", "string")}, ReturnType: "any",
		Description: "Returns the value associated with key in map, or nil if not found.", Category: CatMap},
	{Name: "set", Signature: "set(map, key, value)", Params: []Param{p("map", "map"), p("key", "string"), p("value", "any")}, ReturnType: "map",
		Description: "Sets key to value in map. Creates or updates. Mutates the map in place.", Category: CatMap},
	{Name: "keys", Signature: "keys(map)", Params: []Param{p("map", "map")}, ReturnType: "list",
		Description: "Returns a list of all keys in map. Order is not guaranteed.", Category: CatMap},
	{Name: "values", Signature: "values(map)", Params: []Param{p("map", "map")}, ReturnType: "list",
		Description: "Returns a list of all values in map.", Category: CatMap},

	// ---- Math ----
	{Name: "abs", Signature: "abs(x)", Params: []Param{p("x", "number")}, ReturnType: "number",
		Description: "Returns the absolute value of x.", Category: CatMath},
	{Name: "min", Signature: "min(a, b...)", Params: []Param{p("a", "number"), p("b", "number")}, ReturnType: "number",
		Description: "Returns the smallest of the given numbers.", Category: CatMath},
	{Name: "max", Signature: "max(a, b...)", Params: []Param{p("a", "number"), p("b", "number")}, ReturnType: "number",
		Description: "Returns the largest of the given numbers.", Category: CatMath},
	{Name: "pow", Signature: "pow(base, exp)", Params: []Param{p("base", "number"), p("exp", "number")}, ReturnType: "number",
		Description: "Returns base raised to the power of exp.", Category: CatMath},
	{Name: "sqrt", Signature: "sqrt(x)", Params: []Param{p("x", "number")}, ReturnType: "number",
		Description: "Returns the square root of x.", Category: CatMath},
	{Name: "round", Signature: "round(x)", Params: []Param{p("x", "number")}, ReturnType: "number",
		Description: "Rounds x to the nearest integer. Half values round to the nearest even integer.", Category: CatMath},
	{Name: "ceil", Signature: "ceil(x)", Params: []Param{p("x", "number")}, ReturnType: "number",
		Description: "Rounds x up to the nearest integer.", Category: CatMath},
	{Name: "floor", Signature: "floor(x)", Params: []Param{p("x", "number")}, ReturnType: "number",
		Description: "Rounds x down to the nearest integer.", Category: CatMath},

	// ---- Network & HTTP ----
	{Name: "http_get", Signature: "http_get(url)", Params: []Param{p("url", "string")}, ReturnType: "string",
		Description: "Performs an HTTP GET request to url and returns the response body as a string.", Category: CatNet},
	{Name: "http_post", Signature: "http_post(url, body)", Params: []Param{p("url", "string"), p("body", "string")}, ReturnType: "string",
		Description: "Performs an HTTP POST request to url with body as the request payload.", Category: CatNet},
	{Name: "http_get_json", Signature: "http_get_json(url)", Params: []Param{p("url", "string")}, ReturnType: "any",
		Description: "Performs an HTTP GET request to url and parses the response as JSON.", Category: CatNet},
	{Name: "http_request", Signature: "http_request(method, url, headers?, body?)", Params: []Param{p("method", "string"), p("url", "string"), p("headers", "map"), p("body", "string")}, ReturnType: "map",
		Description: "Performs an HTTP request with custom method, URL, optional headers map and body. Returns map with status, headers, body keys.", Category: CatNet},
	{Name: "http_stream_open", Signature: "http_stream_open(url, headers?)", Params: []Param{p("url", "string"), p("headers", "map")}, ReturnType: "number",
		Description: "Opens a long-lived HTTP GET stream (SSE/streaming) and returns a handle. Use http_stream_read / http_stream_read_line to consume data.", Category: CatNet},
	{Name: "http_stream_read", Signature: "http_stream_read(handle)", Params: []Param{p("handle", "number")}, ReturnType: "string",
		Description: "Reads the next chunk (up to 4096 bytes) from an open HTTP stream. Returns empty string on EOF.", Category: CatNet},
	{Name: "http_stream_read_line", Signature: "http_stream_read_line(handle)", Params: []Param{p("handle", "number")}, ReturnType: "string",
		Description: "Reads the next line from an open HTTP stream. Returns empty string on EOF.", Category: CatNet},
	{Name: "http_stream_close", Signature: "http_stream_close(handle)", Params: []Param{p("handle", "number")}, ReturnType: "nil",
		Description: "Closes an HTTP stream handle and releases the connection.", Category: CatNet},
	{Name: "parse_json", Signature: "parse_json(json_string)", Params: []Param{p("json_string", "string")}, ReturnType: "any",
		Description: "Parses a JSON string into Pipe data structures.", Category: CatNet},
	{Name: "to_json", Signature: "to_json(value)", Params: []Param{p("value", "any")}, ReturnType: "string",
		Description: "Serializes a Pipe value into a JSON string.", Category: CatNet},

	// ---- TCP ----
	{Name: "tcp_listen", Signature: "tcp_listen(address)", Params: []Param{p("address", "string")}, ReturnType: "listener",
		Description: "Starts a TCP server listening on address and returns a listener handle.", Category: CatTCP},
	{Name: "tcp_connect", Signature: "tcp_connect(address)", Params: []Param{p("address", "string")}, ReturnType: "connection",
		Description: "Connects to a TCP server at address and returns a connection handle.", Category: CatTCP},
	{Name: "tcp_accept", Signature: "tcp_accept(listener)", Params: []Param{p("listener", "listener")}, ReturnType: "connection",
		Description: "Accepts an incoming connection on a listener. Blocks until a client connects.", Category: CatTCP},
	{Name: "tcp_read", Signature: "tcp_read(conn, max_bytes)", Params: []Param{p("conn", "connection"), p("max_bytes", "number")}, ReturnType: "string",
		Description: "Reads up to max_bytes from a TCP connection and returns the data as a string.", Category: CatTCP},
	{Name: "tcp_write", Signature: "tcp_write(conn, data)", Params: []Param{p("conn", "connection"), p("data", "string")}, ReturnType: "nil",
		Description: "Writes data to a TCP connection.", Category: CatTCP},
	{Name: "tcp_close", Signature: "tcp_close(handle)", Params: []Param{p("handle", "connection|listener")}, ReturnType: "nil",
		Description: "Closes a TCP connection or listener.", Category: CatTCP},

	// ---- HTTP Server ----
	{Name: "http_server", Signature: "http_server(addr, handler)", Params: []Param{p("addr", "string"), p("handler", "function")}, ReturnType: "server",
		Description: "Starts an HTTP server on addr (e.g. \"0.0.0.0:8080\"). handler is a function fn(req) that receives a request map {method, path, query, headers, body} and returns a response map {status, headers, body}. Returns a server handle.", Category: CatServer},
	{Name: "http_close", Signature: "http_close(server)", Params: []Param{p("server", "server")}, ReturnType: "nil",
		Description: "Shuts down an HTTP server and releases the port.", Category: CatServer},

	// ---- Regex ----
	{Name: "regex_match", Signature: "regex_match(pattern, str)", Params: []Param{p("pattern", "string"), p("str", "string")}, ReturnType: "boolean",
		Description: "Returns true if str matches the regex pattern, false otherwise.", Category: CatRegex},
	{Name: "regex_replace", Signature: "regex_replace(pattern, replacement, str)", Params: []Param{p("pattern", "string"), p("replacement", "string"), p("str", "string")}, ReturnType: "string",
		Description: "Replaces all occurrences of pattern in str with replacement.", Category: CatRegex},

	// ---- Date & Time ----
	{Name: "now", Signature: "now()", Params: nil, ReturnType: "number",
		Description: "Returns the current time as a Unix timestamp in seconds (floating point).", Category: CatDate},
	{Name: "format_time", Signature: "format_time(timestamp, layout)", Params: []Param{p("timestamp", "number"), p("layout", "string")}, ReturnType: "string",
		Description: "Formats a Unix timestamp using Go's reference-time layout (e.g. \"2006-01-02 15:04:05\").", Category: CatDate},
	{Name: "parse_date", Signature: "parse_date(date_string, layout?)", Params: []Param{p("date_string", "string"), p("layout", "string")}, ReturnType: "number",
		Description: "Parses a date string into a Unix timestamp. Default layout: \"2006-01-02\".", Category: CatDate},

	// ---- Random ----
	{Name: "random", Signature: "random()", Params: nil, ReturnType: "number",
		Description: "Returns a random floating-point number in the range [0.0, 1.0).", Category: CatRandom},
	{Name: "random_range", Signature: "random_range(min, max)", Params: []Param{p("min", "number"), p("max", "number")}, ReturnType: "number",
		Description: "Returns a random integer in the range [min, max] inclusive.", Category: CatRandom},

	// ---- Cryptography ----
	{Name: "secure_random", Signature: "secure_random(byte_count)", Params: []Param{p("byte_count", "number")}, ReturnType: "string",
		Description: "Returns a hex-encoded string of cryptographically secure random bytes (1-1024).", Category: CatCrypto},
	{Name: "secure_random_int", Signature: "secure_random_int()", Params: nil, ReturnType: "number",
		Description: "Returns a cryptographically secure random 64-bit integer.", Category: CatCrypto},
	{Name: "secure_random_range", Signature: "secure_random_range(min, max)", Params: []Param{p("min", "number"), p("max", "number")}, ReturnType: "number",
		Description: "Returns a cryptographically secure random integer in [min, max).", Category: CatCrypto},
	{Name: "secure_random_bytes", Signature: "secure_random_bytes(byte_count)", Params: []Param{p("byte_count", "number")}, ReturnType: "bytes",
		Description: "Returns cryptographically secure random bytes (max 1024). Use for keys, IVs, nonces.", Category: CatCrypto},
	{Name: "encrypt", Signature: "encrypt(key, plaintext[, associated_data])", Params: []Param{p("key", "string"), p("plaintext", "string|bytes"), p("associated_data", "string|bytes")}, ReturnType: "string",
		Description: "Encrypts data using AES-GCM. Key must be 16/24/32 bytes. Returns base64 ciphertext with embedded nonce.", Category: CatCrypto},
	{Name: "decrypt", Signature: "decrypt(key, ciphertext[, associated_data])", Params: []Param{p("key", "string"), p("ciphertext", "string"), p("associated_data", "string|bytes")}, ReturnType: "string",
		Description: "Decrypts AES-GCM ciphertext. Returns plaintext or error if authentication fails.", Category: CatCrypto},
	{Name: "hmac_sha1", Signature: "hmac_sha1(key, message)", Params: []Param{p("key", "string"), p("message", "string")}, ReturnType: "string",
		Description: "Computes HMAC-SHA1(key, message). Returns hex-encoded 20-byte MAC.", Category: CatCrypto},
	{Name: "hmac_sha256", Signature: "hmac_sha256(key, message)", Params: []Param{p("key", "string"), p("message", "string")}, ReturnType: "string",
		Description: "Computes HMAC-SHA256(key, message). Returns hex-encoded 32-byte MAC.", Category: CatCrypto},
	{Name: "hmac_sha512", Signature: "hmac_sha512(key, message)", Params: []Param{p("key", "string"), p("message", "string")}, ReturnType: "string",
		Description: "Computes HMAC-SHA512(key, message). Returns hex-encoded 64-byte MAC.", Category: CatCrypto},

	// ---- Encoding ----
	{Name: "base64_encode", Signature: "base64_encode(str)", Params: []Param{p("str", "string")}, ReturnType: "string",
		Description: "Encodes a string to Base64.", Category: CatEncode},
	{Name: "base64_decode", Signature: "base64_decode(str)", Params: []Param{p("str", "string")}, ReturnType: "string",
		Description: "Decodes a Base64-encoded string.", Category: CatEncode},
	{Name: "base64url_encode", Signature: "base64url_encode(str)", Params: []Param{p("str", "string")}, ReturnType: "string",
		Description: "Encodes a string to URL-safe Base64 without padding (RFC 4648 §5). Used for PKCE code challenges, JWTs.", Category: CatEncode},
	{Name: "base64url_decode", Signature: "base64url_decode(str)", Params: []Param{p("str", "string")}, ReturnType: "string",
		Description: "Decodes a URL-safe Base64 string (padding optional).", Category: CatEncode},
	{Name: "url_encode", Signature: "url_encode(str)", Params: []Param{p("str", "string")}, ReturnType: "string",
		Description: "Percent-encodes a string for use in URLs and query strings (RFC 3986).", Category: CatEncode},
	{Name: "url_decode", Signature: "url_decode(str)", Params: []Param{p("str", "string")}, ReturnType: "string",
		Description: "Decodes a percent-encoded URL string.", Category: CatEncode},

	// ---- Hashing ----
	{Name: "sha256", Signature: "sha256(text)", Params: []Param{p("text", "string")}, ReturnType: "string",
		Description: "Computes the SHA-256 hash of text and returns it as a hex string.", Category: CatHash},
	{Name: "md5", Signature: "md5(text)", Params: []Param{p("text", "string")}, ReturnType: "string",
		Description: "Computes the MD5 hash of text and returns it as a hex string.", Category: CatHash},
	{Name: "sha1", Signature: "sha1(text)", Params: []Param{p("text", "string")}, ReturnType: "string",
		Description: "Computes the SHA-1 hash of text and returns it as a hex string.", Category: CatHash},
	{Name: "sha512", Signature: "sha512(text)", Params: []Param{p("text", "string")}, ReturnType: "string",
		Description: "Computes the SHA-512 hash of text and returns it as a hex string.", Category: CatHash},

	// ---- Bytes & Binary ----
	{Name: "to_bytes", Signature: "to_bytes(value)", Params: []Param{p("value", "string|bytes|list")}, ReturnType: "bytes",
		Description: "Converts a string to its UTF-8 bytes, or a list of integers 0-255 to bytes. Returns bytes unchanged.", Category: CatBytes},
	{Name: "from_bytes", Signature: "from_bytes(value)", Params: []Param{p("value", "bytes|string")}, ReturnType: "string",
		Description: "Converts bytes to a string, decoding them as UTF-8. Returns strings unchanged.", Category: CatBytes},
	{Name: "bytes_append", Signature: "bytes_append(bytes, chunk, ...)", Params: []Param{p("bytes", "bytes|string"), p("chunk", "bytes|string")}, ReturnType: "bytes",
		Description: "Appends one or more byte chunks to bytes and returns the result.", Category: CatBytes},
	{Name: "bytes_to_int", Signature: "bytes_to_int(bytes, offset?, n?)", Params: []Param{p("bytes", "bytes"), p("offset", "number"), p("n", "number")}, ReturnType: "number",
		Description: "Interprets n big-endian bytes (max 8) starting at offset as an unsigned integer. Defaults to the whole bytes.", Category: CatBytes},
	{Name: "int_to_bytes", Signature: "int_to_bytes(value, n?)", Params: []Param{p("value", "number"), p("n", "number")}, ReturnType: "bytes",
		Description: "Encodes a non-negative integer as big-endian bytes (minimal, or exactly n bytes if given).", Category: CatBytes},
	{Name: "bytes_compare", Signature: "bytes_compare(a, b)", Params: []Param{p("a", "bytes|string"), p("b", "bytes|string")}, ReturnType: "number",
		Description: "Lexicographically compares two byte sequences: negative if a < b, 0 if equal, positive if a > b.", Category: CatBytes},
	{Name: "hex_encode", Signature: "hex_encode(bytes)", Params: []Param{p("bytes", "bytes")}, ReturnType: "string",
		Description: "Encodes bytes as a lowercase hexadecimal string.", Category: CatBytes},
	{Name: "hex_decode", Signature: "hex_decode(str)", Params: []Param{p("str", "string")}, ReturnType: "bytes",
		Description: "Decodes a hexadecimal string into bytes.", Category: CatBytes},
	{Name: "bit_and", Signature: "bit_and(a, b)", Params: []Param{p("a", "number"), p("b", "number")}, ReturnType: "number",
		Description: "Bitwise AND of two integers.", Category: CatBytes},
	{Name: "bit_or", Signature: "bit_or(a, b)", Params: []Param{p("a", "number"), p("b", "number")}, ReturnType: "number",
		Description: "Bitwise OR of two integers.", Category: CatBytes},
	{Name: "bit_xor", Signature: "bit_xor(a, b)", Params: []Param{p("a", "number"), p("b", "number")}, ReturnType: "number",
		Description: "Bitwise XOR of two integers.", Category: CatBytes},
	{Name: "bit_not", Signature: "bit_not(a)", Params: []Param{p("a", "number")}, ReturnType: "number",
		Description: "Bitwise NOT of an integer.", Category: CatBytes},
	{Name: "bit_lshift", Signature: "bit_lshift(a, n)", Params: []Param{p("a", "number"), p("n", "number")}, ReturnType: "number",
		Description: "Bitwise left-shift of a by n positions.", Category: CatBytes},
	{Name: "bit_rshift", Signature: "bit_rshift(a, n)", Params: []Param{p("a", "number"), p("n", "number")}, ReturnType: "number",
		Description: "Bitwise right-shift of a by n positions.", Category: CatBytes},
	{Name: "crc32", Signature: "crc32(value)", Params: []Param{p("value", "string|bytes")}, ReturnType: "number",
		Description: "Computes the IEEE CRC-32 checksum of value.", Category: CatBytes},
	{Name: "substring", Signature: "substring(str, start, end)", Params: []Param{p("str", "string"), p("start", "number"), p("end", "number")}, ReturnType: "string",
		Description: "Returns the substring of str from start (inclusive) to end (exclusive), clamped to the string bounds.", Category: CatString},

	// ---- Type Checks ----
	{Name: "type_of", Signature: "type_of(value)", Params: []Param{p("value", "any")}, ReturnType: "string",
		Description: "Returns a string indicating the type of value (\"number\", \"string\", \"list\", \"map\", ...).", Category: CatType},
	{Name: "is_num", Signature: "is_num(value)", Params: []Param{p("value", "any")}, ReturnType: "boolean",
		Description: "Returns true if value is a number.", Category: CatType},
	{Name: "is_str", Signature: "is_str(value)", Params: []Param{p("value", "any")}, ReturnType: "boolean",
		Description: "Returns true if value is a string.", Category: CatType},
	{Name: "is_list", Signature: "is_list(value)", Params: []Param{p("value", "any")}, ReturnType: "boolean",
		Description: "Returns true if value is a list.", Category: CatType},
	{Name: "is_map", Signature: "is_map(value)", Params: []Param{p("value", "any")}, ReturnType: "boolean",
		Description: "Returns true if value is a map.", Category: CatType},
	{Name: "is_nil", Signature: "is_nil(value)", Params: []Param{p("value", "any")}, ReturnType: "boolean",
		Description: "Returns true if value is nil.", Category: CatType},

	// ---- Conversion ----
	{Name: "to_str", Signature: "to_str(value)", Params: []Param{p("value", "any")}, ReturnType: "string",
		Description: "Converts value to its string representation.", Category: CatConv},
	{Name: "to_num", Signature: "to_num(str)", Params: []Param{p("str", "string")}, ReturnType: "number",
		Description: "Parses a string into a number. Returns nil if parsing fails.", Category: CatConv},

	// ---- Result ----
	{Name: "Ok", Signature: "Ok(value)", Params: []Param{p("value", "any")}, ReturnType: "Result",
		Description: "Creates a successful Result containing value.", Category: CatResult},
	{Name: "Err", Signature: "Err(message)", Params: []Param{p("message", "string")}, ReturnType: "Result",
		Description: "Creates a failed Result containing the error message.", Category: CatResult},
	{Name: "is_ok", Signature: "is_ok(result)", Params: []Param{p("result", "Result")}, ReturnType: "boolean",
		Description: "Returns true if result is an Ok variant.", Category: CatResult},
	{Name: "is_err", Signature: "is_err(result)", Params: []Param{p("result", "Result")}, ReturnType: "boolean",
		Description: "Returns true if result is an Err variant.", Category: CatResult},
	{Name: "unwrap", Signature: "unwrap(result)", Params: []Param{p("result", "Result")}, ReturnType: "any",
		Description: "Returns the value inside Ok, or raises an error if called on Err.", Category: CatResult},
	{Name: "unwrap_or", Signature: "unwrap_or(result, default)", Params: []Param{p("result", "Result"), p("default", "any")}, ReturnType: "any",
		Description: "Returns the value inside Ok, or default if result is Err.", Category: CatResult},
	{Name: "raise", Signature: "raise(message)", Params: []Param{p("message", "string")}, ReturnType: "error",
		Description: "Raises an error with the given message. Catchable with try/catch.", Category: CatResult},

	// ---- AI Configuration ----
	{Name: "ai_provider", Signature: "ai_provider(name, {model, host, timeout}?)", Params: []Param{p("name", "string"), p("config", "map, optional")}, ReturnType: "string",
		Description: "Sets the AI provider: \"openai\", \"anthropic\", \"deepseek\", or \"ollama\". Optionally overrides defaults with a config block, e.g. ai_provider \"deepseek\" {model: \"deepseek-v4-flash\"}.", Category: CatAIConf},
	{Name: "ai_model", Signature: "ai_model(name)", Params: []Param{p("name", "string")}, ReturnType: "string",
		Description: "Sets the model name for the current AI provider.", Category: CatAIConf},
	{Name: "ai_host", Signature: "ai_host(url)", Params: []Param{p("url", "string")}, ReturnType: "string",
		Description: "Sets a custom API host URL, e.g. for local proxies or Ollama.", Category: CatAIConf},
	{Name: "ai_timeout", Signature: "ai_timeout(seconds)", Params: []Param{p("seconds", "number")}, ReturnType: "nil",
		Description: "Sets the AI request timeout in seconds.", Category: CatAIConf},
	{Name: "ai_cache", Signature: "ai_cache(on_off|ttl|'clear'|'stats')", Params: []Param{p("arg", "bool|number|string")}, ReturnType: "string",
		Description: "Enables/disables AI response caching. Pass true/on/ttl-minutes to enable, false/off to disable, 'clear' to flush, 'stats' for hit/miss count.", Category: CatAIConf},
	{Name: "ai_set_key", Signature: "ai_set_key(provider, key)", Params: []Param{p("provider", "string"), p("key", "string")}, ReturnType: "string",
		Description: "Sets API key for the given provider ('openai', 'deepseek', 'anthropic'). Useful when env vars aren't available (browser, CI).", Category: CatAIConf},

	// ---- AI Chat ----
	{Name: "ai_chat", Signature: "ai_chat(system_prompt, user_prompt, max_tokens?)", Params: []Param{p("system_prompt", "string"), p("user_prompt", "string"), p("max_tokens", "int, optional")}, ReturnType: "string",
		Description: "Sends a chat request with system and user prompts. Optionally limits the response length via max_tokens (e.g. 300 for short answers). Returns the assistant's response.", Category: CatAIChat},
	{Name: "ai_chat_json", Signature: "ai_chat_json(system_prompt, user_prompt, max_tokens?)", Params: []Param{p("system_prompt", "string"), p("user_prompt", "string"), p("max_tokens", "int, optional")}, ReturnType: "any",
		Description: "Like ai_chat, but parses the response as JSON and returns the parsed value. Optionally limits the response length via max_tokens.", Category: CatAIChat},

	// ---- AI High-level ----
	{Name: "summarize", Signature: "summarize(text)", Params: []Param{p("text", "string")}, ReturnType: "string",
		Description: "Summarizes the given text in 2-3 sentences.", Category: CatAIHi},
	{Name: "translate", Signature: "translate(text, target_language)", Params: []Param{p("text", "string"), p("target_language", "string")}, ReturnType: "string",
		Description: "Translates text into the target language.", Category: CatAIHi},
	{Name: "classify", Signature: "classify(text, categories)", Params: []Param{p("text", "string"), p("categories", "string|list")}, ReturnType: "string",
		Description: "Classifies text into exactly one of the given categories.", Category: CatAIHi},
	{Name: "extract", Signature: "extract(text, schema)", Params: []Param{p("text", "string"), p("schema", "string")}, ReturnType: "any",
		Description: "Extracts structured data from text according to a schema description. Returns parsed JSON.", Category: CatAIHi},
	{Name: "generate", Signature: "generate(prompt)", Params: []Param{p("prompt", "string")}, ReturnType: "string",
		Description: "Generates text from a single prompt (no system message).", Category: CatAIHi},
	{Name: "generate_json", Signature: "generate_json(instruction, schema)", Params: []Param{p("instruction", "string"), p("schema", "string")}, ReturnType: "any",
		Description: "Generates structured JSON data matching a schema description. Returns parsed JSON as native Pipe types.", Category: CatAIHi},
	{Name: "ask", Signature: "ask(question)", Params: []Param{p("question", "string")}, ReturnType: "string",
		Description: "Answers a single question conversationally.", Category: CatAIHi},

	// ---- AI Streaming ----
	{Name: "ai_stream", Signature: "ai_stream(system_prompt, user_prompt)", Params: []Param{p("system_prompt", "string"), p("user_prompt", "string")}, ReturnType: "string",
		Description: "Streams the response token-by-token to stdout. Returns the full accumulated response.", Category: CatAIStream},

	// ---- AI Parallel ----
	{Name: "ai_batch", Signature: "ai_batch(system_prompt, items)", Params: []Param{p("system_prompt", "string"), p("items", "list")}, ReturnType: "list",
		Description: "Processes a list of items in parallel with the same system prompt.", Category: CatAIPar},
	{Name: "ai_parallel", Signature: "ai_parallel(concurrency, system_prompt, items)", Params: []Param{p("concurrency", "number"), p("system_prompt", "string"), p("items", "list")}, ReturnType: "list",
		Description: "Like ai_batch but with an explicit concurrency limit (max parallel requests).", Category: CatAIPar},
	{Name: "ai_rate_limit", Signature: "ai_rate_limit(calls_per_second)", Params: []Param{p("calls_per_second", "number")}, ReturnType: "nil",
		Description: "Limits the rate of AI calls for subsequent parallel/batch operations.", Category: CatAIPar},

	// ---- AI Tool Calling ----
	{Name: "ai_tool", Signature: "ai_tool(name, description, parameters, function)", Params: []Param{p("name", "string"), p("description", "string"), p("parameters", "map"), p("function", "function")}, ReturnType: "nil",
		Description: "Registers a tool the AI can call. parameters is a JSON schema map.", Category: CatAITool},
	{Name: "ai_with_tools", Signature: "ai_with_tools(system_prompt, user_prompt, max_rounds?)", Params: []Param{p("system_prompt", "string"), p("user_prompt", "string"), p("max_rounds", "number")}, ReturnType: "string",
		Description: "Sends a chat request with tool-calling enabled. max_rounds defaults to 5.", Category: CatAITool},

	// ---- AI Agents ----
	{Name: "agent", Signature: "agent(name, system_prompt)", Params: []Param{p("name", "string"), p("system_prompt", "string")}, ReturnType: "string",
		Description: "Creates a stateful agent with the given name and system prompt. Agents maintain conversation history across calls.", Category: CatAIAgent},
	{Name: "agent_ask", Signature: "agent_ask(name, message)", Params: []Param{p("name", "string"), p("message", "string")}, ReturnType: "string",
		Description: "Sends a message to the named agent. The agent remembers all previous messages (conversation history).", Category: CatAIAgent},
	{Name: "agent_clear", Signature: "agent_clear(name)", Params: []Param{p("name", "string")}, ReturnType: "string",
		Description: "Clears the conversation history of the named agent, keeping the system prompt.", Category: CatAIAgent},
	{Name: "try_ai_log", Signature: "try_ai_log()", Params: nil, ReturnType: "list",
		Description: "Returns the log of all try_ai fix attempts as a list of maps (time, code, original, fixed, attempt, success).", Category: CatAIAgent},

	// ---- AI Embeddings ----
	{Name: "embed", Signature: "embed(text)", Params: []Param{p("text", "string")}, ReturnType: "list",
		Description: "Converts text into an embedding vector (list of floats).", Category: CatAIEmbed},
	{Name: "embed_batch", Signature: "embed_batch(items)", Params: []Param{p("items", "list")}, ReturnType: "list",
		Description: "Embeds multiple texts in parallel.", Category: CatAIEmbed},
	{Name: "cosine_sim", Signature: "cosine_sim(vec_a, vec_b)", Params: []Param{p("vec_a", "list"), p("vec_b", "list")}, ReturnType: "number",
		Description: "Computes the cosine similarity between two embedding vectors.", Category: CatAIEmbed},
	{Name: "dot_product", Signature: "dot_product(vec_a, vec_b)", Params: []Param{p("vec_a", "list"), p("vec_b", "list")}, ReturnType: "number",
		Description: "Computes the dot product of two vectors.", Category: CatAIEmbed},
	{Name: "nearest", Signature: "nearest(query_vec, doc_vectors, k)", Params: []Param{p("query_vec", "list"), p("doc_vectors", "list"), p("k", "number")}, ReturnType: "list",
		Description: "Finds the k nearest neighbors to query_vec among doc_vectors using cosine similarity.", Category: CatAIEmbed},

	// ---- AI Search ----
	{Name: "web_search", Signature: "web_search(query)", Params: []Param{p("query", "string")}, ReturnType: "list",
		Description: "Searches the web via DuckDuckGo Instant Answer API (free, no key needed). Returns a list of maps with title, snippet, url.", Category: CatAISearch},
	{Name: "wiki_search", Signature: "wiki_search(query)", Params: []Param{p("query", "string")}, ReturnType: "list",
		Description: "Searches Wikipedia (supports CORS for browser/WASM). Returns a list of maps with title, snippet, url.", Category: CatAISearch},

	// ---- Sandbox ----
	{Name: "sandbox_profile", Signature: "sandbox_profile(name)", Params: []Param{p("name", "string")}, ReturnType: "string",
		Description: "Selects a sandbox profile (none, strict, noexec, isolated, networked).", Category: CatSandbox},
	{Name: "set_sandbox", Signature: "set_sandbox(profile)", Params: []Param{p("profile", "map|string")}, ReturnType: "string",
		Description: "Sets the active sandbox from a profile map or name.", Category: CatSandbox},
	{Name: "with_sandbox", Signature: "with_sandbox(profile, fn)", Params: []Param{p("profile", "map|string"), p("fn", "function")}, ReturnType: "any",
		Description: "Runs fn under the given sandbox profile, then restores the previous one.", Category: CatSandbox},

	// ---- Test Assertions ----
	{Name: "assert", Signature: "assert(condition)", Params: []Param{p("condition", "boolean")}, ReturnType: "nil",
		Description: "Asserts that a value is truthy.", Category: CatTest},
	{Name: "assert_eq", Signature: "assert_eq(expected, actual)", Params: []Param{p("expected", "any"), p("actual", "any")}, ReturnType: "nil",
		Description: "Asserts that two values are equal.", Category: CatTest},
	{Name: "assert_not_eq", Signature: "assert_not_eq(unexpected, actual)", Params: []Param{p("unexpected", "any"), p("actual", "any")}, ReturnType: "nil",
		Description: "Asserts that two values are not equal.", Category: CatTest},
	{Name: "assert_lt", Signature: "assert_lt(a, b)", Params: []Param{p("a", "number"), p("b", "number")}, ReturnType: "nil",
		Description: "Asserts that a is numerically less than b.", Category: CatTest},
	{Name: "assert_gt", Signature: "assert_gt(a, b)", Params: []Param{p("a", "number"), p("b", "number")}, ReturnType: "nil",
		Description: "Asserts that a is numerically greater than b.", Category: CatTest},
	{Name: "assert_error", Signature: "assert_error(fn)", Params: []Param{p("fn", "function")}, ReturnType: "nil",
		Description: "Asserts that calling fn() returns an error. Pass a zero-argument function.", Category: CatTest},

	// ---- Concurrency (eval-only) ----
	{Name: "go", Signature: "go(fn, args...)", Params: []Param{p("fn", "function"), p("args", "any")}, ReturnType: "nil",
		Description: "Runs fn asynchronously in a goroutine with the given arguments.", Category: CatIO},
}

var builtinIndex = func() map[string]BuiltinDoc {
	m := make(map[string]BuiltinDoc, len(builtinDocs))
	for _, d := range builtinDocs {
		m[d.Name] = d
	}
	return m
}()

// Builtin returns documentation for a builtin by name.
func Builtin(name string) (BuiltinDoc, bool) {
	d, ok := builtinIndex[name]
	return d, ok
}

// AllBuiltins returns all builtin docs, sorted by name.
func AllBuiltins() []BuiltinDoc {
	out := make([]BuiltinDoc, len(builtinDocs))
	copy(out, builtinDocs)
	sortBuiltinsByName(out)
	return out
}

func sortBuiltinsByName(docs []BuiltinDoc) {
	for i := 1; i < len(docs); i++ {
		for j := i; j > 0 && docs[j].Name < docs[j-1].Name; j-- {
			docs[j], docs[j-1] = docs[j-1], docs[j]
		}
	}
}

// IsBuiltin reports whether name is a known builtin.
func IsBuiltin(name string) bool {
	_, ok := builtinIndex[name]
	return ok
}
