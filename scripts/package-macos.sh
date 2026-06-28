#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist/kidney-darwin-$(uname -m)"
BIN_PATH="${DIST_DIR}/kidney"
LIB_DIR="${DIST_DIR}/lib"

mkdir -p "${LIB_DIR}"

go build -o "${BIN_PATH}" "${ROOT_DIR}/cmd/kidney"

LIBUSB_PATH="$(otool -L "${BIN_PATH}" | awk '/libusb-1.0.0.dylib/ {print $1; exit}')"
if [[ ! -f "${LIBUSB_PATH}" ]]; then
  echo "libusb dylib not found. Install build dependency: brew install libusb" >&2
  exit 1
fi

cp "${LIBUSB_PATH}" "${LIB_DIR}/libusb-1.0.0.dylib"
chmod 755 "${LIB_DIR}/libusb-1.0.0.dylib"

install_name_tool \
  -change "${LIBUSB_PATH}" "@executable_path/lib/libusb-1.0.0.dylib" \
  "${BIN_PATH}"

echo "Packaged ${BIN_PATH}"
otool -L "${BIN_PATH}"
