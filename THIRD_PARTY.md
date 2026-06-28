# Third-Party Notices

Kidney depends on and may package third-party components. This file is a
human-readable summary; individual dependency source distributions remain the
authoritative license texts.

## Bundled In Packaged Builds

### Calibre

- Purpose: EPUB to AZW3 conversion through `ebook-convert`.
- Source: https://calibre-ebook.com/
- License: GPL-3.0-only.
- Packaging:
  - macOS: `scripts/package-calibre-runtime-macos.sh` copies a pruned Calibre
    runtime into `dist/kidney-darwin-*/tools/calibre.app`.
  - Linux: `scripts/package-calibre-runtime-linux.sh` copies the official
    Calibre Linux runtime into `tools/calibre`.
  - Windows: `scripts/package-calibre-runtime-windows.ps1` copies the official
    Calibre Windows runtime into `tools/calibre`.

Because packaged builds distribute Calibre, packaged distributions are
GPL-3.0-only compatible.

### libusb

- Purpose: direct USB/MTP device access.
- Source: https://libusb.info/
- License: LGPL-2.1-or-later.
- Packaging:
  - macOS: `scripts/package-macos.sh` copies the Homebrew
    `libusb-1.0.0.dylib` into `dist/kidney-darwin-*/lib/`.
  - Linux: `scripts/package-libusb-runtime-linux.sh` copies the linked
    `libusb-1.0.so.*` into `lib/` and patches the binary rpath.
  - Windows: release packaging copies `libusb-1.0.dll` into the archive root.

## Go Module Dependencies

### github.com/hanwen/go-mtpfs

- Purpose: MTP device communication.
- Source: https://github.com/hanwen/go-mtpfs
- License: New BSD License.

Copyright notice from the upstream license:

```text
Copyright (c) 2012 Google Inc. All rights reserved.
```

### github.com/hanwen/usb

- Purpose: USB access used by go-mtpfs.
- Source: https://github.com/hanwen/usb
- License: New BSD License.

Copyright notice from the upstream license:

```text
Copyright (c) 2012 Google Inc. All rights reserved.
```
