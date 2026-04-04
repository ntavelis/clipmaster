# Clipmaster

A lightweight desktop clipboard manager built with Wails and Vue 3. It tracks your clipboard history, lets you browse and re-copy past items, and is designed for keyboard-first workflows.

- In-memory clipboard history (text only, up to 50 items)
- Keyboard navigation with shortcuts for quick copying (Ctrl+1..9)
- Expandable entries for viewing long text
- Live Omarchy theme support — colors update automatically when you switch themes
- Multi-machine sync via WebRTC (planned)

## Installation

### Option 1 — One-liner (recommended)

The install script detects your OS, architecture, and package manager, installs the required dependencies, and places the binary in `/usr/local/bin`.

```bash
curl -fsSL https://raw.githubusercontent.com/ntavelis/clipmaster/main/install.sh | sh
```

### Option 2 — Manual installation

#### Linux

Install the runtime dependencies for your distro, then download and install the binary.

**Debian / Ubuntu**

```bash
sudo apt install libgtk-3-0 libwebkit2gtk-4.0-37 wl-clipboard
curl -fsSL https://github.com/ntavelis/clipmaster/releases/latest/download/clipmaster-linux-amd64 -o clipmaster
sudo install -m 755 clipmaster /usr/local/bin/clipmaster
```

**Arch Linux**

```bash
sudo pacman -S --needed gtk3 webkit2gtk wl-clipboard
curl -fsSL https://github.com/ntavelis/clipmaster/releases/latest/download/clipmaster-linux-amd64 -o clipmaster
sudo install -m 755 clipmaster /usr/local/bin/clipmaster
```

**Fedora / RHEL**

```bash
sudo dnf install gtk3 webkit2gtk3 wl-clipboard
curl -fsSL https://github.com/ntavelis/clipmaster/releases/latest/download/clipmaster-linux-amd64 -o clipmaster
sudo install -m 755 clipmaster /usr/local/bin/clipmaster
```

**openSUSE**

```bash
sudo zypper install libgtk-3-0 libwebkit2gtk-4_0-37 wl-clipboard
curl -fsSL https://github.com/ntavelis/clipmaster/releases/latest/download/clipmaster-linux-amd64 -o clipmaster
sudo install -m 755 clipmaster /usr/local/bin/clipmaster
```

> For ARM64 machines replace `clipmaster-linux-amd64` with `clipmaster-linux-arm64`.

#### macOS

No extra dependencies needed — macOS ships with WebKit.

```bash
# Intel
curl -fsSL https://github.com/ntavelis/clipmaster/releases/latest/download/clipmaster-darwin-amd64 -o clipmaster
sudo install -m 755 clipmaster /usr/local/bin/clipmaster

# Apple Silicon (M1/M2/M3)
curl -fsSL https://github.com/ntavelis/clipmaster/releases/latest/download/clipmaster-darwin-arm64 -o clipmaster
sudo install -m 755 clipmaster /usr/local/bin/clipmaster
```

## Live Development

To run in live development mode, run `wails dev` in the project directory. This will run a Vite development
server that will provide very fast hot reload of your frontend changes. If you want to develop in a browser
and have access to your Go methods, there is also a dev server that runs on http://localhost:34115. Connect
to this in your browser, and you can call your Go code from devtools.

## Building

To build a redistributable, production mode package, use `wails build`.
