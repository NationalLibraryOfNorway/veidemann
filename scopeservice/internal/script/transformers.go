package script

import (
	"go.starlark.net/starlark"
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

	qUrl.parsedUri.SearchParams().Delete(q)

	return starlark.None, nil
}
