---
title: "CLI reference"
description: "Every kage command and flag."
weight: 10
---

```
kage [command] [flags]
```

Two commands: `clone` fetches a site into an offline folder, `serve` previews
one. Run `kage <command> --help` for the canonical, up-to-date list.

## kage clone

```
kage clone <url> [flags]
```

Renders each page in headless Chrome, strips all JavaScript, localises CSS,
images, and fonts, and writes a browsable mirror to `<out>/<host>/`.

### Output

| Flag | Default | Meaning |
|------|---------|---------|
| `-o, --out` | `kage-out` | Output root; the mirror lands in `<out>/<host>/` |
| `--reserved` | `_kage` | Reserved directory name for assets and crawl state |
| `-f, --force` | `false` | Delete any existing mirror for the host before crawling |
| `--no-resume` | `false` | Do not read or write resume state |

### Scope

| Flag | Default | Meaning |
|------|---------|---------|
| `-p, --max-pages` | `0` | Stop after N pages (0 = unlimited) |
| `-d, --max-depth` | `0` | Link-follow depth cap (0 = unlimited) |
| `--scope-prefix` | | Only crawl pages whose path starts with this prefix |
| `--subdomains` | `false` | Treat subdomains of the seed host as in scope |
| `--exclude` | | Path prefixes to skip (repeatable) |
| `--traversal` | `bfs` | Frontier order: `bfs` or `dfs` |

### Politeness

| Flag | Default | Meaning |
|------|---------|---------|
| `--no-robots` | `false` | Ignore `robots.txt` |
| `--no-sitemap` | `false` | Do not seed URLs from `sitemap.xml` |
| `--user-agent` | Chrome UA | User-Agent for asset and robots fetches |

### Rendering

| Flag | Default | Meaning |
|------|---------|---------|
| `--scroll` | `false` | Auto-scroll each page to trigger lazy loading |
| `--settle` | `1.5s` | Network-idle quiet period before snapshotting the DOM |
| `--render-timeout` | `30s` | Hard cap per page render |
| `--headful` | `false` | Run Chrome with a visible window (debugging) |
| `--chrome` | | Path to the Chrome/Chromium binary |
| `--control-url` | | Attach to an existing Chrome DevTools endpoint |
| `--keep-noscript` | `false` | Unwrap `<noscript>` content instead of dropping it |

### Concurrency and limits

| Flag | Default | Meaning |
|------|---------|---------|
| `--workers` | `4` | Concurrent page render workers |
| `--asset-workers` | `8` | Concurrent asset download workers |
| `--browser-pages` | `4` | Chrome page-pool size |
| `--max-asset-mb` | `25` | Skip assets larger than N MB |
| `--timeout` | `30s` | Per-request timeout |
| `-q, --quiet` | `false` | Suppress per-page progress lines |

## kage serve

```
kage serve [dir] [flags]
```

Runs a local static file server over a cloned folder. With no `dir`, serves the
current directory.

| Flag | Default | Meaning |
|------|---------|---------|
| `-a, --addr` | `127.0.0.1:8800` | Address to listen on |
