---
title: "Matchers"
date: 2021-03-17T15:34:11+01:00
---

{{< funcdef def="test(match=False)" >}}
Returns a [Match]({{< ref "types#match" >}}) object with the same True/False value.
{{< /funcdef >}}

{{< funcdef def="isScheme(scheme)" >}}
Takes a space separated string of schemes and checks if the Uri candidate has a scheme matching one of them.
Returns a `True` [Match]({{< ref "types#match" >}}) value if the URI has the submitted scheme.
{{< /funcdef >}}

{{< funcdef def="isReferrer(referrer)" >}}
Space separated string with referrer urls
{{< /funcdef >}}

{{< funcdef def="isSameHost(includeSubdomains=False, altSeeds='')" >}}
Returns a `True` [Match]({{< ref "types#match" >}}) value if the Candidate URL has the same canonical host as its seed or one of the space-separated `altSeeds`.

If `includeSubdomains=True`, the candidate may also use the seed host's scope anchor or one of its subdomains. One leading label matching `www[0-9]*` is removed from each seed host to derive that anchor, provided the remainder contains a registrable domain. This makes `example.no`, `www.example.no`, and `www2.example.no` part of the same host scope while preventing `www.co.uk` from widening scope to the `co.uk` public suffix.

Only one leading `www` or numbered-`www` label is treated as an alias. The matcher does not promote arbitrary subdomains to their registrable domain, resolve DNS aliases, derive scope from redirects, or modify stored URIs. Pass `includeSubdomains=False` to retain exact-host matching.
{{< /funcdef >}}

{{< funcdef def="maxHopsFromSeed(hops, includeRedirects=False)" >}}
{{< /funcdef >}}

{{< funcdef def="isUrl(url)" >}}
Space separated string with urls
```
isUrl("http://example.com")
```
{{< /funcdef >}}

{{< funcdef def="isPathPrefix(prefix)" >}}
Takes a space separated string of path prefixes and checks if the candidate URI path starts with one of them.
The path compared is the canonicalized path.

```
isPathPrefix("/api/ /static/")
```
{{< /funcdef >}}
