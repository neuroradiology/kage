# Changelog

All notable changes to kage are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/), and the project aims to follow
[Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.1.0] - 2026-06-14

The first release. kage clones a live website into a self-contained folder you
can browse offline, with every script stripped out.

### Added

- `kage clone <url>` renders each page in headless Chrome, snapshots the final
  DOM, removes every `<script>`, `on*` handler, and `javascript:` URL, and
  downloads the CSS, images, fonts, and media, rewriting them to local paths.
- `kage serve [dir]` runs a local static file server over a cloned folder so the
  mirror's links and assets resolve the way they would on a real host.
- Deterministic URL-to-path mapping: pages become `<slug>/index.html`
  directories, assets live under the reserved `_kage/<host>/` tree, and query
  strings fold into a short hash suffix so versioned URLs never collide.
- Three concurrency tiers run in parallel: page-render workers (`--workers`),
  asset-download workers (`--asset-workers`), and a Chrome page pool
  (`--browser-pages`).
- A polite crawl by default: honours `robots.txt`, seeds from `sitemap.xml`,
  and scopes to the seed host. `--scope-prefix`, `--max-depth`, `--max-pages`,
  `--subdomains`, and `--exclude` shape the frontier.
- Idempotent, resumable crawling. Each page is keyed by the file it writes, so
  the same URL reached over http and https, with or without a trailing slash,
  or as `/index.html` versus `/`, is fetched exactly once. A re-run resumes from
  `_kage/state.json`; `--refresh` re-renders a mirror in place to pull in
  changed content; `--force` wipes and starts clean; `--no-resume` runs
  stateless.
- Defaults to a per-user data directory (`$HOME/data/kage`), overridable with
  `-o/--out`.
- Cross-platform distribution: prebuilt archives, `.deb`/`.rpm`/`.apk` packages,
  a multi-arch container image on GHCR (Chromium bundled), checksums, SBOMs, and
  a cosign signature, all cut from one version tag by GoReleaser.

[Unreleased]: https://github.com/tamnd/kage/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/tamnd/kage/releases/tag/v0.1.0
