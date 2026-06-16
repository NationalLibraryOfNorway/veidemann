# veidemann-browser-controller

## Build container

Need to use the context of the monorepo root when building the container:

    docker build -f Dockerfile ..

## Test

    go test ./...

## Browser scripts

Runtime-invoked browser scripts may return either their final JSON value or a Promise resolving to that value.

- `ON_LOAD` scripts are executed via the Chrome runtime `CallFunctionOn` API and may resolve asynchronously to the usual `{waitForData, next, data}` object.
- `EXTRACT_OUTLINKS` scripts are executed via the Chrome runtime `Evaluate` API and may resolve asynchronously to the usual `[]string` outlink list.
- Plain functions that return `Promise.resolve(...)` are supported in addition to `async` functions.

`ON_NEW_DOCUMENT` scripts are different: they are injected with `page.AddScriptToEvaluateOnNewDocument` and run fire-and-forget, so their return values are not observed by browser-controller.

## Run integration test

The integration test is useful to validate compatibility between the `browserless` container image and the `chromedp` library.

### Using docker

    go test --tags=integration -run TestSession_Fetch ./server

### Using podman

> Note that you have to manually clean up containers after every run with using podman as container provider because the container reaper (ryuk) is disabled.

    # Setup podman socket
    systemctl --user start podman

    # Run integration test
    DOCKER_HOST="unix:///run/user/$UID/podman/podman.sock" go test -run TestSession_Fetch --tags=integration ./server -provider=podman
