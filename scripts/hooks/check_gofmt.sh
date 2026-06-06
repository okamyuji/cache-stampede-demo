#!/usr/bin/env bash
set -euo pipefail

unformatted=$(gofmt -l . 2>&1 | grep -v vendor/ || true)
if [ -n "$unformatted" ]; then
  echo "ERROR: 以下のファイルがgofmtされていません:" >&2
  echo "$unformatted" >&2
  echo "  gofmt -w . を実行してください" >&2
  exit 2
fi
