package eval

import "testing"

func TestTryCatchErrorMessageAccess(t *testing.T) {
	expectValue(t, "try\n    read_file \"/nonexistent/x.txt\"\ncatch e\n    e", "<test>: read_file: open /nonexistent/x.txt: no such file or directory\n  in fn(read_file)")
	expectValue(t, "try\n    1 / 0\ncatch e\n    \"caught:\" ++ e", "caught:<test>: E003: division by zero")
}

func TestEvalZeroArityBuiltinWithArgs(t *testing.T) {
	expectValue(t, `ai_cost "reset"`, "cost metrics reset")
}
