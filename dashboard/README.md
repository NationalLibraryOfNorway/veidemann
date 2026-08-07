# Veidemann Dashboard

Veidemann Dashboard is the Veidemann UI.

## Runtime configuration

The dashboard reads `public/config.json` at startup. In addition to the API and
authentication settings, an environment can configure WARC playback and links
for selected labels:

```json
{
  "playbackBaseUrl": "https://pywb.example.org/pywb",
  "labelLinks": {
    "organisasjonsnummer": {
      "urlTemplate": "https://virksomhet.brreg.no/nb/oppslag/enheter/{value}",
      "text": "Brønnøysundregistrene"
    }
  }
}
```

`playbackBaseUrl` may be a root-relative path or an absolute HTTP(S) URL. When a
crawl execution has finished, been aborted, or failed, its Playback chip opens
the closest capture to the crawl start time in a new tab. An empty value hides
the chip.

`labelLinks` maps an exact, case-sensitive label key to an external link. The
URL template must be an absolute HTTP(S) URL containing `{value}`. The dashboard
replaces every placeholder with the percent-encoded label value. Invalid entries
are ignored, and the original label chip continues to search the configuration
list.

## Hosting pywb on a separate origin

A separate origin is recommended because archived pages can contain untrusted
JavaScript. Give pywb dedicated DNS and TLS, keep dashboard cookies and other
credentials scoped away from that origin, and apply authentication or pywb
access controls when the archived content must not be public.

Set `playbackBaseUrl` to the absolute HTTPS URL. The public path must agree in
all three places: the dashboard setting, the ingress path, and pywb's uWSGI
mount. For example, `/pywb` requires a mount such as
`UWSGI_MOUNT=/pywb=/pywb/pywb/apps/wayback.py` with
`UWSGI_MANAGE_SCRIPT_NAME=true`.

Ingress should proxy ordinary HTTP to the pywb container on port 8080. It does
not need to support the uWSGI protocol or expose port 8081. CORS, iframe CSP
directives, and `X-Frame-Options` changes are unnecessary because Playback is a
normal top-level navigation. Verify that redirects and rewritten asset URLs use
the external HTTPS scheme and configured path before enabling the link.

## License

[Apache License 2.0](https://github.com/nlnwa/veidemann-dashboard/blob/master/LICENSE)
