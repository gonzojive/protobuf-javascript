#!/bin/bash

set -e  # Exit on non-zero exit code from any command in the script
set -o pipefail  # Exit on non-zero exit code from any command in a pipe

# Create directory with parents
mkdir -p /tmp/protobuf-javascript-release

# Remove existing files
rm /tmp/protobuf-javascript-release/*

# Force create git tag
git tag "$TAG" --force

# Force push tag
git push origin tag $TAG --force

# Archive code with prefix
git archive \
  --format zip \
  --output /tmp/protobuf-javascript-release/protobuf-javascript-$TAG.zip \
  --prefix "protobuf-javascript-$TAG/" \
  "$TAG"

# Create release on Github using 'gh' cli
gh --repo gonzojive/protobuf-javascript \
  release create \
    "$TAG" \
    /tmp/protobuf-javascript-release/protobuf-javascript-$TAG.zip \
  \
  --verify-tag \
  --title "$TAG" \
  --notes "experimental version with bzlmod support." \
  --draft \
  --prerelease
