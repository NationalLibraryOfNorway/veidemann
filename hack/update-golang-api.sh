#!/bin/bash

set -e

API_MODULE="github.com/NationalLibraryOfNorway/veidemann/api"
API_VERSION="v1.2.3"

find . -name go.mod -not -path './api/go.mod' -print0 |
while IFS= read -r -d '' modfile; do
  dir="$(dirname "$modfile")"

  if (cd "$dir" && go list -m "$API_MODULE" >/dev/null 2>&1); then
    echo "Updating $dir"
    (
      cd "$dir"
      # GOWORK=off go get "$API_MODULE@$API_VERSION"
      # GOWORK=off go mod tidy
    )
  fi
done