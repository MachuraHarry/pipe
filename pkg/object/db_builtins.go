package object

// db_* stubs removed per Phase 4 — the 'sqlite' Pipe module now provides these.
// See SQLite.md and examples/sqlite.pipe.

func init() {
	// Previously registered db_open, db_close, db_exec, db_query as error stubs.
	// Now removed: consumers import the 'sqlite' module instead.
}
