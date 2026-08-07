package object

func bOk(args ...Object) Object {
	if len(args) != 1 {
		return err("Ok expects 1 argument")
	}
	return &Result{Ok: true, Val: args[0]}
}

func bErr(args ...Object) Object {
	if len(args) != 1 {
		return err("Err expects 1 argument (error message)")
	}
	msg, ok := args[0].(*String)
	if !ok {
		msg = &String{Value: args[0].Inspect()}
	}
	return &Result{Ok: false, Err: msg.Value}
}

func bIsOk(args ...Object) Object {
	if len(args) != 1 {
		return err("is_ok expects 1 argument")
	}
	r, ok := args[0].(*Result)
	if !ok {
		return FALSE
	}
	return NativeBoolToBoolean(r.Ok)
}

func bIsErr(args ...Object) Object {
	if len(args) != 1 {
		return err("is_err expects 1 argument")
	}
	r, ok := args[0].(*Result)
	if !ok {
		return TRUE
	}
	return NativeBoolToBoolean(!r.Ok)
}

func bUnwrap(args ...Object) Object {
	if len(args) != 1 {
		return err("unwrap expects 1 argument")
	}
	r, ok := args[0].(*Result)
	if !ok {
		return err("unwrap expects a result")
	}
	if !r.Ok {
		return err("unwrap on Err: " + r.Err)
	}
	return r.Val
}

func bUnwrapOr(args ...Object) Object {
	if len(args) != 2 {
		return err("unwrap_or expects 2 arguments (Result, Default)")
	}
	r, ok := args[0].(*Result)
	if !ok {
		return args[1]
	}
	if !r.Ok {
		return args[1]
	}
	return r.Val
}
