package object

import (
	"strings"

	"github.com/MachuraHarry/pipe/pkg/util"
)

func bUpper(args ...Object) Object {
	if s, ok := strArg(args, "upper"); ok {
		return &String{Value: strings.ToUpper(s.Value)}
	}
	return err("upper expects a string")
}

func bLower(args ...Object) Object {
	if s, ok := strArg(args, "lower"); ok {
		return &String{Value: strings.ToLower(s.Value)}
	}
	return err("lower expects a string")
}

func bTrim(args ...Object) Object {
	if s, ok := strArg(args, "trim"); ok {
		return &String{Value: strings.TrimSpace(s.Value)}
	}
	return err("trim expects a string")
}

func bSplit(args ...Object) Object {
	if len(args) != 2 {
		return err("split expects 2 arguments")
	}
	s, ok := args[0].(*String)
	d, ok2 := args[1].(*String)
	if !ok || !ok2 {
		return err("split: both arguments must be strings")
	}
	parts := strings.Split(s.Value, d.Value)
	elems := make([]Object, len(parts))
	for i, p := range parts {
		elems[i] = &String{Value: p}
	}
	return &List{Elements: elems}
}

func bJoin(args ...Object) Object {
	if len(args) != 2 {
		return err("join expects 2 arguments")
	}
	l, ok := args[0].(*List)
	d, ok2 := args[1].(*String)
	if !ok || !ok2 {
		return err("join: list and string expected")
	}
	parts := make([]string, len(l.Elements))
	for i, e := range l.Elements {
		parts[i] = e.Inspect()
	}
	return &String{Value: strings.Join(parts, d.Value)}
}

func bContains(args ...Object) Object {
	if len(args) != 2 {
		return err("contains expects 2 arguments")
	}
	switch c := args[0].(type) {
	case *String:
		if sub, ok := args[1].(*String); ok {
			return NativeBoolToBoolean(strings.Contains(c.Value, sub.Value))
		}
		return err("contains: Substring must be a string")
	case *List:
		for _, e := range c.Elements {
			if ValuesEqual(e, args[1]) {
				return TRUE
			}
		}
		return FALSE
	case *Map:
		needle, ok := args[1].(*String)
		if !ok {
			return err("contains: Map key must be a string")
		}
		_, exists := c.Get(needle.Value)
		return NativeBoolToBoolean(exists)
	}
	return err("contains expects string, list, or map")
}

func bRepeat(args ...Object) Object {
	if len(args) != 2 {
		return err("repeat expects 2 arguments (string, count)")
	}
	str, ok := args[0].(*String)
	if !ok {
		return err("repeat: first argument must be a string")
	}
	count, ok := ToInt(args[1])
	if !ok || count < 0 {
		return err("repeat: count must be a non-negative number")
	}
	return &String{Value: strings.Repeat(str.Value, int(count))}
}

func bReplace(args ...Object) Object {
	if len(args) != 3 {
		return err("replace expects 3 arguments (str, old, new)")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("replace: first argument must be a string")
	}
	old, ok := args[1].(*String)
	if !ok {
		return err("replace: second argument must be a string")
	}
	newS, ok := args[2].(*String)
	if !ok {
		return err("replace: third argument must be a string")
	}
	return &String{Value: strings.Replace(s.Value, old.Value, newS.Value, 1)}
}

func bReplaceAll(args ...Object) Object {
	if len(args) != 3 {
		return err("replace_all expects 3 arguments (str, old, new)")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("replace_all: first argument must be a string")
	}
	old, ok := args[1].(*String)
	if !ok {
		return err("replace_all: second argument must be a string")
	}
	newS, ok := args[2].(*String)
	if !ok {
		return err("replace_all: third argument must be a string")
	}
	return &String{Value: strings.ReplaceAll(s.Value, old.Value, newS.Value)}
}

func bCsvParse(args ...Object) Object {
	if len(args) != 1 {
		return err("csv_parse expects 1 argument (text)")
	}
	text, ok := args[0].(*String)
	if !ok {
		return err("csv_parse: argument must be a string")
	}
	rows, parseErr := util.ParseCSV(text.Value)
	if parseErr != nil {
		return err(parseErr.Error())
	}

	elems := make([]Object, len(rows))
	for i, row := range rows {
		pairs := make(map[string]Object)
		for k, v := range row {
			pairs[k] = &String{Value: v}
		}
		elems[i] = MapFromGo(pairs)
	}
	return &List{Elements: elems}
}

func bCsvFormat(args ...Object) Object {
	if len(args) < 1 {
		return err("csv_format expects at least 1 argument (list of maps)")
	}
	list, ok := args[0].(*List)
	if !ok {
		return err("csv_format: first argument must be a list of maps")
	}

	var headers []string
	if len(args) >= 2 {
		headerList, ok := args[1].(*List)
		if ok {
			for _, h := range headerList.Elements {
				if s, ok := h.(*String); ok {
					headers = append(headers, s.Value)
				}
			}
		}
	}

	if len(headers) == 0 && len(list.Elements) > 0 {
		if first, ok := list.Elements[0].(*Map); ok {
			headers = append(headers, first.Keys()...)
		}
	}

	rows := make([]map[string]string, len(list.Elements))
	for i, elem := range list.Elements {
		m, ok := elem.(*Map)
		if !ok {
			return err("csv_format: all elements must be maps")
		}
		row := make(map[string]string)
		for _, p := range m.Pairs {
			row[p.Key] = p.Value.Inspect()
		}
		rows[i] = row
	}

	return &String{Value: util.FormatCSV(rows, headers)}
}
