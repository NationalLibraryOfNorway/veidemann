---
title: "Functions"
date: 2021-03-17T15:34:11+01:00
---

In addition to Starlarks [built in functions](https://github.com/bazelbuild/starlark/blob/master/spec.md#built-in-constants-and-functions),
Scope Checker defines a number of functions needed for building scope evaluation scripts.

{{< funcdef def="param(name)" >}}
Returns a named parameter from the Candidate URL as a String.
{{</funcdef >}}

{{< funcdef def="abort()" >}}
End script evaluation and return the current [Status]({{< ref "constants#status" >}}) set by either an explicit call
to [setStatus()]({{< ref "#setstatusstatus" >}}).
{{</funcdef >}}

{{< funcdef def="getStatus()" >}}
Returns the currently set [Status]({{< ref "constants#status" >}}) value.
{{< /funcdef >}}

{{< funcdef def="setStatus(status)" >}}
Set the [Status]({{< ref "constants#status" >}}) value to be returned by the script when evaluation ends.
{{< /funcdef >}}

{{< funcdef def="debug(boolean)" >}}
Turn on/of debugging.
{{< /funcdef >}}
