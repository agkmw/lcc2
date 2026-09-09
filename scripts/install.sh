#!/usr/bin/env sh
# install.sh — Automated installer for lcc
# Checks platform prerequisites, installs dependencies, compiles, and installs lcc.
set -e

# --- 1. Platform Verification ---
OS="$(uname -s)"
if [ "$OS" != "Linux" ]; then
  echo "Error: lcc requires Linux (/proc, /sys, systemd, Linux process semantics)." >&2
  echo "Detected OS: $OS" >&2
  exit 1
fi

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64|aarch64|arm64|armv7l|armv6l)
    ;;
  *)
    echo "Warning: Unverified architecture ($ARCH). Attempting build anyway..." >&2
    ;;
esac

# --- 2. Configuration & Arguments ---
SKIP_DEPS=0
ASSUME_YES=0
DEFAULT_BIN_DIR="${HOME}/.local/bin"
if [ "$(id -u)" -eq 0 ]; then
  DEFAULT_BIN_DIR="/usr/local/bin"
fi
BIN_DIR="${INSTALL_DIR:-$DEFAULT_BIN_DIR}"

print_usage() {
  cat <<EOF
Usage: ./scripts/install.sh [options]

Options:
  --bin-dir <dir>   Installation directory for lcc binary (default: $DEFAULT_BIN_DIR)
  --no-deps         Skip installing system package dependencies
  -y, --yes         Non-interactive mode (pass yes to package manager)
  -h, --help        Show this help message
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    --bin-dir)
      if [ -n "${2:-}" ]; then
        BIN_DIR="$2"
        shift 2
      else
        echo "Error: --bin-dir requires a path argument" >&2
        exit 1
      fi
      ;;
    --no-deps)
      SKIP_DEPS=1
      shift
      ;;
    -y|--yes)
      ASSUME_YES=1
      shift
      ;;
    -h|--help)
      print_usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      print_usage
      exit 1
      ;;
  esac
done

mkdir -p "$BIN_DIR"

# --- 3. Locate Source Directory ---
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

if [ ! -f "$REPO_DIR/go.mod" ] || ! grep -q "module lcc" "$REPO_DIR/go.mod" 2>/dev/null; then
  echo "=> go.mod not found in parent directory. Cloning lcc repository..."
  CLONE_DIR="$(mktemp -d)"
  trap 'rm -rf "$CLONE_DIR"' EXIT
  if ! command -v git >/dev/null 2>&1; then
    echo "Error: git is required to clone the lcc repository." >&2
    exit 1
  fi
  git clone https://github.com/agkmw/lcc.git "$CLONE_DIR/lcc"
  REPO_DIR="$CLONE_DIR/lcc"
fi

# --- 4. Package Manager & Dependency Resolution ---
SUDO=""
if [ "$(id -u)" -ne 0 ] && command -v sudo >/dev/null 2>&1; then
  SUDO="sudo"
fi

detect_pkg_mgr() {
  if command -v apt-get >/dev/null 2>&1; then
    echo "apt"
  elif command -v dnf >/dev/null 2>&1; then
    echo "dnf"
  elif command -v pacman >/dev/null 2>&1; then
    echo "pacman"
  elif command -v zypper >/dev/null 2>&1; then
    echo "zypper"
  elif command -v apk >/dev/null 2>&1; then
    echo "apk"
  else
    echo "unknown"
  fi
}

install_packages() {
  PKG_MGR="$(detect_pkg_mgr)"
  PKGS="$*"
  if [ -z "$PKGS" ]; then
    return 0
  fi

  echo "=> Installing dependencies ($PKGS) via $PKG_MGR..."
  case "$PKG_MGR" in
    apt)
      $SUDO apt-get update -y
      # shellcheck disable=SC2086
      $SUDO apt-get install -y $PKGS
      ;;
    dnf)
      # shellcheck disable=SC2086
      $SUDO dnf install -y $PKGS
      ;;
    pacman)
      # shellcheck disable=SC2086
      $SUDO pacman -S --needed --noconfirm $PKGS
      ;;
    zypper)
      # shellcheck disable=SC2086
      $SUDO zypper --non-interactive install $PKGS
      ;;
    apk)
      # shellcheck disable=SC2086
      $SUDO apk add $PKGS
      ;;
    *)
      echo "Warning: Could not detect supported package manager. Please install the following manually: $PKGS" >&2
      ;;
  esac
}

