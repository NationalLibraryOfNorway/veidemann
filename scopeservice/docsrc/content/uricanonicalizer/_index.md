---
title: "URI canonicalizer"
date: 2021-03-17T10:15:16+01:00
---
The URI canonicalizer is a service which parses a URI, does some normalization and returns a structured object similar
to the URL object in browsers. The motivation for having this as a service is to ensure that URIs are parsed and
normalized in the same way independently of programming language, configuration and so on.

## Canoncalization
The canoncalization is (for now) configured in code and tries to not change the URIs semantics. Examples of canoncalization
are:
* Remove port numbers for well known schemes (i.e. `http://example.com:80` → `http://example.com`)
* Normalize slash for empty path (i.e. `http://example.com` → `http://example.com/`)
* Normalize path (i.e. `http://example.com/a//b/./c` → `http://example.com/a/b/c`)
* Remove user info (i.e. `http://user:passwd@example.com` → `http://example.com/`)
* Sort query (i.e. `http://example.com/foo?b=2&a=3&c=4&b=1/` → `http://example.com/foo?a=3&b=2&b=1&c=4`)
  {{% notice note %}}
  Only the query parameter names are sorted. This is less likely to alter semantics than also sort values.
  {{% /notice %}}

## API
The API is implemented as a [gRPC service](https://github.com/nlnwa/veidemann-api/blob/master/protobuf/uricanonicalizer/v1/uricanonicalizer.proto).

The request is the URI a as string.

The respones is a ParsedUri object defined as follows:

```protobuf
message ParsedUri {
    // The entire uri
    string href = 1;
    // The scheme (protocol) part of the uri
    string scheme = 2;
    // The hostname of the uri
    string host = 3;
    // The port number of the uri
    int32 port = 4;
    // The username part of the uri
    string username = 5;
    // The password part of the uri
    string password = 6;
    // The path part of the uri
    string path = 7;
    // The query (search) part of the uri
    string query = 8;
    // The fragment (hash) part of the uri
    string fragment = 9;
}
```
