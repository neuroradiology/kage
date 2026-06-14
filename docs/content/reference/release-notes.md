---
title: "Release notes"
description: "What changed in each kage release."
weight: 40
---

The authoritative, commit-level history lives in
[`CHANGELOG.md`](https://github.com/tamnd/kage/blob/main/CHANGELOG.md) and on the
[releases page](https://github.com/tamnd/kage/releases). This page summarises
each version.

## v0.1.0

The first release. kage clones a live website into a self-contained folder you
can browse offline, with every script stripped out.

- **`kage clone <url>`** renders each page in headless Chrome, strips all
  JavaScript, and localises CSS, images, and fonts to relative paths.
- **`kage serve [dir]`** previews a cloned folder over a local file server.
- **Idempotent and resumable.** Each page is keyed by the file it writes, so a
  page reached over http and https, or as `/index.html` versus `/`, is fetched
  once. Re-running resumes; `--refresh` re-renders in place; `--force` starts
  clean.
- **Polite by default.** Honours `robots.txt`, seeds from `sitemap.xml`, scopes
  to the seed host, and runs three parallel worker tiers.
- **Packaged everywhere.** Archives, `.deb`/`.rpm`/`.apk`, a multi-arch GHCR
  image with Chromium bundled, checksums, SBOMs, and a cosign signature.
