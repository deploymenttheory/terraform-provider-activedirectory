#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
target_os=${1:?Select linux, windows or darwin}
target_arch=${2:?Select amd64 or arm64}
version=${3:-dev}
commit=${4:?Supply the source commit}
case "$target_os/$target_arch" in
  linux/amd64|linux/arm64|windows/amd64|windows/arm64|darwin/amd64|darwin/arm64) ;;
  *) echo 'Unsupported provider build target' >&2; exit 1 ;;
esac
if [[ ! "$version" =~ ^[a-zA-Z0-9.+-]+$ || ! "$commit" =~ ^[a-f0-9]{40}$ ]]; then
  echo 'Invalid build version or commit' >&2
  exit 1
fi
export GOOS="$target_os" GOARCH="$target_arch" CGO_ENABLED=0
suffix=''
if [[ "$target_os" == windows ]]; then suffix='.exe'; fi
output="bin/$target_os-$target_arch"
mkdir -p "$output"
verification=$(mktemp -d)
trap 'rm -f -- "$verification/provider$suffix"; rmdir -- "$verification"' EXIT
flags="-s -w -buildid= -X main.version=$version -X main.commit=$commit"
binary="$output/terraform-provider-activedirectory_v$version$suffix"
go build -mod=readonly -trimpath -buildvcs=false -ldflags "$flags" -o "$binary" .
go build -mod=readonly -trimpath -buildvcs=false -ldflags "$flags" -o "$verification/provider$suffix" .
cmp -- "$binary" "$verification/provider$suffix"
(cd "$output" && sha256sum "$(basename "$binary")" > SHA256SUMS)
echo "Verified reproducible binary: $target_os/$target_arch"
