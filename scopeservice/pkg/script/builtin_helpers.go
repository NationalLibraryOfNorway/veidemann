package script

import (
	"fmt"
	"github.com/pkg/errors"
	"go.starlark.net/starlark"
	"strconv"
	"strings"
)

// Error indicating input was starlark.None type
var None = errors.New("None")

func debugEnabled(thread *starlark.Thread) bool {
	if debug, ok := thread.Local(debugKey).(starlark.Bool); ok {
		return bool(debug)
	} else {
		return false
	}
}

func stackTraceEnabled(thread *starlark.Thread) bool {
	if stacktrace, ok := thread.Local(stacktraceKey).(starlark.Bool); ok {
		return bool(stacktrace)
	} else {
		return false
	}
}

func printDebugf(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple, format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	printDebug(thread, b, args, kwargs, msg)
}

func printDebug(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple, msg string) {
	if debugEnabled(thread) {
		var funcName string
		if b.Receiver() != nil {
			funcName = fmt.Sprintf("%v.%v", b.Receiver().Type(), b.Name())
		} else {
			funcName = fmt.Sprintf("%v", b.Name())
		}
		m := fmt.Sprintf("%v(%v) %v", funcName, joinArgs(args, kwargs), msg)
		if stackTraceEnabled(thread) {
			m += "\n" + thread.CallStack().String()
		}
		thread.Print(thread, m)
	}
}

func joinArgs(a starlark.Tuple, k []starlark.Tuple) string {
	var b strings.Builder
	if len(a) > 0 {
		b.WriteString(a[0].String())
		for _, s := range a[1:] {
			b.WriteString(", ")
			b.WriteString(s.String())
		}
	}

	if len(k) > 0 {
		if len(a) > 0 {
			b.WriteString(", ")
		}
		b.WriteString(string(k[0][0].(starlark.String)) + "=" + k[0][1].String())
		for _, s := range k[1:] {
			b.WriteString(", ")
			b.WriteString(string(s[0].(starlark.String)) + "=" + s[1].String())
		}
	}

	return b.String()
}

func parameterAsInt64(v starlark.Value) (int64, error) {
	if v == nil {
		return 0, None
	}

	switch t := v.(type) {
	case starlark.String:
		if t == "None" {
			return 0, None
		}
		i, err := strconv.ParseInt(string(t), 10, 0)
		if err != nil {
			return 0, err
		}
		return i, nil
	case starlark.Int:
		return t.BigInt().Int64(), nil
	default:
		return 0, None
	}
}

func parameterAsBool(v starlark.Value) bool {
	if v == nil {
		return false
	}

	switch t := v.(type) {
	case starlark.String:
		switch strings.ToLower(string(t)) {
		case "true":
			fallthrough
		case "yes":
			fallthrough
		case "ok":
			return true
		default:
			return false
		}
	default:
		return bool(v.Truth())
	}
}
