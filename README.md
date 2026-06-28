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
- Converts EPUB files to AZW3 before direct USB/MTP upload so Kindle indexes
  them as local books.
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

For a self-contained macOS package with bundled `libusb` and `boko`:

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
- `boko` for EPUB conversion when running unpackaged development builds:
  `cargo install boko`.

Packaged macOS builds include `libusb` and `boko`, so users do not need to
install those tools separately.

## Supported Files

- `.epub`
- `.pdf`
- `.mobi`
- `.azw`
- `.azw3`
- `.kfx`
- `.txt`

EPUB files are converted to `.azw3` before USB/MTP upload because Kindle
indexes Kindle-native files reliably when sideloaded directly. The conversion
uses bundled `boko` in packaged macOS builds. Local development falls back to a
system `boko`; install the build dependency with `cargo install boko`.

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
backend and the `boko` binary used for EPUB conversion. Local development can use
Homebrew `libusb` and Cargo-installed `boko`; packaged releases bundle both with
the app instead of asking users to install them separately.

## License

Kidney is licensed under GPL-3.0-or-later. See [LICENSE](LICENSE).

Packaged builds bundle `boko`, which is also GPL-3.0-or-later. See
[THIRD_PARTY.md](THIRD_PARTY.md) for third-party notices.
