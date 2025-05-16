---
title: "Types"
date: 2021-03-17T10:07:20+01:00
---
### UrlValue
{{< funcdef def="urlValue.host()" >}}
Returns a string with the host part of the Url
{{< /funcdef >}}

{{< funcdef def="urlValue.port()" >}}
Returns a string with the port part of the Url
{{< /funcdef >}}

### Match
All built in matching functions returns a `Match` object. The `Match` object has the value `True` or `False` and can be
used everywhere a boolean is expected. The difference is that the `Match` object has a few convenient built in methods
to make scripts more compact.

To turn a boolean into a `Match` object, you can use the [Test()]({{< ref "matchers#testmatchfalse" >}}) method

{{< funcdef def="match.then(status, continueEvaluation=False)" >}}
Sets the submitted [Status]({{< ref "constants#status" >}}) as the script response if match is `True`.
If `continueEvaluation` (optional parameter) is `True`, then the script evaluation continues.

Typical use:
```
isScheme("mailto").then(Blocked)
```

Which is equivalent to:
```
if isScheme("mailto"):
    setStatus(Blocked)
    abort()
```
`match.then()` returns the same `match` object so that it can be chained with otherwise to form an if-then-else expression.
```
isScheme("mailto").then(Blocked).otherwise(Include, continueEvaluation=True)
```

Which is equivalent to:
```
if isScheme("mailto"):
    setStatus(Blocked)
    abort()
else:
    setStatus(Include)
```
{{< /funcdef >}}

{{< funcdef def="match.otherwise(status, continueEvaluation=False)" >}}
Sets the script response status if match is `False`. If continueEvaluation is False, then the script returns.

Typical use:
```
isScheme('http').otherwise(Blocked)
```

Which is equivalent to:
```
if not isScheme("http"):
    setStatus(Blocked)
    abort()
```
{{< /funcdef >}}
