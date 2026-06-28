#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${1:-${ROOT_DIR}/dist/kidney-linux-$(uname -m)}"
BIN_PATH="${DIST_DIR}/kidney"
LIB_DIR="${DIST_DIR}/lib"

if [[ ! -x "${BIN_PATH}" ]]; then
  echo "Kidney binary not found at ${BIN_PATH}" >&2
  exit 1
fi

if ! command -v patchelf >/dev/null 2>&1; then
  echo "patchelf is required to package Linux libusb runtime" >&2
  exit 1
fi

LIBUSB_PATH="$(ldd "${BIN_PATH}" | awk '/libusb-1.0.so/ {print $3; exit}')"
if [[ -z "${LIBUSB_PATH}" || ! -f "${LIBUSB_PATH}" ]]; then
  echo "libusb shared library not found in ${BIN_PATH} dependencies" >&2
  exit 1
fi

mkdir -p "${LIB_DIR}"
cp "${LIBUSB_PATH}" "${LIB_DIR}/"
patchelf --set-rpath '$ORIGIN/lib' "${BIN_PATH}"

echo "Packaged ${LIBUSB_PATH} into ${LIB_DIR}"
ldd "${BIN_PATH}"
