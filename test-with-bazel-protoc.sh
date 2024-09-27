#!/bin/bash
set -eu pipefail

TMP_FOR_PROTOC="$PWD/tmp-for-protobuf-install"
mkdir -p "$TMP_FOR_PROTOC"

bazel build @com_google_protobuf//pkg:protoc_release
unzip -o bazel-bin/external/protobuf~/pkg/protoc-27.1-unknown.zip -d "$TMP_FOR_PROTOC"

export PROTOC_INC="$TMP_FOR_PROTOC/include"
export PROTOC="$TMP_FOR_PROTOC/bin/protoc"

npm run build
#npm test
