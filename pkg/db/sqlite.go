package db

import (
	"database/sql"
	"fmt"
	"sync"

	_ "modernc.org/sqlite"
)

var (
	dbRegistry   = map[int]*sql.DB{}
	dbRegistryMu sync.Mutex
	nextHandle   = 1
)

func Open(path string) (int, error) {
	d, err := sql.Open("sqlite", path)
	if err != nil {
		return 0, fmt.Errorf("db_open: %w", err)
	}
	d.SetMaxOpenConns(1)

	dbRegistryMu.Lock()
	handle := nextHandle
	nextHandle++
	dbRegistry[handle] = d
	dbRegistryMu.Unlock()

	return handle, nil
}

func Close(handle int) error {
	dbRegistryMu.Lock()
	d, ok := dbRegistry[handle]
	if ok {
		delete(dbRegistry, handle)
	}
	dbRegistryMu.Unlock()

	if !ok {
		return fmt.Errorf("db_close: invalid handle %d", handle)
	}
	return d.Close()
}

func Exec(handle int, sqlText string) (int64, error) {
	d, ok := getDB(handle)
	if !ok {
		return 0, fmt.Errorf("db_exec: invalid handle %d", handle)
	}
	result, err := d.Exec(sqlText)
	if err != nil {
		return 0, fmt.Errorf("db_exec: %w", err)
	}
	return result.RowsAffected()
}

func Query(handle int, sqlText string) ([]map[string]interface{}, error) {
	d, ok := getDB(handle)
	if !ok {
		return nil, fmt.Errorf("db_query: invalid handle %d", handle)
	}
	rows, err := d.Query(sqlText)
	if err != nil {
		return nil, fmt.Errorf("db_query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("db_query: %w", err)
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("db_query: %w", err)
		}

		row := make(map[string]interface{})
		for i, col := range cols {
			row[col] = normalizeValue(values[i])
		}
		results = append(results, row)
	}

	return results, nil
}

func getDB(handle int) (*sql.DB, bool) {
	dbRegistryMu.Lock()
	defer dbRegistryMu.Unlock()
	d, ok := dbRegistry[handle]
	return d, ok
}

func normalizeValue(v interface{}) interface{} {
	switch val := v.(type) {
	case []byte:
		return string(val)
	case int64:
		return float64(val)
	case float64:
		return val
	case nil:
		return nil
	default:
		return fmt.Sprintf("%v", val)
	}
}
