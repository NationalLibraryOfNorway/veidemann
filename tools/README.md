# Go build tools

This module keeps locally executed Go build tools and their dependency graphs
separate from the application modules. It currently pins the Buf CLI used to
generate the Go API. The API invokes it from
[`api/generate.go`](../api/generate.go):

```text
go tool -modfile=../tools/go.mod buf generate --template ../buf.gen.go.yaml ..
```

## Updating tools

From the repository root, update every tool declared in `tools/go.mod` to its
latest version:

```bash
./hack/update-go-tools.sh
```

To update one tool to a specific version, provide its full package path:

```bash
./hack/update-go-tools.sh github.com/bufbuild/buf/cmd/buf@v1.73.0
```

The equivalent direct Go commands for Buf are:

```bash
GOWORK=off go -C tools get -tool github.com/bufbuild/buf/cmd/buf@v1.73.0
GOWORK=off go -C tools mod tidy
```

Do not edit the indirect requirements by hand. They are the dependency graph
of the declared tools and are maintained by `go get` and `go mod tidy`.

The update helper changes only `tools/go.mod` and `tools/go.sum`; it does not
update Buf's remote code-generation plugins. For convenience, it queries their
upstream Go modules and reports each configured version alongside the latest
available version. After a local tool update, regenerate the Go API and review
all resulting changes before committing:

```bash
cd api
go generate ./...
cd ..
git diff -- tools api
```

## Updating remote plugins

The `protoc-gen-go` and `protoc-gen-go-grpc` plugins run remotely through Buf.
Their versions are pinned in [`buf.gen.go.yaml`](../buf.gen.go.yaml), not in
this Go module. To update one, change the version suffix on its `remote` entry,
then regenerate and test the API:

```bash
cd api
go generate ./...
go test ./...
cd ..
git diff -- buf.gen.go.yaml api
```

The configuration intentionally pins upstream plugin versions but omits Buf
package revisions. This prevents upgrades to a different generator release
while allowing Buf to use a newer packaging revision of the selected release.
Pin an explicit `revision` in `buf.gen.go.yaml` as well if immutable plugin
packaging becomes necessary.
