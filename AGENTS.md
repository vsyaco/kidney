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
reliably index raw EPUB files copied over USB/MTP.

- Packaged builds use bundled `dist/.../tools/boko`.
- Development builds may use `KIDNEY_BOKO` or a `boko` binary on `PATH`.
- Do not reintroduce a bundled Calibre `.app`; it makes the package too large.
- If replacing `boko`, check the converter output on a real Kindle and review
license compatibility before changing packaging.

## Licensing

This project is GPL-3.0-or-later because packaged builds bundle `boko`, which is
GPL-3.0-or-later. Keep `README.md`, `LICENSE`, and `THIRD_PARTY.md` in sync when
runtime dependencies or bundled tools change.

## Checks

Run before committing:

```bash
go test ./...
scripts/package-macos.sh
```

For packaging changes, also verify:

```bash
env PATH=/usr/bin:/bin:/usr/sbin:/sbin dist/kidney-darwin-$(uname -m)/kidney upload <epub-file>
```

Use a disposable file for Kindle upload checks and delete it after verification.
