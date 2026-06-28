#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CALIBRE_APP="${KIDNEY_CALIBRE_APP:-/Applications/calibre.app}"
DIST_DIR="${1:-${ROOT_DIR}/dist/kidney-darwin-$(uname -m)}"
TOOLS_DIR="${DIST_DIR}/tools"
RUNTIME_APP="${TOOLS_DIR}/calibre.app"

if [[ ! -x "${CALIBRE_APP}/Contents/MacOS/ebook-convert" ]]; then
  echo "Calibre ebook-convert not found. Install build dependency: brew install --cask calibre" >&2
  echo "Or set KIDNEY_CALIBRE_APP=/path/to/calibre.app" >&2
  exit 1
fi

SRC="${CALIBRE_APP}/Contents"
DST="${RUNTIME_APP}/Contents"

copy_item() {
  local relative_path="$1"
  local source_path="${SRC}/${relative_path}"
  local target_path="${DST}/${relative_path}"

  if [[ ! -e "${source_path}" ]]; then
    echo "Required Calibre runtime item is missing: ${source_path}" >&2
    exit 1
  fi

  mkdir -p "$(dirname "${target_path}")"
  ditto "${source_path}" "${target_path}"
}

copy_if_exists() {
  local relative_path="$1"
  local source_path="${SRC}/${relative_path}"

  if [[ -e "${source_path}" ]]; then
    copy_item "${relative_path}"
  fi
}

make_symlink() {
  local link_path="${DST}/$1"
  local target="$2"

  rm -rf "${link_path}"
  mkdir -p "$(dirname "${link_path}")"
  ln -s "${target}" "${link_path}"
}

rm -rf "${RUNTIME_APP}"
mkdir -p "${DST}/MacOS" "${DST}/Frameworks/plugins" "${DST}/Resources"

copy_item "MacOS/ebook-convert"
copy_item "MacOS/calibre-parallel"
copy_if_exists "Info.plist"

frameworks=(
  "Python.framework"
  "QtCore.framework"
  "QtDBus.framework"
  "QtGui.framework"
  "QtWidgets.framework"
)

dylibs=(
  "calibre-launcher.dylib"
  "libcrypto.3.dylib"
  "libexpat.1.dylib"
  "libexslt.0.dylib"
  "libfreetype.6.dylib"
  "libiconv.2.dylib"
  "libicudata.78.dylib"
  "libicui18n.78.dylib"
  "libicuio.78.dylib"
  "libicuuc.78.dylib"
  "libjbig.2.1.dylib"
  "libjpeg.8.dylib"
  "liblzma.5.dylib"
  "libopenjp2.7.dylib"
  "libpng16.16.dylib"
  "libssl.3.dylib"
  "libtiff.6.dylib"
  "libwebp.7.dylib"
  "libxml2.16.dylib"
  "libxslt.1.dylib"
  "libz.1.dylib"
  "libzstd.1.dylib"
)

plugins=(
  "PIL._imaging.so"
  "PyQt6.QtCore.so"
  "PyQt6.QtGui.so"
  "PyQt6.QtWidgets.so"
  "PyQt6.sip.so"
  "_bisect.so"
  "_blake2.so"
  "_bz2.so"
  "_ctypes.so"
  "_elementtree.so"
  "_hashlib.so"
  "_heapq.so"
  "_interpreters.so"
  "_json.so"
  "_lzma.so"
  "_multiprocessing.so"
  "_pickle.so"
  "_posixsubprocess.so"
  "_queue.so"
  "_random.so"
  "_scproxy.so"
  "_socket.so"
  "_ssl.so"
  "_struct.so"
  "_uuid.so"
  "_zoneinfo.so"
  "_zstd.so"
  "array.so"
  "binascii.so"
  "cPalmdoc.so"
  "fast_html_entities.so"
  "fcntl.so"
  "grp.so"
  "html5_parser.html_parser.so"
  "icu.so"
  "imageops.so"
  "lxml._elementpath.so"
  "lxml.builder.so"
  "lxml.etree.so"
  "math.so"
  "msgpack._cmsgpack.so"
  "pyexpat.so"
  "python-lib.bypy.frozen"
  "regex._regex.so"
  "resource.so"
  "select.so"
  "speedup.so"
  "tokenizer.so"
  "translator.so"
  "unicodedata.so"
  "usbobserver.so"
  "zlib.so"
)

for item in "${frameworks[@]}"; do
  copy_item "Frameworks/${item}"
done

for item in "${dylibs[@]}"; do
  copy_item "Frameworks/${item}"
done

for item in "${plugins[@]}"; do
  copy_item "Frameworks/plugins/${item}"
