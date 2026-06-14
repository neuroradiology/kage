---
title: "Configuration"
description: "Environment variables kage reads, and the layout of a cloned mirror on disk."
weight: 20
---

kage is configured almost entirely through command-line flags (see the
[CLI reference](/reference/cli/)). It reads a couple of environment variables for
locating the browser.

## Environment variables

| Variable | Meaning |
|----------|---------|
| `KAGE_CHROME` | Path to the Chrome/Chromium binary. Takes precedence over autodetection. Equivalent to `--chrome`. |
| `CHROME_BIN` | Fallback Chrome path, read if `KAGE_CHROME` is unset. |

If neither is set and no system Chrome is found in the usual install locations,
kage's launcher can download a private copy of Chromium on first use.

## Output layout

A clone of `example.com` lands under `kage-out/example.com/`:

```
kage-out/example.com/
├── index.html                  # the home page (/), scripts stripped
├── about/index.html            # /about
├── blog/
│   ├── index.html              # /blog
│   └── a-post/index.html       # /blog/a-post
├── _kage/                      # reserved directory
│   ├── example.com/
│   │   ├── site.css            # localised stylesheet, url() rewritten
│   │   ├── logo.png
│   │   └── fonts/body.woff2
│   ├── cdn.example.com/        # assets from other hosts, by host
│   └── state.json              # visited set, for --resume
└── ...
```

Key points:

- **Pages become directories.** A page at `/about` is written as
  `about/index.html`, so a link to `/about` resolves to a real file when served.
- **Assets live under the reserved directory.** Everything kage downloads, CSS,
  images, fonts, media, goes under `_kage/<asset-host>/`, mirroring the path it
  had on its origin. Cross-origin assets are grouped by their own host.
- **Query strings are folded into the filename.** An asset like
  `style.css?v=3` is saved with a short hash suffix so two versions never
  collide.
- **State lives in the mirror.** `_kage/state.json` records every page written,
  which is what makes `--resume` able to skip completed work. Rename the reserved
  directory with `--reserved` if `_kage` would clash with a real path on the site.
