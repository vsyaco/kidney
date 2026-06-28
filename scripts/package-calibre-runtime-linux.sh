#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CALIBRE_DIR="${KIDNEY_CALIBRE_DIR:-/opt/calibre}"
DIST_DIR="${1:-${ROOT_DIR}/dist/kidney-linux-$(uname -m)}"
TOOLS_DIR="${DIST_DIR}/tools"
RUNTIME_DIR="${TOOLS_DIR}/calibre"

if [[ ! -x "${CALIBRE_DIR}/ebook-convert" ]]; then
  echo "Calibre ebook-convert not found at ${CALIBRE_DIR}/ebook-convert" >&2
  echo "Install the official Calibre Linux binary runtime or set KIDNEY_CALIBRE_DIR." >&2
  exit 1
fi

rm -rf "${RUNTIME_DIR}" "${TOOLS_DIR}/ebook-convert"
mkdir -p "${TOOLS_DIR}"
cp -a "${CALIBRE_DIR}" "${RUNTIME_DIR}"
ln -s "calibre/ebook-convert" "${TOOLS_DIR}/ebook-convert"

"${RUNTIME_DIR}/ebook-convert" --version
echo "Packaged Calibre runtime at ${RUNTIME_DIR}"
du -sh "${RUNTIME_DIR}"
