# Clipmaster — Project Instructions

## Overview

A Wails desktop clipboard manager that tracks clipboard history and syncs across multiple computers on the local network.

## Tech Stack

- **Backend**: Go (Wails v2)
- **Frontend**: Vue 3 + Vite + Pinia
- **CSS**: TailwindCSS only — no other CSS frameworks, no custom CSS classes outside of Tailwind utilities
- **Peer Discovery**: mDNS via `hashicorp/mdns` (`_clipmaster._tcp` service type)
- **Sync Transport**: HTTPS with self-signed TLS certificates (no CA — `InsecureSkipVerify`)

## Features

1. **Clipboard history** — continuously poll/monitor the system clipboard and store a history of copied items (text only for now)
2. **History UI** — browse and re-copy from clipboard history
3. **Multi-machine sync** — peers discover each other via mDNS and pull each other's clipboard over HTTPS
4. **Omarchy theming** — automatically reads the active Omarchy color theme and applies it to the UI; colors update live when the theme changes
5. **Remote clipboard disable** — `--remote-clipboards-disable` flag skips networking entirely for single-machine use

## Clipboard History

- In-memory only (no persistence across restarts)
- Default max: 50 items (`CLIPMASTER_CLIPBOARD_MAX_HISTORY`)
- Text only (no image support yet)
- Wayland-aware: uses `wl-paste`/`wl-copy` if available, falls back to Wails runtime clipboard

## Multi-Machine Sync (mDNS + HTTPS)

- Each instance starts an HTTPS server on a random OS-assigned port with a self-signed RSA-2048 cert
- mDNS advertises the port and a SHA-256 hash (first 8 bytes) of the passphrase in TXT records
- Peers filter by passphrase hash — only instances sharing the same passphrase connect
- Peer fetcher polls all discovered peers at 1s intervals: `GET /api/clipboard` with `X-Clipmaster-Pass` header
- Handler uses `subtle.ConstantTimeCompare` to validate the passphrase (timing-attack resistant)
- `InsecureSkipVerify` is intentional — certs are self-signed with no CA; mDNS passphrase filtering is the security boundary
- Peers expire after ~3 poll cycles (~6s) if not seen again

## Passphrase

- Minimum 8 chars, maximum 128 chars, no leading/trailing whitespace
- Stored in `$HOME/.config/clipmaster/config.json` (mode 0600) as `{ "passphrase": "..." }`
- Loaded at startup; if invalid, the app logs an error and exits immediately
- On first launch (no config), the UI shows a passphrase setup screen before enabling networking
- If `DisableRemoteClipboards` is true, passphrase is never required

## Omarchy Theming

- Reads `$HOME/.config/omarchy/current/theme/colors.toml`; gracefully skips if missing
- Watches file for changes (fsnotify, 200ms debounce) and emits `theme:loaded` Wails event
- All UI colors come from Omarchy theme tokens via CSS custom properties (`--color-*`); no hardcoded colors

## App Behavior

- Normal window (no system tray)
- No global hotkey yet
- Wails events: `clipboard:new`, `remote:updated`, `theme:loaded`

## Architecture Notes

- `app/app.go` — main Wails struct; exposes Go methods to the frontend via `Bind`
- `app/routes.go` — registers HTTP routes on the sync server
- `app/handlers/` — HTTP handlers (clipboard endpoint)
- `business/clipboard/` — clipboard monitor, history, reader/writer interfaces
- `business/passphrase/` — thread-safe passphrase store + validation
- `business/sync/` — HTTPS server wrapper
- `business/peersclipsync/` — periodic fetcher of remote peer clipboards
- `business/theme/` — Omarchy theme loader and file watcher
- `foundation/clipboard/` — platform clipboard backends (Wayland + Wails runtime)
- `foundation/config/` — JSON config file read/write
- `foundation/mdns/` — mDNS registration and peer discovery
- `foundation/tlscert/` — self-signed TLS certificate generation
- Frontend Vue stores (Pinia): `clipboard.js`, `remote.js`, `theme.js`, shared `navigation.js` composable

## CLI Flags

All configurable via environment variables (`CLIPMASTER_<FLAG>`) or command-line args:

| Flag | Default | Description |
|------|---------|-------------|
| `CLIPMASTER_DEBUG` | `false` | Enable debug-level logging |
| `CLIPMASTER_CLIPBOARD_MAX_HISTORY` | `50` | Max local clipboard entries |
| `CLIPMASTER_CLIPBOARD_POLL_INTERVAL` | `500ms` | Local clipboard poll frequency |
| `CLIPMASTER_REMOTE_CLIPBOARDS_MAX_HISTORY` | `3` | Max remote entries shown per peer |
| `CLIPMASTER_REMOTE_CLIPBOARDS_POLL_INTERVAL` | `1s` | Peer fetch frequency |
| `CLIPMASTER_REMOTE_CLIPBOARDS_DISABLE` | `false` | Disable remote sync entirely |
| `CLIPMASTER_PEERS_POLL_INTERVAL` | `2s` | mDNS browse frequency |
| `CLIPMASTER_THEME_COLOR_PATH` | `~/.config/omarchy/current/theme/colors.toml` | Omarchy theme file path |
| `CLIPMASTER_CONFIG_PATH` | `~/.config/clipmaster/config.json` | Config file path |

## Code Style

- No inline comments on every line — comments only on top of functions
- No custom CSS — Tailwind utility classes only
- Keep Go packages small and focused
