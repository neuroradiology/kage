---
title: "Installation"
description: "Install kage from Go, Homebrew, a release archive, a Linux package, or the container image, and point it at a browser."
weight: 20
---

kage is a single binary. Pick whichever channel suits you.

## Go

```bash
go install github.com/tamnd/kage/cmd/kage@latest
```

## Homebrew

```bash
brew install tamnd/tap/kage
```

## Release archives and Linux packages

Every [release](https://github.com/tamnd/kage/releases) attaches `tar.gz`
archives (and a `.zip` for Windows) for Linux, macOS, Windows, and FreeBSD, plus
`.deb`, `.rpm`, and `.apk` packages and a `checksums.txt` with a cosign
signature. Download the one for your platform, extract `kage`, and put it on your
`PATH`.

```bash
# Debian/Ubuntu
sudo dpkg -i kage_*_linux_amd64.deb

# Fedora/RHEL
sudo rpm -i kage_*_linux_amd64.rpm
```

## Container

The image bundles Chromium, so it needs nothing else:

```bash
docker run -v "$PWD/out:/out" ghcr.io/tamnd/kage clone example.com
```

The mirror lands in `./out/example.com/` on your host.

## You need a browser

kage drives a real Chrome to render pages. Outside the container image, it needs
Chrome or Chromium available on the machine. It looks for a system install
automatically (Google Chrome on macOS and Windows, `google-chrome`/`chromium` on
Linux). To use a specific binary:

```bash
kage clone example.com --chrome /path/to/chromium
# or
export KAGE_CHROME=/path/to/chromium
```

If no browser is found, kage's launcher can download a private copy of Chromium
on first use.

Next: [the quick start](/getting-started/quick-start/).
