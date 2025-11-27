package script

import (
	"errors"
	"fmt"

	commonsV1 "github.com/NationalLibraryOfNorway/veidemann/api/commons/v1"
	scopecheckerV1 "github.com/NationalLibraryOfNorway/veidemann/api/scopechecker/v1"
	"go.starlark.net/starlark"
)

// init inserts constants into starlark environment.
//
//   - -5    RUNTIME_EXCEPTION           Unexpected runtime exception.
//   - -7    ILLEGAL_URI                 URI recognized as unsupported or illegal.
//   - -4000 CHAFF_DETECTION             Chaff detection of traps/content with negligible value applied.
//   - -4001 TOO_MANY_HOPS               The URI is too many link hops away from the seed.
//   - -4002 TOO_MANY_TRANSITIVE_HOPS    The URI is too many embed/transitive hops away from the last URI in scope.
//   - -5001 BLOCKED                     Blocked from fetch by user setting.
//   - -5002 BLOCKED_BY_CUSTOM_PROCESSOR Blocked by a custom processor.
func init() {
	for k, v := range statusValues {
		starlark.Universe[k] = v
	}
}

var (
	Include                         = Status(scopecheckerV1.ScopeCheckResponse_INCLUDE)
	RuntimeException         Status = -5
	IllegalUri               Status = -7
	ChaffDetection           Status = -4000
	TooManyHops              Status = -4001
	TooManyTransitiveHops    Status = -4002
	Blocked                  Status = -5001
	BlockedByCustomProcessor Status = -5002
)

var statusNames = map[Status]string{
	Include:                  "Include",
	RuntimeException:         "RuntimeException",
	IllegalUri:               "IllegalUri",
	ChaffDetection:           "ChaffDetection",
	TooManyHops:              "TooManyHops",
	TooManyTransitiveHops:    "TooManyTransitiveHops",
	Blocked:                  "Blocked",
	BlockedByCustomProcessor: "BlockedByCustomProcessor",
}

var statusValues = map[string]Status{
	"Include":                  Include,
	"RuntimeException":         RuntimeException,
	"IllegalUri":               IllegalUri,
	"ChaffDetection":           ChaffDetection,
	"TooManyHops":              TooManyHops,
	"TooManyTransitiveHops":    TooManyTransitiveHops,
	"Blocked":                  Blocked,
	"BlockedByCustomProcessor": BlockedByCustomProcessor,
}

type Status int32

// Implement starlark.Value interface
func (s Status) Type() string          { return "end of computation status" }
func (s Status) Freeze()               {} // immutable
func (s Status) Truth() starlark.Bool  { return true }
func (s Status) Hash() (uint32, error) { return starlark.MakeUint(uint(s)).Hash() }
func (s Status) String() string        { return statusNames[s] }

// Implement starlark.Unpacker interface
func (s *Status) Unpack(v starlark.Value) error {
	switch val := v.(type) {
	case Status:
		*s = val
	case starlark.String:
		value, ok := statusValues[val.GoString()]
		if !ok {
			return errors.New("Illegal type " + val.String())
		}
		*s = value
	default:
		return errors.New("Illegal type " + val.String())
	}
	return nil
}

func (s Status) AsInt32() int32 {
	return int32(s)
}

func (s Status) asError(detail string) *wrappedError {
	return &wrappedError{
		Code:   s.AsInt32(),
		Msg:    s.String(),
		Detail: detail,
	}
}

type wrappedError commonsV1.Error

func (w *wrappedError) Error() string {
	return fmt.Sprintf("%d %s: %s", w.Code, w.Msg, w.Detail)
}
