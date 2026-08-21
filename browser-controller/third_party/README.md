# Third-party source

## chromedp

`chromedp/` is a patched Git subtree of
<https://github.com/chromedp/chromedp>. It was imported from tag `v0.16.0`,
upstream commit `7963c203`, and remains licensed under the MIT license in
`chromedp/LICENSE`.

Browser-controller selects this copy with a local `replace` directive in its
`go.mod`. The public module and import path remain
`github.com/chromedp/chromedp`.

### Downstream patches

- Serialize DOM event access to `cdp.Frame.Nodes` with document rebuilds. This
  prevents `fatal error: concurrent map read and map write` in
  `chromedp.(*Target).domEvent` when an attached target is initialized while
  Chrome is delivering DOM events.

Do not copy `cdproto` or edit its generated protocol code here. It remains a
normal Go module dependency selected by browser-controller's module graph.

### Updating

Review upstream changes and select a release tag deliberately. From the
repository root, import it as a squash merge:

```sh
git subtree pull \
  --prefix=browser-controller/third_party/chromedp \
  https://github.com/chromedp/chromedp.git \
  <tag> \
  --squash
```

Resolve any conflict with the downstream patch, update the tag and commit
recorded above, and verify both modules:

```sh
cd browser-controller/third_party/chromedp
go test ./...
go test -race -run '^TestDOMEvent' .

cd ../..
go test ./...
docker build .
```
