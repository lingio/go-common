#!/usr/bin/env bash

set -euo pipefail

SERVICE_NAME=$(basename $(pwd))
OAPI_VERSION=$(curl -sL https://raw.githubusercontent.com/lingio/go-common/master/script/select-oapi-codegen.sh | bash)

>/dev/null pushd ../oapi-codegen
2>/dev/null git fetch
2>/dev/null git checkout ${OAPI_VERSION}
go install ./cmd/oapi-codegen
>/dev/null popd

>/dev/null pushd build
shaBefore=$(sha1sum ../restapi/spec.gen.go)
oapi-codegen -package models -service ${SERVICE_NAME} -generate types ../spec.yaml > ../models/model.gen.go
oapi-codegen -package restapi -service ${SERVICE_NAME} -generate spec ../spec.yaml > ../restapi/spec.gen.go
oapi-codegen -package restapi -service ${SERVICE_NAME} -generate server ../spec.yaml > ../restapi/server.gen.go
shaAfter=$(sha1sum ../restapi/spec.gen.go)
if [[ "$shaBefore" != "$shaAfter" ]]
then
  versionNumber=$(<version)
  newVersion=$((versionNumber+1))
  echo "$newVersion" > "version"
fi
>/dev/null popd
