# Third-Party Notices

Kidney depends on and may package third-party components. This file is a
human-readable summary; individual dependency source distributions remain the
authoritative license texts.

## Bundled In Packaged Builds

### Calibre

- Purpose: EPUB to AZW3 conversion through `ebook-convert`.
- Source: https://calibre-ebook.com/
- License: GPL-3.0-only.
- Packaging: `scripts/package-calibre-runtime-macos.sh` copies a pruned Calibre
  runtime into `dist/kidney-darwin-*/tools/calibre.app`.

Because packaged builds distribute Calibre, packaged distributions are
GPL-3.0-only compatible. Do not replace the pruned runtime with a full
`calibre.app` bundle without a separate product decision.

### libusb

- Purpose: direct USB/MTP device access.
- Source: https://libusb.info/
- License: LGPL-2.1-or-later.
- Packaging: `scripts/package-macos.sh` copies the Homebrew
  `libusb-1.0.0.dylib` into `dist/kidney-darwin-*/lib/`.

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
