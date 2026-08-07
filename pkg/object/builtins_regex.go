package object

import "regexp"

func bRegexMatch(args ...Object) Object {
	if len(args) != 2 {
		return err("regex_match expects 2 arguments (Pattern, Text)")
	}
	pat, ok := args[0].(*String)
	txt, ok2 := args[1].(*String)
	if !ok || !ok2 {
		return err("regex_match: Pattern und Text must be strings")
	}
	matched, e := regexp.MatchString(pat.Value, txt.Value)
	if e != nil {
		return err("regex_match: " + e.Error())
	}
	if matched {
		return TRUE
	}
	return FALSE
}

func bRegexReplace(args ...Object) Object {
	if len(args) != 3 {
		return err("regex_replace expects 3 arguments (Pattern, replacement, Text)")
	}
	pat, ok := args[0].(*String)
	repl, ok2 := args[1].(*String)
	txt, ok3 := args[2].(*String)
	if !ok || !ok2 || !ok3 {
		return err("regex_replace: All arguments must be strings")
	}
	re, e := regexp.Compile(pat.Value)
	if e != nil {
		return err("regex_replace: " + e.Error())
	}
	return &String{Value: re.ReplaceAllString(txt.Value, repl.Value)}
}
