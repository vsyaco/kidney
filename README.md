# Kidney

Local Kindle USB library manager.

Kidney runs as one local Go binary and serves a browser UI on `127.0.0.1`.
It manages files already present on your computer and copies them to the Kindle
documents folder. It does not download from Amazon, authenticate to Amazon, or
remove DRM.

## Features

- Detects Kindle devices over mounted disk volumes or direct MTP.
- Lists supported files recursively under the Kindle `documents` folder.
- Uploads, downloads, renames, and deletes files from a local web UI or CLI.
- Converts EPUB files to AZW3 with Calibre before direct USB/MTP upload so
  Kindle indexes them as local books.
- Packages macOS builds with the native MTP and EPUB conversion runtime.

## Install

From source:

```bash
git clone https://github.com/vsyaco/kidney.git
cd kidney
go run ./cmd/kidney serve
```

Build a local binary:

```bash
go build -o kidney ./cmd/kidney
./kidney serve
```

For a self-contained macOS package with bundled `libusb` and Calibre conversion
runtime:

```bash
scripts/package-macos.sh
dist/kidney-darwin-$(uname -m)/kidney serve
```

## Usage

```bash
go run ./cmd/kidney serve
go run ./cmd/kidney devices
go run ./cmd/kidney doctor
go run ./cmd/kidney list
go run ./cmd/kidney upload ./book.epub
go run ./cmd/kidney download book.epub
go run ./cmd/kidney rename book.epub renamed.epub
go run ./cmd/kidney delete renamed.epub
go run ./cmd/kidney unmount
```

The default UI URL is:

```text
http://127.0.0.1:8765
```

Use a different port when needed:

```bash
go run ./cmd/kidney serve -port 8799
```

## Requirements

Local development requires:

- Go 1.26 or newer.
- Homebrew `libusb` on macOS for the direct MTP backend.
- Calibre `ebook-convert` for EPUB conversion in unpackaged development builds.

Packaged macOS builds include `libusb` and a pruned Calibre runtime, so users do
not need to install Calibre separately. Development builds can install Calibre
with:

```bash
brew install --cask calibre
```

## Supported Files

- `.epub`
- `.pdf`
- `.mobi`
- `.azw`
- `.azw3`
- `.kfx`
- `.txt`

EPUB files are converted to `.azw3` before USB/MTP upload because Kindle indexes
Kindle-native files reliably when sideloaded directly. Kidney uses Calibre:

```bash
ebook-convert input.epub output.azw3
```

Kidney resolves `ebook-convert` in this order:

1. Packaged Calibre runtime,
   `tools/calibre.app/Contents/MacOS/ebook-convert`.
2. Packaged flat tool, `tools/ebook-convert`.
3. Explicit path override: `KIDNEY_EBOOK_CONVERT`.
4. `ebook-convert` on `PATH`.

There is no converter selection in the CLI or web UI. Calibre is the only EPUB
conversion runtime. Packaged macOS builds bundle a pruned Calibre runtime, not
the full Calibre app or Kindle Previewer.

Rename changes the file name on Kindle storage only. Kidney does not edit book
metadata in v1.

Kidney lists supported files recursively under the Kindle `documents` folder.
For nested files, CLI commands and API calls use the relative path shown by
`kidney list`, for example `SomeFolder/book.epub`.

## MTP Backend

Older Kindles can appear as mounted disk volumes. Newer Paperwhite devices often
use MTP instead of disk mounting.

Kidney uses direct MTP operations for those devices:

- No `simple-mtpfs`.
- No macFUSE.
- No mounted temporary filesystem.

The application package includes the native `libusb` runtime used by the MTP
backend and a pruned Calibre runtime used by EPUB conversion. Local development
can use Homebrew `libusb` and Calibre `ebook-convert`; packaged releases bundle
both runtime pieces instead of asking users to install them separately.

## License

Kidney source is licensed under GPL-3.0-or-later. Packaged builds bundle
Calibre, which is GPL-3.0-only, so packaged distributions are GPL-3.0-only
compatible. See [LICENSE](LICENSE) and [THIRD_PARTY.md](THIRD_PARTY.md) for
third-party notices.
