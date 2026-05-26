package script

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/scopeservice/pkg/telemetry"

	commonsV1 "github.com/NationalLibraryOfNorway/veidemann/api/commons/v1"
	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	scopecheckerV1 "github.com/NationalLibraryOfNorway/veidemann/api/scopechecker/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

const (
	urlKey        = "url"
	resultKey     = "result"
	debugKey      = "debug"
	stacktraceKey = "stacktrace"
)

var errEndOfComputation = errors.New("end of computation")

var scriptLogger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).With().
	Timestamp().Logger().Level(zerolog.DebugLevel)

func newScriptOptions() *syntax.FileOptions {
	return &syntax.FileOptions{
		Set:            true,
		Recursion:      true,
		GlobalReassign: true,
	}
}

func parseScopeURI(qUri *frontierV1.QueuedUri, consoleLog *strings.Builder) (*UrlValue, *commonsV1.ParsedUri, *scopecheckerV1.ScopeCheckResponse) {
	qUrl, err := Url(qUri)
	if err != nil {
		return nil, nil, &scopecheckerV1.ScopeCheckResponse{
			Evaluation:      scopecheckerV1.ScopeCheckResponse_EXCLUDE,
			ExcludeReason:   IllegalUri.AsInt32(),
			IncludeCheckUri: &commonsV1.ParsedUri{Href: qUri.Uri},
			Error: &commonsV1.Error{
				Code:   IllegalUri.AsInt32(),
				Msg:    "error parsing uri",
				Detail: err.Error(),
			},
			Console: consoleLog.String(),
		}
	}
	return qUrl, qUrl.AsCommonsParsedUri(), nil
}

func compileScopeProgram(name string, src interface{}, options *syntax.FileOptions) (*starlark.Program, error) {
	t := prometheus.NewTimer(telemetry.CompileScriptSeconds)
	defer t.ObserveDuration()
	_, prog, err := starlark.SourceProgramOptions(options, name, src, starlark.StringDict{}.Has)
	return prog, err
}

func newScopeThread(consoleLog *strings.Builder) *starlark.Thread {
	return &starlark.Thread{
		Name: "scope",
		Print: func(thread *starlark.Thread, msg string) {
			if thread.CallStackDepth() > 1 {
				line := thread.CallFrame(1).Pos.String() + " " + msg
				consoleLog.WriteString(line)
				consoleLog.WriteByte('\n')
				scriptLogger.Debug().Msg(line)
			} else {
				consoleLog.WriteString(msg)
				consoleLog.WriteByte('\n')
				scriptLogger.Debug().Msg(msg)
			}
		},
	}
}

func setThreadLocals(thread *starlark.Thread, qUrl *UrlValue, qUri *frontierV1.QueuedUri, debug bool) {
	thread.SetLocal(urlKey, qUrl)
	for _, a := range qUri.Annotation {
		thread.SetLocal(a.Key, starlark.String(a.Value))
	}
	thread.SetLocal(debugKey, starlark.Bool(debug))
}

func executeScopeProgram(prog *starlark.Program, thread *starlark.Thread) error {
	t := prometheus.NewTimer(telemetry.ExecuteScriptSeconds)
	defer t.ObserveDuration()
	_, err := prog.Init(thread, nil)
	return err
}

func runtimeErrorResponse(includeCheckUri *commonsV1.ParsedUri, console, msg, detail string) *scopecheckerV1.ScopeCheckResponse {
	return &scopecheckerV1.ScopeCheckResponse{
		Evaluation:      scopecheckerV1.ScopeCheckResponse_EXCLUDE,
		ExcludeReason:   RuntimeException.AsInt32(),
		IncludeCheckUri: includeCheckUri,
		Error: &commonsV1.Error{
			Code:   RuntimeException.AsInt32(),
			Msg:    msg,
			Detail: detail,
		},
		Console: console,
	}
}

func responseFromExecutionError(err error, includeCheckUri *commonsV1.ParsedUri, console string) *scopecheckerV1.ScopeCheckResponse {
	if err == nil {
		return nil
	}

	evalErr := new(starlark.EvalError)
	if errors.As(err, &evalErr) {
		if errors.Is(evalErr, errEndOfComputation) {
			return nil
		}

		w := new(wrappedError)
		if errors.As(evalErr, &w) {
			e := (*commonsV1.Error)(w)
			return &scopecheckerV1.ScopeCheckResponse{
				Evaluation:      scopecheckerV1.ScopeCheckResponse_EXCLUDE,
				ExcludeReason:   e.Code,
				IncludeCheckUri: includeCheckUri,
				Error:           e,
				Console:         console,
			}
		}

		return runtimeErrorResponse(includeCheckUri, console, "error executing scope script", evalErr.Backtrace())
	}

	return runtimeErrorResponse(includeCheckUri, console, "unknown error executing scope script", err.Error())
}

func responseFromThreadResult(thread *starlark.Thread, includeCheckUri *commonsV1.ParsedUri, console string) *scopecheckerV1.ScopeCheckResponse {
	s, ok := thread.Local(resultKey).(Status)
	if ok {
		if s == 0 {
			return &scopecheckerV1.ScopeCheckResponse{
				Evaluation:      scopecheckerV1.ScopeCheckResponse_INCLUDE,
				IncludeCheckUri: includeCheckUri,
				Console:         console,
			}
		}
		return &scopecheckerV1.ScopeCheckResponse{
			Evaluation:      scopecheckerV1.ScopeCheckResponse_EXCLUDE,
			ExcludeReason:   s.AsInt32(),
			IncludeCheckUri: includeCheckUri,
			Console:         console,
		}
	}

	return &scopecheckerV1.ScopeCheckResponse{
		Evaluation:      scopecheckerV1.ScopeCheckResponse_EXCLUDE,
		ExcludeReason:   Blocked.AsInt32(),
		IncludeCheckUri: includeCheckUri,
		Error:           (*commonsV1.Error)(Blocked.asError("No scope rules matched")),
		Console:         console,
	}
}

// RunScopeScript runs the Scope checking script and returns the Scope status.
func RunScopeScript(name string, src interface{}, qUri *frontierV1.QueuedUri, debug bool) *scopecheckerV1.ScopeCheckResponse {
	options := newScriptOptions()
	consoleLog := strings.Builder{}

	qUrl, includeCheckUri, parseResponse := parseScopeURI(qUri, &consoleLog)
	if parseResponse != nil {
		return parseResponse
	}

	prog, err := compileScopeProgram(name, src, options)
	if err != nil {
		return runtimeErrorResponse(includeCheckUri, consoleLog.String(), "error parsing scope script", err.Error())
	}

	thread := newScopeThread(&consoleLog)
	setThreadLocals(thread, qUrl, qUri, debug)

	err = executeScopeProgram(prog, thread)
	if response := responseFromExecutionError(err, includeCheckUri, consoleLog.String()); response != nil {
		return response
	}

	return responseFromThreadResult(thread, includeCheckUri, consoleLog.String())
}
