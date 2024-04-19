#!/usr/bin/env bash

set -euo pipefail

go install github.com/lingio/go-common/script/semvercomp@latest

CURRENT_VERSION=$(go list -m -json all | \
  jq -r 'select(.Path == "github.com/lingio/go-common") | .Version')

if semvercomp ${CURRENT_VERSION} "v1.17.0"; then
 	WANTED_VERSION=$(curl -s https://api.github.com/repos/lingio/go-common/releases/latest | jq -r .tag_name)
elif semvercomp ${CURRENT_VERSION} "v1.13.0"; then
 	WANTED_VERSION="v1.16.4"
else
	WANTED_VERSION="v1.12.4"
fi

if semvercomp ${WANTED_VERSION} ${CURRENT_VERSION}; then
  # only upgrade
  if [[ "${WANTED_VERSION}" != "${CURRENT_VERSION}" ]]; then
    >&2 echo "Upgrading go-common ${CURRENT_VERSION} --> ${WANTED_VERSION}"
    >&2 go get "github.com/lingio/go-common@${WANTED_VERSION}"
    >&2 go mod tidy
    >&2 git add go.mod go.sum
    >&2 git commit -m "autobump go-common to ${WANTED_VERSION}"
  fi
else
  WANTED_VERSION=${CURRENT_VERSION}
fi

echo $WANTED_VERSION