if [ "$SKIP_DEPS" -eq 0 ]; then
  PKG_MGR="$(detect_pkg_mgr)"
  NEEDED_PKGS=""

  # Check Go
  if ! command -v go >/dev/null 2>&1; then
    case "$PKG_MGR" in
      apt|dnf) NEEDED_PKGS="$NEEDED_PKGS golang" ;;
      pacman|zypper|apk) NEEDED_PKGS="$NEEDED_PKGS go" ;;
    esac
  fi

  # Check fd (file search)
  if ! command -v fd >/dev/null 2>&1 && ! command -v fdfind >/dev/null 2>&1; then
    case "$PKG_MGR" in
      apt|dnf) NEEDED_PKGS="$NEEDED_PKGS fd-find" ;;
      pacman|zypper|apk) NEEDED_PKGS="$NEEDED_PKGS fd" ;;
    esac
  fi

  # Check ripgrep (content grep)
  if ! command -v rg >/dev/null 2>&1; then
    NEEDED_PKGS="$NEEDED_PKGS ripgrep"
  fi

  # Check gio (glib trash backend)
  if ! command -v gio >/dev/null 2>&1; then
    case "$PKG_MGR" in
      apt) NEEDED_PKGS="$NEEDED_PKGS libglib2.0-bin" ;;
      dnf|pacman|zypper) NEEDED_PKGS="$NEEDED_PKGS glib2" ;;
      apk) NEEDED_PKGS="$NEEDED_PKGS glib-tools" ;;
    esac
  fi

  # Check clipboard helper (wl-copy or xclip)
  if ! command -v wl-copy >/dev/null 2>&1 && ! command -v xclip >/dev/null 2>&1; then
    case "$PKG_MGR" in
      apt|dnf|pacman|zypper|apk) NEEDED_PKGS="$NEEDED_PKGS wl-clipboard xclip" ;;
    esac
  fi

  # Check less (default pager)
  if ! command -v less >/dev/null 2>&1; then
    NEEDED_PKGS="$NEEDED_PKGS less"
  fi

  # Check a terminal editor: the files 'e' and services 'E' gestures
  # need one that can save; the app probes $EDITOR/$VISUAL, then
  # nvim/vim/vi/nano/editor.
  if ! command -v nvim >/dev/null 2>&1 && ! command -v vim >/dev/null 2>&1 \
     && ! command -v vi >/dev/null 2>&1 && ! command -v nano >/dev/null 2>&1 \
     && ! command -v editor >/dev/null 2>&1; then
    NEEDED_PKGS="$NEEDED_PKGS nano"
  fi

  # Trim leading spaces
  NEEDED_PKGS="$(echo "$NEEDED_PKGS" | sed 's/^[[:space:]]*//')"
  if [ -n "$NEEDED_PKGS" ]; then
    install_packages "$NEEDED_PKGS"
  else
    echo "=> All runtime and build tools already present."
  fi
fi

# --- 5. Fix Debian/Ubuntu fd symlink if needed ---
# On Debian/Ubuntu systems, fd is installed as 'fdfind'. lcc searches for 'fd'.
if ! command -v fd >/dev/null 2>&1 && command -v fdfind >/dev/null 2>&1; then
  FDFIND_PATH="$(command -v fdfind)"
  echo "=> Creating symlink for fd -> $FDFIND_PATH in $BIN_DIR..."
  ln -sf "$FDFIND_PATH" "$BIN_DIR/fd"
fi

# --- 6. Go Version Check ---
if ! command -v go >/dev/null 2>&1; then
  echo "Error: Go compiler is not installed. Please install Go 1.26+ (https://go.dev/dl/)." >&2
  exit 1
fi

GO_VERSION="$(go version | awk '{print $3}' | sed 's/^go//')"
GO_MAJOR="$(echo "$GO_VERSION" | cut -d. -f1)"
GO_MINOR="$(echo "$GO_VERSION" | cut -d. -f2)"

if [ -n "$GO_MAJOR" ] && [ -n "$GO_MINOR" ]; then
  if [ "$GO_MAJOR" -lt 1 ] || { [ "$GO_MAJOR" -eq 1 ] && [ "$GO_MINOR" -lt 26 ]; }; then
    echo "Warning: lcc specifies Go 1.26+ (detected go$GO_VERSION). Build will be attempted, but newer Go features may be required." >&2
  fi
fi

# --- 7. Build and Install lcc ---
echo "=> Building lcc binary from $REPO_DIR..."
TARGET_BIN="$BIN_DIR/lcc"
(
  cd "$REPO_DIR"
  CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$TARGET_BIN" ./cmd/lcc
)

chmod +x "$TARGET_BIN"
echo "=> Successfully installed lcc to $TARGET_BIN"

# --- 8. PATH Verification & Notes ---
case ":$PATH:" in
  *":$BIN_DIR:"*)
    ;;
  *)
    echo ""
    echo "Notice: $BIN_DIR is not in your PATH."
    echo "Add it to your shell configuration (e.g. ~/.bashrc or ~/.zshrc):"
    echo "    export PATH=\"$BIN_DIR:\$PATH\""
    ;;
esac

echo ""
echo "lcc is ready to run!"
echo "  Run:         lcc"
echo "  With root:   sudo lcc (required for service control and account management)"
echo "  Help overlay: press '?' at any time inside the app"
