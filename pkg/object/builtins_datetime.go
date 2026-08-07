package object

import "time"

func bNow(args ...Object) Object {
	if len(args) > 0 {
		return err("now takes no arguments")
	}
	ts := time.Now().Unix()
	return &Integer{Value: ts}
}

func bFormatTime(args ...Object) Object {
	if len(args) < 1 || len(args) > 2 {
		return err("format_time expects 1-2 arguments (Timestamp, Format?)")
	}
	ts, ok := ToInt(args[0])
	if !ok {
		return err("format_time: Timestamp must be a number")
	}
	layout := "2006-01-02 15:04:05"
	if len(args) >= 2 {
		s, ok := args[1].(*String)
		if !ok {
			return err("format_time: Format must be a string")
		}
		layout = s.Value
	}
	t := time.Unix(ts, 0)
	return &String{Value: t.Format(layout)}
}

func bParseDate(args ...Object) Object {
	if len(args) < 1 || len(args) > 2 {
		return err("parse_date expects 1-2 arguments (DateString, Format?)")
	}
	s, ok := args[0].(*String)
	if !ok {
		return err("parse_date: DateString must be a string")
	}
	layout := "2006-01-02"
	if len(args) >= 2 {
		ls, ok := args[1].(*String)
		if !ok {
			return err("parse_date: Format must be a string")
		}
		layout = ls.Value
	}
	t, e := time.Parse(layout, s.Value)
	if e != nil {
		return err("parse_date: " + e.Error())
	}
	return &Integer{Value: t.Unix()}
}
