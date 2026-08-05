package eval

import "testing"

func TestTryCatchErrorMessageAccess(t *testing.T) {
	expectValue(t, "try\n    read_file \"/nonexistent/x.txt\"\ncatch e\n    e.message", "read_file: open /nonexistent/x.txt: no such file or directory")
	expectValue(t, "try\n    1 / 0\ncatch e\n    \"caught:\" ++ e.message", "caught:E003: division by zero")
}

func TestEvalZeroArityBuiltinWithArgs(t *testing.T) {
	expectValue(t, `ai_cost "reset"`, "cost metrics reset")
}
