package util

import (
	"encoding/csv"
	"fmt"
	"strings"
)

func ParseCSV(text string) ([]map[string]string, error) {
	r := csv.NewReader(strings.NewReader(strings.TrimSpace(text)))
	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("csv_parse: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("csv_parse: empty input")
	}

	headers := rows[0]
	var result []map[string]string
	for _, row := range rows[1:] {
		m := make(map[string]string)
		for i, val := range row {
			if i < len(headers) {
				m[headers[i]] = val
			}
		}
		result = append(result, m)
	}
	return result, nil
}

func FormatCSV(data []map[string]string, headers []string) string {
	var buf strings.Builder
	w := csv.NewWriter(&buf)

	if len(headers) > 0 {
		w.Write(headers)
		for _, row := range data {
			vals := make([]string, len(headers))
			for i, h := range headers {
				vals[i] = row[h]
			}
			w.Write(vals)
		}
	}
	w.Flush()
	return buf.String()
}
