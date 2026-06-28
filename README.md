# Kidney

Local Kindle USB library manager.

Kidney runs as one local Go binary and serves a browser UI on `127.0.0.1`.
It manages files already present on your computer and copies them to the Kindle
documents folder. It does not download from Amazon, authenticate to Amazon, or
remove DRM.

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

## Supported Files

- `.epub`
- `.pdf`
- `.mobi`
- `.azw`
- `.azw3`
- `.kfx`
- `.txt`

Rename changes the file name on Kindle storage only. Kidney does not edit book
metadata or convert formats in v1.

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

The application package must include the native `libusb` runtime used by the MTP
backend. Local development can use Homebrew `libusb`; packaged releases bundle
the matching `libusb` dylib/shared library with the app instead of asking users
to install it separately.

Build a self-contained macOS CLI package:

```bash
scripts/package-macos.sh
dist/kidney-darwin-$(uname -m)/kidney serve
```