done

copy_item "Resources/resources"

mkdir -p "${DST}/ebook-viewer.app/Contents/ebook-edit.app/Contents/headless.app/Contents/MacOS"
copy_if_exists "ebook-viewer.app/Contents/Info.plist"
copy_if_exists "ebook-viewer.app/Contents/ebook-edit.app/Contents/Info.plist"
copy_if_exists "ebook-viewer.app/Contents/ebook-edit.app/Contents/headless.app/Contents/Info.plist"
copy_item "ebook-viewer.app/Contents/ebook-edit.app/Contents/headless.app/Contents/MacOS/calibre-parallel"

make_symlink "ebook-viewer.app/Contents/Frameworks" "../../Frameworks"
make_symlink "ebook-viewer.app/Contents/Resources" "../../Resources"
make_symlink "ebook-viewer.app/Contents/PlugIns" "../../PlugIns"
make_symlink "ebook-viewer.app/Contents/ebook-edit.app/Contents/Frameworks" "../../../../Frameworks"
make_symlink "ebook-viewer.app/Contents/ebook-edit.app/Contents/Resources" "../../../../Resources"
make_symlink "ebook-viewer.app/Contents/ebook-edit.app/Contents/PlugIns" "../../../../PlugIns"
make_symlink "ebook-viewer.app/Contents/ebook-edit.app/Contents/headless.app/Contents/Frameworks" "../../../../../../Frameworks"
make_symlink "ebook-viewer.app/Contents/ebook-edit.app/Contents/headless.app/Contents/Resources" "../../../../../../Resources"
make_symlink "ebook-viewer.app/Contents/ebook-edit.app/Contents/headless.app/Contents/PlugIns" "../../../../../../PlugIns"

validate_dir="$(mktemp -d "${TMPDIR:-/tmp}/kidney-calibre-runtime-validate-XXXXXX")"
cleanup() {
  rm -rf "${validate_dir}"
}
trap cleanup EXIT

mkdir -p "${validate_dir}/epub/META-INF" "${validate_dir}/epub/OEBPS/images"
printf 'application/epub+zip' > "${validate_dir}/epub/mimetype"
cat > "${validate_dir}/epub/META-INF/container.xml" <<'XML'
<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>
XML
cat > "${validate_dir}/epub/OEBPS/content.opf" <<'XML'
<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" unique-identifier="bookid" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:identifier id="bookid">kidney-calibre-runtime-validation</dc:identifier>
    <dc:title>Kidney Calibre Runtime Validation</dc:title>
    <dc:language>ru</dc:language>
  </metadata>
  <manifest>
    <item id="chapter" href="chapter.xhtml" media-type="application/xhtml+xml"/>
    <item id="css" href="styles.css" media-type="text/css"/>
    <item id="png" href="images/pixel.png" media-type="image/png"/>
  </manifest>
  <spine>
    <itemref idref="chapter"/>
  </spine>
</package>
XML
cat > "${validate_dir}/epub/OEBPS/styles.css" <<'CSS'
body { font-family: serif; line-height: 1.45; margin: 1em; }
h1 { font-size: 1.8em; }
.note { border-left: 3px solid #777; padding-left: .5em; }
CSS
cat > "${validate_dir}/epub/OEBPS/chapter.xhtml" <<'XML'
<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
  <head>
    <title>Проверка Calibre</title>
    <link rel="stylesheet" type="text/css" href="styles.css"/>
  </head>
  <body>
    <h1>Кириллица, CSS и изображение</h1>
    <p class="note">Проверка bundled Calibre runtime.</p>
    <p><img src="images/pixel.png" alt="pixel"/></p>
  </body>
</html>
XML
base64 -D > "${validate_dir}/epub/OEBPS/images/pixel.png" <<'BASE64'
iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAFgwJ/lK3uCgAAAABJRU5ErkJggg==
BASE64

(
  cd "${validate_dir}/epub"
  zip -X0 "${validate_dir}/input.epub" mimetype >/dev/null
  zip -Xur9D "${validate_dir}/input.epub" META-INF OEBPS >/dev/null
)

"${RUNTIME_APP}/Contents/MacOS/ebook-convert" \
  "${validate_dir}/input.epub" \
  "${validate_dir}/output.azw3" \
  >/dev/null

if [[ ! -s "${validate_dir}/output.azw3" ]]; then
  echo "Packaged Calibre runtime validation failed: output.azw3 was not created" >&2
  exit 1
fi

echo "Packaged Calibre runtime at ${RUNTIME_APP}"
du -sh "${RUNTIME_APP}"
