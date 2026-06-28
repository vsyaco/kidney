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

## EPUB Conversion

EPUB upload is converted to AZW3 before sideloading because Kindle does not
reliably index raw EPUB files copied over USB/MTP. Calibre `ebook-convert` is
the only supported EPUB conversion runtime.

- Do not add converter selection to the CLI or web UI without an explicit
  product decision.
- Command resolution order is packaged `tools/ebook-convert`, explicit
  `KIDNEY_EBOOK_CONVERT`, then `PATH`.
- Packaged builds do not bundle full `calibre.app`, Kindle Previewer, or boko.
- If bundling a pruned Calibre runtime later, verify output on a real Kindle
  Paperwhite and review license compatibility before changing packaging.

## Licensing

This project is GPL-3.0-or-later. Keep `README.md`, `LICENSE`, and
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
  KIDNEY_EBOOK_CONVERT=/opt/homebrew/bin/ebook-convert \
  dist/kidney-darwin-$(uname -m)/kidney upload <epub-file>
```

Use a disposable file for Kindle upload checks and delete it after verification.
