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

{{< funcdef def="isSameHost(includeSubdomains=False)" >}}
Returns a `True` [Match]({{< ref "types#match" >}}) value if the Candidate URL has the same domain as its seed.

If `includeSubdomains=True` then the Candidate URL might have a subdomain of the Seeds domain. 
{{< /funcdef >}}

{{< funcdef def="maxHopsFromSeed(hops, includeRedirects=False)" >}}
{{< /funcdef >}}

{{< funcdef def="isUrl(url)" >}}
Space separated string with urls
```
isUrl("http://example.com")
```
{{< /funcdef >}}
