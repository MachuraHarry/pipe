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
	CatList     = "List"
	CatMap      = "Map"
	CatMath     = "Math"
	CatNet      = "Network & HTTP"
	CatTCP      = "TCP"
	CatRegex    = "Regex"
	CatDate     = "Date & Time"
	CatRandom   = "Random"
	CatEncode   = "Encoding"
	CatHash     = "Hashing"
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

	// ---- List ----
	{Name: "len", Signature: "len(value)", Params: []Param{p("value", "string|list|map")}, ReturnType: "number",
		Description: "Returns the length of a string (characters), list (elements), or map (keys).", Category: CatList},
	{Name: "push", Signature: "push(list, value...)", Params: []Param{p("list", "list"), p("value", "any")}, ReturnType: "list",
		Description: "Appends value(s) to the end of list. Mutates the list in place and returns it.", Category: CatList},
	{Name: "pop", Signature: "pop(list)", Params: []Param{p("list", "list")}, ReturnType: "any",
		Description: "Removes and returns the last element of list. Returns nil if empty.", Category: CatList},
	{Name: "at", Signature: "at(collection, index)", Params: []Param{p("collection", "list|string"), p("index", "number")}, ReturnType: "any",
		Description: "Returns the element at 0-based index in a list or string. Returns nil if out of bounds.", Category: CatList},
	{Name: "slice_list", Signature: "slice_list(list, start, end)", Params: []Param{p("list", "list"), p("start", "number"), p("end", "number")}, ReturnType: "list",
		Description: "Returns a sublist from start to end (exclusive).", Category: CatList},
	{Name: "sort", Signature: "sort(list)", Params: []Param{p("list", "list")}, ReturnType: "list",
		Description: "Sorts list in ascending order. Numbers numerically, strings lexicographically.", Category: CatList},
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

	// ---- Network & HTTP ----
	{Name: "http_get", Signature: "http_get(url)", Params: []Param{p("url", "string")}, ReturnType: "string",
		Description: "Performs an HTTP GET request to url and returns the response body as a string.", Category: CatNet},
	{Name: "http_post", Signature: "http_post(url, body)", Params: []Param{p("url", "string"), p("body", "string")}, ReturnType: "string",
		Description: "Performs an HTTP POST request to url with body as the request payload.", Category: CatNet},
	{Name: "http_get_json", Signature: "http_get_json(url)", Params: []Param{p("url", "string")}, ReturnType: "any",
		Description: "Performs an HTTP GET request to url and parses the response as JSON.", Category: CatNet},
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

	// ---- Random ----
	{Name: "random", Signature: "random()", Params: nil, ReturnType: "number",
		Description: "Returns a random floating-point number in the range [0.0, 1.0).", Category: CatRandom},
	{Name: "random_range", Signature: "random_range(min, max)", Params: []Param{p("min", "number"), p("max", "number")}, ReturnType: "number",
		Description: "Returns a random integer in the range [min, max] inclusive.", Category: CatRandom},

	// ---- Encoding ----
	{Name: "base64_encode", Signature: "base64_encode(str)", Params: []Param{p("str", "string")}, ReturnType: "string",
		Description: "Encodes a string to Base64.", Category: CatEncode},
	{Name: "base64_decode", Signature: "base64_decode(str)", Params: []Param{p("str", "string")}, ReturnType: "string",
		Description: "Decodes a Base64-encoded string.", Category: CatEncode},

	// ---- Hashing ----
	{Name: "sha256", Signature: "sha256(text)", Params: []Param{p("text", "string")}, ReturnType: "string",
		Description: "Computes the SHA-256 hash of text and returns it as a hex string.", Category: CatHash},
	{Name: "md5", Signature: "md5(text)", Params: []Param{p("text", "string")}, ReturnType: "string",
		Description: "Computes the MD5 hash of text and returns it as a hex string.", Category: CatHash},
	{Name: "sha1", Signature: "sha1(text)", Params: []Param{p("text", "string")}, ReturnType: "string",
		Description: "Computes the SHA-1 hash of text and returns it as a hex string.", Category: CatHash},
	{Name: "sha512", Signature: "sha512(text)", Params: []Param{p("text", "string")}, ReturnType: "string",
		Description: "Computes the SHA-512 hash of text and returns it as a hex string.", Category: CatHash},

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

	// ---- AI Configuration ----
	{Name: "ai_provider", Signature: "ai_provider(name)", Params: []Param{p("name", "string")}, ReturnType: "string",
		Description: "Sets the AI provider: \"openai\", \"anthropic\", \"deepseek\", or \"ollama\".", Category: CatAIConf},
	{Name: "ai_model", Signature: "ai_model(name)", Params: []Param{p("name", "string")}, ReturnType: "string",
		Description: "Sets the model name for the current AI provider.", Category: CatAIConf},
	{Name: "ai_host", Signature: "ai_host(url)", Params: []Param{p("url", "string")}, ReturnType: "string",
		Description: "Sets a custom API host URL, e.g. for local proxies or Ollama.", Category: CatAIConf},
	{Name: "ai_timeout", Signature: "ai_timeout(seconds)", Params: []Param{p("seconds", "number")}, ReturnType: "nil",
		Description: "Sets the AI request timeout in seconds.", Category: CatAIConf},
	{Name: "ai_cache", Signature: "ai_cache(on_off|ttl|'clear'|'stats')", Params: []Param{p("arg", "bool|number|string")}, ReturnType: "string",
		Description: "Enables/disables AI response caching. Pass true/on/ttl-minutes to enable, false/off to disable, 'clear' to flush, 'stats' for hit/miss count.", Category: CatAIConf},

	// ---- AI Chat ----
	{Name: "ai_chat", Signature: "ai_chat(system_prompt, user_prompt)", Params: []Param{p("system_prompt", "string"), p("user_prompt", "string")}, ReturnType: "string",
		Description: "Sends a chat request with system and user prompts. Returns the assistant's response.", Category: CatAIChat},
	{Name: "ai_chat_json", Signature: "ai_chat_json(system_prompt, user_prompt)", Params: []Param{p("system_prompt", "string"), p("user_prompt", "string")}, ReturnType: "any",
		Description: "Like ai_chat, but parses the response as JSON and returns the parsed value.", Category: CatAIChat},

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
