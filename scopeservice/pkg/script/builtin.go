package script

import (
	"fmt"
	"go.starlark.net/starlark"
)

func init() {
	starlark.Universe["param"] = starlark.NewBuiltin("param", param)
	starlark.Universe["abort"] = starlark.NewBuiltin("abort", abort)
	starlark.Universe["url"] = starlark.NewBuiltin("url", getUrl)
	starlark.Universe["getStatus"] = starlark.NewBuiltin("getStatus", getStatus)
	starlark.Universe["setStatus"] = starlark.NewBuiltin("setStatus", setStatus)
	starlark.Universe["debug"] = starlark.NewBuiltin("debug", debug)
}

func abort(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs(b.Name(), args, kwargs); err != nil {
		return nil, err
	}
	if match, ok := b.Receiver().(Match); ok {
		if match {
			return match, EndOfComputation
		} else {
			return match, nil
		}
	}
	return starlark.None, EndOfComputation
}

func param(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var name string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "name", &name); err != nil {
		return nil, err
	}
	v := thread.Local(name)
	if v == nil {
		return starlark.None, fmt.Errorf("no value with name '%v'", name)
	}
	if result, ok := v.(starlark.String); ok {
		return result, nil
	} else {
		return starlark.None, fmt.Errorf("could not convert '%v' to string", v)
	}
}

func getUrl(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs(b.Name(), args, kwargs); err != nil {
		return nil, err
	}
	if result, ok := thread.Local(urlKey).(*UrlValue); ok {
		return result, nil
	} else {
		return starlark.None, nil
	}
}

func getStatus(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs(b.Name(), args, kwargs); err != nil {
		return nil, err
	}
	if result, ok := thread.Local(resultKey).(Status); ok {
		return result, nil
	} else {
		return starlark.None, nil
	}
}

func setStatus(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var status Status
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "status", &status); err != nil {
		return nil, err
	}
	thread.SetLocal(resultKey, status)
	printDebug(thread, b, args, kwargs, "status="+status.String())
	return starlark.None, nil
}

func debug(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	debug := starlark.True
	stacktrace := starlark.False
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "debug?", &debug, "stacktrace?", &stacktrace); err != nil {
		return nil, err
	}
	thread.SetLocal(debugKey, debug)
	thread.SetLocal(stacktraceKey, stacktrace)
	return starlark.None, nil
}
