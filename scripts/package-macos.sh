#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist/kidney-darwin-$(uname -m)"
BIN_PATH="${DIST_DIR}/kidney"
LIB_DIR="${DIST_DIR}/lib"
TOOLS_DIR="${DIST_DIR}/tools"
BOKO_PATH="${TOOLS_DIR}/boko"

mkdir -p "${LIB_DIR}" "${TOOLS_DIR}"
rm -rf "${TOOLS_DIR}/calibre.app"

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

if command -v boko >/dev/null 2>&1; then
  cp "$(command -v boko)" "${BOKO_PATH}"
else
  if ! command -v cargo >/dev/null 2>&1; then
    echo "boko not found and cargo is unavailable. Install build dependency: cargo install boko" >&2
    exit 1
  fi

  BOKO_ROOT="${ROOT_DIR}/dist/.build-tools/boko"
  cargo install boko --root "${BOKO_ROOT}" --locked --force
  cp "${BOKO_ROOT}/bin/boko" "${BOKO_PATH}"
fi

chmod 755 "${BOKO_PATH}"
"${BOKO_PATH}" --version

echo "Packaged ${BIN_PATH}"
otool -L "${BIN_PATH}"
