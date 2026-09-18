#!/usr/bin/env bash
set -Eeuo pipefail

PROJECT_ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="${HOME}/.local/bin"
TARGET="${INSTALL_DIR}/historic"
TEMP_BINARY="$(mktemp "${TMPDIR:-/tmp}/historic.XXXXXX")"

cleanup() {
    rm -f -- "${TEMP_BINARY}"
}
trap cleanup EXIT

mkdir -p -- "${INSTALL_DIR}"

cd -- "${PROJECT_ROOT}"
go build -o "${TEMP_BINARY}" .
install -m 0755 -- "${TEMP_BINARY}" "${TARGET}"

if ! command -v historic >/dev/null 2>&1; then
    echo "historic was installed at ${TARGET}, but it is not available on PATH" >&2
    exit 1
fi

printf 'Installed %s\n' "$(command -v historic)"
historic version
