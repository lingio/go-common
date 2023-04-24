#!/bin/bash

set -euo pipefail

go install github.com/lingio/go-common/script/semvercomp@latest

CURRENT_VERSION=$(go list -m -json all | \
  jq -r 'select(.Path == "github.com/lingio/go-common") | .Version')

if semvercomp ${CURRENT_VERSION} "v1.14.0"; then
 	echo $(curl -s https://api.github.com/repos/lingio/oapi-codegen/releases/latest | jq -r .tag_name)
else
	echo "1.0.2"
fi
