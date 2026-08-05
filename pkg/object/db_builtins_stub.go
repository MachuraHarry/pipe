//go:build js || pipe_lite

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
	return &Error{Message: "db_open: SQLite not available in this build (use full build: make build)"}
}

func bDbCloseStub(args ...Object) Object {
	return &Error{Message: "db_close: SQLite not available in this build (use full build: make build)"}
}

func bDbExecStub(args ...Object) Object {
	return &Error{Message: "db_exec: SQLite not available in this build (use full build: make build)"}
}

func bDbQueryStub(args ...Object) Object {
	return &Error{Message: "db_query: SQLite not available in this build (use full build: make build)"}
}
