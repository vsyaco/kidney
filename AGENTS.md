# Kidney Agent Instructions

Project-specific guidance for automated coding agents working in this repository.

## Project Scope

Kidney is a local Kindle USB library manager written in Go. It exposes a local
browser UI and CLI for files already owned by the user. Do not add network
services, cloud sync, Amazon account integration, DRM handling, or telemetry
without an explicit product decision.

## Architecture

- Keep domain types in `internal/domain`.
- Keep Kindle detection and file transport details in `internal/transport`.
- Keep use-case orchestration in `internal/library`.
- Keep HTTP/API/UI serving in `internal/server`.
- Keep CLI wiring in `cmd/kidney`.

Transport implementations must satisfy `domain.Transport` and preserve path
safety. Do not allow absolute paths, `..`, empty path parts, or host filesystem
paths to cross into Kindle file operations.

## Upload Format Handling

EPUB upload is converted to AZW3 before sideloading because Kindle does not
reliably index raw EPUB files copied over USB/MTP. Calibre `ebook-convert` is
the only supported EPUB conversion runtime.

Supported passthrough formats are PDF, MOBI, AZW, AZW3, KFX, and TXT. Do not
send PDF through the conversion pipeline because Kindle reads PDF natively.

- Do not add converter selection to the CLI or web UI without an explicit
  product decision.
- Command resolution order is packaged
  `tools/calibre.app/Contents/MacOS/ebook-convert`, packaged
  `tools/calibre/ebook-convert`, packaged `tools/ebook-convert`, explicit
  `KIDNEY_EBOOK_CONVERT`, then `PATH`.
- Packaged macOS builds bundle a pruned Calibre runtime through
  `scripts/package-calibre-runtime-macos.sh`.
- Packaged Linux and Windows builds bundle the official Calibre runtime under
  `tools/calibre`.
- Packaged Linux builds bundle `libusb` through
  `scripts/package-libusb-runtime-linux.sh`.
- Packaged builds do not bundle Kindle Previewer or boko.
- If changing the pruned Calibre runtime allowlist, validate representative
  EPUB files and verify output on a real Kindle Paperwhite when possible.

## Licensing

This project source is GPL-3.0-or-later. Packaged builds that distribute
Calibre are GPL-3.0-only compatible. Keep `README.md`, `LICENSE`, and
`THIRD_PARTY.md` in sync when runtime dependencies or bundled tools change.

## Checks

Run before committing:

```bash
go test ./...
scripts/package-macos.sh
```

For packaging changes, also verify:

```bash
env PATH=/usr/bin:/bin:/usr/sbin:/sbin \
  dist/kidney-darwin-$(uname -m)/kidney upload <epub-file>
```

Use disposable files for Kindle upload checks and delete them after
verification. For format-handling changes, verify one EPUB conversion and at
least one passthrough upload such as PDF.
