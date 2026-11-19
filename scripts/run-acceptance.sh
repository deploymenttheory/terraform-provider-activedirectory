#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
: "${TF_ACC:?Set TF_ACC=1 explicitly}"
: "${AD_ACCEPTANCE_AUTHORIZED:?Set AD_ACCEPTANCE_AUTHORIZED=1 explicitly}"
: "${AD_HOST:?Select the lab endpoint}"
: "${AD_ACCEPTANCE_HOSTS:?Allowlist the selected lab endpoint}"
: "${AD_ACCEPTANCE_PARTITION:?Select the lab naming context}"
: "${AD_ACCEPTANCE_SERVER_VERSION:?Select the lab server version}"
: "${AD_ACCEPTANCE_RUN_ID:?Select an externally unique fixture run}"
if [[ "$TF_ACC" != 1 || "$AD_ACCEPTANCE_AUTHORIZED" != 1 || "${GITHUB_EVENT_NAME:-}" == pull_request* ]]; then
  echo 'Live acceptance refused: authorization or runner policy is invalid' >&2
  exit 1
fi
if [[ ! "$AD_ACCEPTANCE_RUN_ID" =~ ^[a-z][a-z0-9-]{5,39}$ || ! "$AD_ACCEPTANCE_SERVER_VERSION" =~ ^(2016|2019|2022|2025)$ ]]; then
  echo 'Live acceptance refused: fixture identity or server version is invalid' >&2
  exit 1
fi
allowed=0
IFS=',' read -r -a hosts <<< "$AD_ACCEPTANCE_HOSTS"
for host in "${hosts[@]}"; do
  host="${host//[[:space:]]/}"
  if [[ "${host,,}" == "${AD_HOST,,}" ]]; then allowed=1; fi
done
if [[ "$allowed" != 1 || "$AD_HOST" == *'/'* || "$AD_HOST" == *'@'* ]]; then
  echo 'Live acceptance refused: endpoint is not the selected lab host' >&2
  exit 1
fi
export TF_ACC_AD_LAB=1
if [[ -n "${AD_ACCEPTANCE_TEST_BINARY:-}" ]]; then
  : "${AD_ACCEPTANCE_BINARY_SHA256:?Pin the acceptance test executable SHA256}"
  actual=$(sha256sum "$AD_ACCEPTANCE_TEST_BINARY")
  if [[ "${actual%% *}" != "$AD_ACCEPTANCE_BINARY_SHA256" ]]; then
    echo 'Acceptance binary checksum differs' >&2
    exit 1
  fi
  exec "$AD_ACCEPTANCE_TEST_BINARY" "-test.run=${2:-^TestAccProviderLabConnection$}" -test.count=1 -test.timeout=15m
fi
exec go test -mod=readonly "${1:-./internal/provider/core}" -run "${2:-^TestAccProviderLabConnection$}" -count=1 -timeout=15m
