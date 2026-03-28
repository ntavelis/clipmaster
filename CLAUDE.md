# Clipmaster — Project Instructions

## Overview
A Wails desktop clipboard manager that tracks clipboard history and syncs across multiple computers via WebRTC.

## Tech Stack
- **Backend**: Go (Wails v2)
- **Frontend**: Vue 3 + Vite
- **CSS**: TailwindCSS only — no other CSS frameworks, no custom CSS classes outside of Tailwind utilities
- **P2P Sync**: WebRTC (pion/webrtc for Go, native browser WebRTC on frontend)
- **Signalling**: A lightweight Go signalling server (separate binary or embedded)

## Features
1. **Clipboard history** — continuously poll/monitor the system clipboard and store a history of copied items (text only for now)
2. **History UI** — browse, search, and re-copy from clipboard history
3. **Multi-machine sync** — bidirectional; all peers share entries with everyone in real-time over WebRTC data channels
4. **Omarchy theming** — automatically reads the active Omarchy color theme and applies it to the UI; colors update live when the theme changes

## Clipboard History
- In-memory only (no persistence across restarts) for now
- Default max: 50 items; must be user-configurable
- Text only (no image support yet)

## Multi-Machine Sync (WebRTC)
- One instance acts as **host** (runs embedded signalling server); others connect to it
- Peers discover each other via a shared **passphrase**
- Sync is **bidirectional** — all connected peers share clipboard entries with everyone

## Omarchy Theming
- Active theme name is stored in `~/.config/omarchy/current/theme.name`
- Watch that file for changes and update the UI dynamically via Wails events
- All colors in the UI must come from Omarchy theme tokens; no hardcoded colors

## App Behavior
- Normal window (no system tray for now)
- No global hotkey configured yet — to be decided later

## Architecture Notes
- `app.go` — main Wails struct; expose Go methods to the frontend via `Bind`
- Clipboard polling runs as a goroutine started in `startup()`
- WebRTC peer connection logic lives in its own Go package (`internal/rtc/`)
- Signalling server is **embedded** in the main binary — one instance acts as host, others connect to it via passphrase
- Frontend uses Wails' generated TypeScript bindings to call Go methods
- Vue stores (Pinia) manage clipboard history and peer connection state

## Code Style
- No inline comments on every line — comments only on top of functions
- No custom CSS — Tailwind utility classes only
- Keep Go packages small and focused
