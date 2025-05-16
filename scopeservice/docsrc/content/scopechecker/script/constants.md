---
title: "Constants"
date: 2021-03-17T15:34:11+01:00
---

## Starlark built in constants
Starlark defines fundamental values and functions needed by all Starlark programs like `None`, `True` and `False`.
The complete list of Starlark built in constants and functions can be found
[here](https://github.com/bazelbuild/starlark/blob/master/spec.md#built-in-constants-and-functions).

## Status

Scope Checker defines a number of constants to be used as result of scope evaluation. The different statuses corresponds
to the special status codes used in Veidemann logs which in turn is an extended set of Heritrix status codes.

{{< funcdef def="Include" >}}
Candidate URI is in scope. The status code will be the result of fetching the actual resource.
{{< /funcdef >}}

{{< funcdef def="Blocked" >}}
Candidate URL is blocked from fetch.
Status code is `-5001`.
{{</funcdef >}}

{{< funcdef def="BlockedByCustomProcessor" >}}
Blocked by a custom processor.
Status code is `-5002`.
{{< /funcdef >}}

{{< funcdef def="ChaffDetection" >}}
Chaff detection of traps/content with negligible value applied. Status code is `-4000`.
{{< /funcdef >}}

{{< funcdef def="IllegalUri" >}}
Candidate URI is unsupported or has illegal format. Status code is `-7`.
{{< /funcdef >}}

{{< funcdef def="RuntimeException" >}}
Evaluation of script failed. Status code is `-5`.
{{< /funcdef >}}

{{< funcdef def="TooManyHops" >}}
Candidate URL was too many hops away from seed.
Status code is `-4001`.
{{< /funcdef >}}

{{< funcdef def="TooManyTransitiveHops" >}}
The URI is too many embed/transitive hops away from the last URI in scope.
Status code is `-4002`.
{{< /funcdef >}}
