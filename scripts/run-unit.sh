#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
while IFS= read -r name; do
  if [[ "$name" == AD_* || "$name" == TF_ACC || "$name" == TF_ACC_AD_LAB ]]; then
    unset "$name"
  fi
done < <(compgen -e)
if [[ -z "${TF_ACC_TERRAFORM_PATH:-}" ]]; then
  export TF_ACC_TERRAFORM_VERSION=1.17.2
fi
exec go test -mod=readonly -count=1 ./...
