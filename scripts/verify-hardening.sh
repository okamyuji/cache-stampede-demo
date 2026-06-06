#!/usr/bin/env bash
# verify-hardening.sh
# ビルドとテストが通ることをローカルで確認する。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

YELLOW='\033[33m'
GREEN='\033[32m'
RED='\033[31m'
NC='\033[0m'

pass=0
fail=0

check() {
  local name="$1"
  shift
  printf "${YELLOW}>>> %s${NC}\n" "$name"
  if "$@"; then
    printf "${GREEN}    PASS${NC}\n"
    pass=$((pass+1))
  else
    printf "${RED}    FAIL${NC}\n"
    fail=$((fail+1))
  fi
}

command -v go >/dev/null 2>&1 || { echo "go not found"; exit 2; }

WORK="$ROOT/.hardening-tmp"
rm -rf "$WORK"
mkdir -p "$WORK"
trap 'rm -rf "$WORK"' EXIT

check "unit tests (go test -race ./...)" bash -c "go test -race -count=1 ./... >/dev/null"
check "build stampede-server binary" bash -c "go build -o '$WORK/stampede-server' ./cmd/server"

echo
echo "----------------------------------------"
printf "passed: ${GREEN}%d${NC}  failed: ${RED}%d${NC}\n" "$pass" "$fail"
echo "----------------------------------------"
if [ "$fail" -gt 0 ]; then
  exit 1
fi
