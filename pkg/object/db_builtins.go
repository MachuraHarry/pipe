//go:build !js

package object

import (
	"fmt"

	"github.com/MachuraHarry/pipe/pkg/db"
)

func init() {
	Builtins = append(Builtins,
		BuiltinInfo{Name: "db_open", Fn: bDbOpen},
		BuiltinInfo{Name: "db_close", Fn: bDbClose},
		BuiltinInfo{Name: "db_exec", Fn: bDbExec},
		BuiltinInfo{Name: "db_query", Fn: bDbQuery},
	)
}

func bDbOpen(args ...Object) Object {
	if len(args) != 1 {
		return &Error{Message: "db_open expects 1 argument (path)"}
	}
	path, ok := args[0].(*String)
	if !ok {
		return &Error{Message: "db_open: argument must be a string (file path)"}
	}
	handle, openErr := db.Open(path.Value)
	if openErr != nil {
		return &Error{Message: openErr.Error()}
	}
	return &Integer{Value: int64(handle)}
}

func bDbClose(args ...Object) Object {
	if len(args) != 1 {
		return &Error{Message: "db_close expects 1 argument (handle)"}
	}
	h, ok := ToInt(args[0])
	if !ok {
		return &Error{Message: "db_close: argument must be a number (database handle)"}
	}
	if closeErr := db.Close(int(h)); closeErr != nil {
		return &Error{Message: closeErr.Error()}
	}
	return NILOBJ
}

func bDbExec(args ...Object) Object {
	if len(args) < 2 {
		return &Error{Message: "db_exec expects 2 arguments (handle, sql)"}
	}
	h, ok := ToInt(args[0])
	if !ok {
		return &Error{Message: "db_exec: first argument must be a number (database handle)"}
	}
	sqlStr, ok := args[1].(*String)
	if !ok {
		return &Error{Message: "db_exec: second argument must be a string (SQL)"}
	}
	rows, execErr := db.Exec(int(h), sqlStr.Value)
	if execErr != nil {
		return &Error{Message: execErr.Error()}
	}
	return &Integer{Value: rows}
}

func bDbQuery(args ...Object) Object {
	if len(args) < 2 {
		return &Error{Message: "db_query expects 2 arguments (handle, sql)"}
	}
	h, ok := ToInt(args[0])
	if !ok {
		return &Error{Message: "db_query: first argument must be a number (database handle)"}
	}
	sqlStr, ok := args[1].(*String)
	if !ok {
		return &Error{Message: "db_query: second argument must be a string (SQL)"}
	}
	rows, queryErr := db.Query(int(h), sqlStr.Value)
	if queryErr != nil {
		return &Error{Message: queryErr.Error()}
	}

	elems := make([]Object, len(rows))
	for i, row := range rows {
		pairs := make(map[string]Object)
		for k, v := range row {
			switch val := v.(type) {
			case float64:
				pairs[k] = &Float{Value: val}
			case string:
				pairs[k] = &String{Value: val}
			case nil:
				pairs[k] = NILOBJ
			default:
				pairs[k] = &String{Value: fmt.Sprintf("%v", val)}
			}
		}
		elems[i] = &Map{Pairs: pairs}
	}
	return &List{Elements: elems}
}
