//go:build js

package object

import ()

func init() {
	Builtins = append(Builtins,
		BuiltinInfo{Name: "db_open", Fn: bDbOpenStub},
		BuiltinInfo{Name: "db_close", Fn: bDbCloseStub},
		BuiltinInfo{Name: "db_exec", Fn: bDbExecStub},
		BuiltinInfo{Name: "db_query", Fn: bDbQueryStub},
	)
}

func bDbOpenStub(args ...Object) Object {
	return &Error{Message: "db_open: SQLite not available in WASM"}
}

func bDbCloseStub(args ...Object) Object {
	return &Error{Message: "db_close: SQLite not available in WASM"}
}

func bDbExecStub(args ...Object) Object {
	return &Error{Message: "db_exec: SQLite not available in WASM"}
}

func bDbQueryStub(args ...Object) Object {
	return &Error{Message: "db_query: SQLite not available in WASM"}
}
