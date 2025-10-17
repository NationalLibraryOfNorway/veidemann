package script

import (
	"sort"

	"github.com/NationalLibraryOfNorway/veidemann/api/commons"
	"github.com/NationalLibraryOfNorway/veidemann/api/frontier"
	"github.com/nlnwa/whatwg-url/url"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

type UrlValue struct {
	qUri      *frontier.QueuedUri
	parsedUri *url.Url
}

func Url(u *frontier.QueuedUri) (*UrlValue, error) {
	r := &UrlValue{
		qUri: u,
	}
	var err error
	r.parsedUri, err = ScopeCanonicalizationProfile.Parse(u.Uri)
	return r, err
}

func (u *UrlValue) String() string {
	if u.parsedUri == nil {
		return u.qUri.Uri
	}
	return u.parsedUri.String()
}

func (u *UrlValue) AsCommonsParsedUri() *commons.ParsedUri {
	if u.parsedUri == nil {
		return nil
	}
	return &commons.ParsedUri{
		Href:     u.parsedUri.String(),
		Scheme:   u.parsedUri.Scheme(),
		Host:     u.parsedUri.Hostname(),
		Port:     int32(u.parsedUri.DecodedPort()),
		Username: u.parsedUri.Username(),
		Password: u.parsedUri.Password(),
		Path:     u.parsedUri.Pathname(),
		Query:    u.parsedUri.Query(),
		Fragment: u.parsedUri.Fragment(),
	}
}

func (u *UrlValue) Type() string {
	return "url"
}

func (u *UrlValue) Freeze() {
	panic("implement me")
}

func (u *UrlValue) Truth() starlark.Bool {
	panic("implement me")
}

func (u *UrlValue) Hash() (uint32, error) {
	panic("implement me")
}

func (u *UrlValue) Attr(name string) (starlark.Value, error) {
	return builtinAttr(u, name, urlMethods)
}

func (u *UrlValue) AttrNames() []string {
	return builtinAttrNames(urlMethods)
}

var urlMethods = map[string]*starlark.Builtin{
	"host": starlark.NewBuiltin("host", uriGetHost),
	"port": starlark.NewBuiltin("port", uriGetPort),
}

func uriGetHost(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 0); err != nil {
		return nil, err
	}

	u := b.Receiver().(*UrlValue)
	return starlark.String(u.parsedUri.Host()), nil
}

func uriGetPort(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 0); err != nil {
		return nil, err
	}

	u := b.Receiver().(*UrlValue)
	return starlark.String(u.parsedUri.Port()), nil
}

var (
	True  Match = true
	False Match = false
)

type Match bool

func (m Match) String() string {
	if m {
		return "True"
	} else {
		return "False"
	}
}

func (m Match) Type() string {
	return "match"
}

func (m Match) Freeze() {} // Immutable

func (m Match) Truth() starlark.Bool {
	return starlark.Bool(m)
}

func (m Match) Hash() (uint32, error) {
	return uint32(b2i(bool(m))), nil
}

func (m Match) CompareSameType(op syntax.Token, y_ starlark.Value, _ int) (bool, error) {
	y := y_.(Match)
	return threeway(op, b2i(bool(m))-b2i(bool(y))), nil
}

func (m Match) Attr(name string) (starlark.Value, error) {
	return builtinAttr(m, name, matchMethods)
}

func (m Match) AttrNames() []string {
	return builtinAttrNames(matchMethods)
}

var matchMethods = map[string]*starlark.Builtin{
	"then":      starlark.NewBuiltin("then", then),
	"otherwise": starlark.NewBuiltin("otherwise", otherwise),
}

func then(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return matchAction(thread, b, args, kwargs, false)
}

func otherwise(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return matchAction(thread, b, args, kwargs, true)
}

func matchAction(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple, invert bool) (starlark.Value, error) {
	var status Status
	var continueEvaluation = starlark.False
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "status", &status, "continueEvaluation?", &continueEvaluation); err != nil {
		return nil, err
	}
	match, _ := b.Receiver().(Match)
	if bool(match) != invert {
		printDebug(thread, b, args, kwargs, "status="+status.String())
		thread.SetLocal(resultKey, status)
		if continueEvaluation {
			return match, nil
		} else {
			return match, errEndOfComputation
		}
	} else {
		return match, nil
	}
}

// threeway interprets a three-way comparison value cmp (-1, 0, +1)
// as a boolean comparison (e.g. x < y).
func threeway(op syntax.Token, cmp int) bool {
	switch op {
	case syntax.EQL:
		return cmp == 0
	case syntax.NEQ:
		return cmp != 0
	case syntax.LE:
		return cmp <= 0
	case syntax.LT:
		return cmp < 0
	case syntax.GE:
		return cmp >= 0
	case syntax.GT:
		return cmp > 0
	}
	panic(op)
}

func b2i(b bool) int {
	if b {
		return 1
	} else {
		return 0
	}
}

func builtinAttr(recv starlark.Value, name string, methods map[string]*starlark.Builtin) (starlark.Value, error) {
	b := methods[name]
	if b == nil {
		return nil, nil // no such method
	}
	return b.BindReceiver(recv), nil
}

func builtinAttrNames(methods map[string]*starlark.Builtin) []string {
	names := make([]string, 0, len(methods))
	for name := range methods {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
