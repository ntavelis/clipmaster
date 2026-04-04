#!/usr/bin/env bash
set -euo pipefail

REPO="ntavelis/clipmaster"
BINARY="clipmaster"
INSTALL_DIR="/usr/local/bin"
DESKTOP_DIR="${HOME}/.local/share/applications"

# Detect OS
OS="$(uname -s)"
case "${OS}" in
  Linux)  OS="linux" ;;
  Darwin) OS="darwin" ;;
  *)
    echo "Unsupported OS: ${OS}"
    exit 1
    ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "${ARCH}" in
  x86_64)          ARCH="amd64" ;;
  aarch64 | arm64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: ${ARCH}"
    exit 1
    ;;
esac

echo "Detected: ${OS}/${ARCH}"

# Install runtime dependencies (Linux only)
install_deps_linux() {
  if command -v apt-get &>/dev/null; then
    echo "Installing dependencies via apt..."
    sudo apt-get install -y libgtk-3-0 libwebkit2gtk-4.0-37 wl-clipboard
  elif command -v pacman &>/dev/null; then
    echo "Installing dependencies via pacman..."
    sudo pacman -S --needed --noconfirm gtk3 webkit2gtk wl-clipboard
  elif command -v dnf &>/dev/null; then
    echo "Installing dependencies via dnf..."
    sudo dnf install -y gtk3 webkit2gtk3 wl-clipboard
  elif command -v zypper &>/dev/null; then
    echo "Installing dependencies via zypper..."
    sudo zypper install -y libgtk-3-0 libwebkit2gtk-4_0-37 wl-clipboard
  else
    echo "No supported package manager found (apt, pacman, dnf, zypper)."
    echo "Please install the following libraries manually:"
    echo "  - GTK 3 runtime"
    echo "  - WebKit2GTK 4.0 runtime"
    echo "  - wl-clipboard"
    exit 1
  fi
}

if [ "${OS}" = "linux" ]; then
  install_deps_linux
fi

# Download binary
ASSET_NAME="${BINARY}-${OS}-${ARCH}"
DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${ASSET_NAME}"
TMP_FILE="$(mktemp)"

echo "Downloading ${ASSET_NAME}..."
if command -v curl &>/dev/null; then
  curl -fsSL "${DOWNLOAD_URL}" -o "${TMP_FILE}"
elif command -v wget &>/dev/null; then
  wget -qO "${TMP_FILE}" "${DOWNLOAD_URL}"
else
  echo "Neither curl nor wget found. Please install one and retry."
  exit 1
fi

chmod +x "${TMP_FILE}"
echo "Installing to ${INSTALL_DIR}/${BINARY}..."
sudo mv "${TMP_FILE}" "${INSTALL_DIR}/${BINARY}"

# Create .desktop entry (Linux only)
create_desktop_entry() {
  mkdir -p "${DESKTOP_DIR}"
  cat > "${DESKTOP_DIR}/clipmaster.desktop" <<EOF
[Desktop Entry]
Name=Clipmaster
Comment=Clipboard manager with multi-machine sync
Exec=${INSTALL_DIR}/${BINARY}
Type=Application
Categories=Utility;
Terminal=false
EOF
  echo "Desktop entry created at ${DESKTOP_DIR}/clipmaster.desktop"
}

if [ "${OS}" = "linux" ]; then
  create_desktop_entry
fi

echo "Clipmaster installed successfully. Run 'clipmaster' to start."
