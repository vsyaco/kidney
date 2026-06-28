# Third-Party Notices

Kidney depends on and may package third-party components. This file is a
human-readable summary; individual dependency source distributions remain the
authoritative license texts.

## External Runtime Dependencies

### Calibre

- Purpose: EPUB to AZW3 conversion through `ebook-convert`.
- Source: https://calibre-ebook.com/
- Packaging: not bundled. Do not copy full `calibre.app` into `dist` without a
  separate product and licensing decision.

## Bundled In Packaged Builds

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
