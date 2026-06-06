#!/usr/bin/env bash
# compare.sh
# 4つの戦略を順にk6で負荷テストし結果を比較する
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
STRATEGIES=("naive" "mutex" "probabilistic" "warmup")

command -v k6 >/dev/null 2>&1 || { echo "k6が見つかりません。brew install k6でインストールしてください"; exit 1; }

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "========================================="
echo " Cache Stampede Strategy Comparison"
echo "========================================="
echo ""

for strategy in "${STRATEGIES[@]}"; do
  echo "--- ${strategy} ---"
  k6 run \
    --env "STRATEGY=${strategy}" \
    --env "BASE_URL=${BASE_URL}" \
    --quiet \
    "${SCRIPT_DIR}/stampede.js"
  echo ""
  sleep 2
done

echo "========================================="
echo " Complete"
echo "========================================="
