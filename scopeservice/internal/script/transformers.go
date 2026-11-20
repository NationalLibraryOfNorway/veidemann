package script

import (
	"strings"

	"github.com/nationallibraryofnorway/whatwg-url/canonicalizer"
	"github.com/nationallibraryofnorway/whatwg-url/url"
	"go.starlark.net/starlark"
	"golang.org/x/text/encoding/unicode"
)

func init() {
	starlark.Universe["removeQuery"] = starlark.NewBuiltin("removeQuery", removeQuery)
}

func removeQuery(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var q string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "query", &q); err != nil {
		return nil, err
	}

	qUrl := thread.Local(urlKey).(*UrlValue)

	decoder := unicode.UTF8.NewDecoder()
	qUrl.parsedUri.RewriteQuery(func(_ *url.Url, query url.Query, hasQuery bool) (url.Query, bool) {
		query.Filter(func(name, value string, hasEquals bool) bool {
			// Empty separator terms are not parameters, even when removing "".
			if name == "" && value == "" && !hasEquals {
				return true
			}
			// Decode only names for matching; retain the serialized surviving
			// terms. UTF-8 replacement decoding cannot fail and preserves BOMs.
			name = url.PercentDecode(strings.ReplaceAll(name, "+", " "))
			name, _ = decoder.String(name)
			return name != q
		})
		return query, hasQuery && len(query) != 0
	})
	// Retain the shared canonicalization policy for empty-valued parameters.
	canonicalizer.RemoveEmptyQueryEquals(qUrl.parsedUri)

	return starlark.None, nil
}
