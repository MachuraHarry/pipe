package ai

import (
	"fmt"
	"strings"
)

var AITryFixPrompt = `You are a Pipe expression fixer. Pipe is a pipeline language with space-separated args.

RULES:
1. Return ONLY the corrected expression, no explanation, no markdown.
2. ALWAYS wrap the result in (parentheses): (to_num "42") * 3
3. If the fix is multi-statement, write each statement on its own line.
4. Do NOT add comments or explanations.

PIPE BUILTINS:
  Conversion: to_num, to_str
  Math: abs, min, max, pow, sqrt, round
  String: len, upper, lower, trim, split, join, contains, at
  List: len, push, pop, at, sort, range, map, filter, reduce, each, slice_list
  Map: get, set, keys, values
  Type: type_of, is_num, is_str, is_list, is_map, is_nil
  JSON: parse_json, to_json
  Regex: regex_match, regex_replace
  Time: now, format_time
  Random: random, random_range
  Hash: sha256, md5, sha1, sha512
  Encoding: base64_encode, base64_decode

PIPE KEYWORDS:
  fn, match, if, else, elif, while, for, break, continue, return, defer
  import, export, enum, true, false, nil, try, catch, try_ai, test, assert, assert_eq

PIPE OPERATORS:
  Arithmetic: + - * / % **
  Comparison: == != < > <= >=
  Logic: && || ! not
  String: ++ (concat)
  Pipeline: > (pipe) >> (parallel)

PIPE SYNTAX:
  Function call: fn_name arg1 arg2       (space-separated, no commas)
  Wrap in parens if nested: (fn arg1 arg2)
  Lists: [elem1, elem2, elem3]           (comma-separated)
  Maps: {key1: val1, key2: val2}         (key: value, comma-separated)
  Index access: list[index]  or  map["key"]
  Assignment: name: value
  Strings: "double-quoted only"

ERROR FIXING STRATEGIES:
  Type errors (E002): If string looks numeric, use to_num. If number should be string, use to_str.
  Division by zero (E003): Replace divisor with max(divisor, 1) or guard with if.
  Not a function (E004): Wrap in parentheses or use a builtin instead.
  Operator not supported (E005): Convert one operand to the other's type.
  Cannot index (E006): Check list length with len first, or use get with map.
  Undefined variable (E001): Use a literal default value like 0, "", nil, or [].

If truly impossible to fix, return exactly: UNFIXABLE`

func TryAIFixExpression(errorMsg, sourceCode string) string {
	if !isAIFixableCode(extractAICode(errorMsg)) {
		return ""
	}

	for attempt := 1; attempt <= 3; attempt++ {
		var prompt string
		if attempt == 1 {
			prompt = fmt.Sprintf("Error: %s\nExpression: %s\nFix it.", errorMsg, sourceCode)
		} else {
			prompt = fmt.Sprintf("Error: %s\nExpression: %s\nPrevious fix failed. Try a different approach.\nFix it.", errorMsg, sourceCode)
		}

		resp, aiErr := Chat(ChatRequest{
			Messages: []Message{
				{Role: "system", Content: AITryFixPrompt},
				{Role: "user", Content: prompt},
			},
		})
		if aiErr != nil {
			LogTryAIFix(extractAICode(errorMsg), sourceCode, "", attempt, false)
			return ""
		}

		fix := strings.TrimSpace(resp.Content)
		if fix == "" || strings.HasPrefix(fix, "UNFIXABLE") || strings.HasPrefix(fix, "unfixable") {
			LogTryAIFix(extractAICode(errorMsg), sourceCode, fix, attempt, false)
			return ""
		}

		LogTryAIFix(extractAICode(errorMsg), sourceCode, fix, attempt, true)
		return fix
	}

	return ""
}

func isAIFixableCode(code string) bool {
	switch code {
	case "E001", "E002", "E003", "E004", "E005", "E006":
		return true
	}
	return false
}

func extractAICode(msg string) string {
	if idx := strings.Index(msg, "E0"); idx >= 0 {
		end := idx + 4
		if end > len(msg) {
			end = len(msg)
		}
		return msg[idx:end]
	}
	return ""
}
