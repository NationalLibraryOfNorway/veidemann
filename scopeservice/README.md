# veidemann-scopeservice

Service for deciding if URIs are in scope for crawling

## Overview

`scopeservice` has two main responsibilities:

- run scope-check scripts for candidate URIs
- canonicalize URIs in the same way across the harvester

The implementation is split into a few small packages:

- `pkg/server`: gRPC service layer for scope checking and URI canonicalization
- `pkg/script`: the Starlark-based scope DSL, built-in matchers, and canonicalization profiles
- `pkg/telemetry`: Prometheus metrics and metrics server wiring
- `pkg/logger`: local logging setup

## Scope Scripts

Most scope behavior is expressed in the Starlark DSL in `pkg/script`.
Typical rules include:

- allowed schemes
- same-host or subdomain checks
- excluded exact URLs
- excluded path prefixes
- referrer-based rules
- maximum hop count from seed

The current default scope script is:

```python
isScheme(param('scope_allowedSchemes')).otherwise(Blocked)
isSameHost(param('scope_includeSubdomains'), altSeeds=param('scope_altSeeds')).then(Include, continueEvaluation=True).otherwise(Blocked, continueEvaluation=False)
maxHopsFromSeed(param('scope_maxHopsFromSeed'), param('scope_hopsIncludeRedirects')).then(TooManyHops)
isUrl(param('scope_excludedUris')).then(Blocked)
```

For path-based rules, the normal use case is prefix matching rather than exact URL matching. The DSL now supports that directly:

```python
isScheme(param('scope_allowedSchemes')).otherwise(Blocked)
isSameHost(param('scope_includeSubdomains'), altSeeds=param('scope_altSeeds')).then(Include, continueEvaluation=True).otherwise(Blocked, continueEvaluation=False)
maxHopsFromSeed(param('scope_maxHopsFromSeed'), param('scope_hopsIncludeRedirects')).then(TooManyHops)
isPathPrefix(param('scope_excludedPathPrefixes')).then(Blocked)
isUrl(param('scope_excludedUris')).then(Blocked)
```

For more custom path logic, scripts can also inspect the canonicalized path directly:

```python
test(url().path().startswith('/api/')).then(Blocked)
```

Both forms operate on the canonicalized path, so matching happens after URI normalization.
