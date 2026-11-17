#!/usr/bin/env bash

set -euo pipefail

expected_artifacts=(
  fritz-mcp-linux-amd64
  fritz-mcp-linux-arm64
  fritz-mcp-linux-386
  fritz-mcp-linux-arm
  fritz-mcp-darwin-amd64
  fritz-mcp-darwin-arm64
  fritz-mcp-windows-amd64.exe
  fritz-mcp-windows-arm64.exe
  fritz-mcp-wasi.wasm
)

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

validate_manifest() {
  local forge=$1
  local manifest=$2
  local artifact
  local actual_names
  local expected_names

  [[ -f $manifest ]] || fail "$forge manifest not found: $manifest"

  actual_names=$(awk '{print $2}' "$manifest" | sed 's#^\*##' | LC_ALL=C sort)
  expected_names=$(printf '%s\n' "${expected_artifacts[@]}" | LC_ALL=C sort)

  if [[ $actual_names != "$expected_names" ]]; then
    printf '%s\n' "--- expected artifacts" "+++ $forge artifacts" >&2
    diff -u <(printf '%s\n' "$expected_names") <(printf '%s\n' "$actual_names") >&2 || true
    fail "$forge SHA256SUMS does not describe the canonical artifact set"
  fi

  while read -r _ artifact; do
    [[ -n $artifact ]] || fail "$forge SHA256SUMS contains an invalid line"
  done < "$manifest"
}

normalize_manifest() {
  awk '{print $1 "  " $2}' "$1" | LC_ALL=C sort -k2
}

compare_manifests() {
  local github_manifest=$1
  local gitlab_manifest=$2
  local azure_manifest=$3
  local failed=0

  validate_manifest GitHub "$github_manifest"
  validate_manifest GitLab "$gitlab_manifest"
  validate_manifest Azure "$azure_manifest"

  diff -u \
    <(normalize_manifest "$github_manifest") \
    <(normalize_manifest "$gitlab_manifest") || failed=1
  diff -u \
    <(normalize_manifest "$github_manifest") \
    <(normalize_manifest "$azure_manifest") || failed=1

  ((failed == 0)) || fail "release digests differ between forges"
  printf 'Three-forge reproducibility verified for %s\n' "${TAG:-provided manifests}"
}

if [[ $# -eq 3 ]]; then
  compare_manifests "$1" "$2" "$3"
  exit 0
fi

[[ $# -eq 0 ]] || fail "usage: $0 [GITHUB_SHA256SUMS GITLAB_SHA256SUMS AZURE_SHA256SUMS]"

: "${TAG:?set TAG to the release tag, for example v0.8.2}"
: "${AZURE_ORGANIZATION:?set AZURE_ORGANIZATION to the Azure DevOps organization name}"
: "${AZURE_PROJECT:?set AZURE_PROJECT to the Azure DevOps project name or ID}"
: "${AZURE_RUN_ID:?set AZURE_RUN_ID to the tagged Azure Pipeline run ID}"

GITHUB_REPO=${GITHUB_REPO:-kambriso/fritzbox-mcp-server}
GITLAB_REPO=${GITLAB_REPO:-kambriso/fritzbox-mcp-server}
AZURE_ARTIFACT=${AZURE_ARTIFACT:-release-artifacts}
VERIFY_DIR=${VERIFY_DIR:-build/verify/three-forge}

require_command az
require_command find
require_command gh
require_command glab

github_dir=$VERIFY_DIR/github
gitlab_dir=$VERIFY_DIR/gitlab
azure_dir=$VERIFY_DIR/azure

mkdir -p "$github_dir" "$gitlab_dir" "$azure_dir"
rm -f "$github_dir/SHA256SUMS" "$gitlab_dir/SHA256SUMS"

gh release download "$TAG" \
  --repo "$GITHUB_REPO" \
  --pattern SHA256SUMS \
  --dir "$github_dir" \
  --clobber

glab release download "$TAG" \
  --repo "$GITLAB_REPO" \
  --asset-name SHA256SUMS \
  --dir "$gitlab_dir"

az pipelines runs artifact download \
  --artifact-name "$AZURE_ARTIFACT" \
  --path "$azure_dir" \
  --run-id "$AZURE_RUN_ID" \
  --organization "https://dev.azure.com/$AZURE_ORGANIZATION/" \
  --project "$AZURE_PROJECT" \
  --only-show-errors

mapfile -t azure_manifests < <(find "$azure_dir" -type f -name SHA256SUMS -print)
[[ ${#azure_manifests[@]} -eq 1 ]] || \
  fail "expected one Azure SHA256SUMS, found ${#azure_manifests[@]} below $azure_dir"

compare_manifests \
  "$github_dir/SHA256SUMS" \
  "$gitlab_dir/SHA256SUMS" \
  "${azure_manifests[0]}"
