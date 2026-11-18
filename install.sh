#!/bin/sh
# tidetop 설치 스크립트 / installer
#   curl -fsSL https://raw.githubusercontent.com/tidepark/tidetop/main/install.sh | sh
#   wget -qO- https://raw.githubusercontent.com/tidepark/tidetop/main/install.sh | sh
# TIDETOP_VERSION=0.1.0 으로 판을 고르고, TIDETOP_INSTALL_DIR 로 깔 곳을 고른다.
set -eu

repo="tidepark/tidetop"
say() { printf '%s\n' "$*" >&2; }
# curl 이 없으면 wget 으로 받는다
if command -v curl >/dev/null 2>&1; then
  get() { curl -fsSL "$1" -o "$2"; }
  read_url() { curl -fsSL "$1"; }
elif command -v wget >/dev/null 2>&1; then
  get() { wget -q "$1" -O "$2"; }
  read_url() { wget -qO- "$1"; }
else
  printf '%s\n' "tidetop: curl or wget is needed" >&2
  exit 1
fi
fail() { say "tidetop: $*"; exit 1; }

case "$(uname -s)" in
  Darwin) os=darwin ;;
  Linux) os=linux ;;
  *) fail "unsupported system: $(uname -s) (download from https://github.com/$repo/releases)" ;;
esac
case "$(uname -m)" in
  x86_64 | amd64) arch=amd64 ;;
  arm64 | aarch64) arch=arm64 ;;
  *) fail "unsupported CPU: $(uname -m)" ;;
esac

version="${TIDETOP_VERSION:-}"
if [ -z "$version" ]; then
  version="$(read_url "https://api.github.com/repos/$repo/releases/latest" | sed -n 's/.*"tag_name": *"v\{0,1\}\([^"]*\)".*/\1/p' | head -n 1)"
  [ -n "$version" ] || fail "could not find the latest version"
fi

name="tidetop_${version}_${os}_${arch}.tar.gz"
base="https://github.com/$repo/releases/download/v$version"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

say "tidetop $version ($os/$arch)"
get "$base/$name" "$tmp/$name" || fail "download failed: $base/$name"
get "$base/checksums.txt" "$tmp/checksums.txt" || fail "download failed: checksums.txt"

want="$(grep " $name\$" "$tmp/checksums.txt" | cut -d ' ' -f 1)"
if command -v sha256sum >/dev/null 2>&1; then
  got="$(sha256sum "$tmp/$name" | cut -d ' ' -f 1)"
else
  got="$(shasum -a 256 "$tmp/$name" | cut -d ' ' -f 1)"
fi
[ -n "$want" ] && [ "$want" = "$got" ] || fail "checksum does not match"

tar -xzf "$tmp/$name" -C "$tmp"

dir="${TIDETOP_INSTALL_DIR:-}"
if [ -z "$dir" ]; then
  if [ -w /usr/local/bin ]; then dir=/usr/local/bin; else dir="$HOME/.local/bin"; fi
fi
mkdir -p "$dir"
install -m 0755 "$tmp/tidetop" "$dir/tidetop"
say "installed: $dir/tidetop"
# man 페이지: 실행 파일 옆의 share/man 에 둔다(쓸 수 있을 때만)
man="$(dirname "$dir")/share/man"
if mkdir -p "$man/man1" "$man/ko/man1" 2>/dev/null && [ -w "$man/man1" ]; then
  install -m 0644 "$tmp/man/tidetop.1" "$man/man1/tidetop.1"
  install -m 0644 "$tmp/man/tidetop.ko.1" "$man/ko/man1/tidetop.1"
  say "man page: $man/man1/tidetop.1"
fi
case ":$PATH:" in
  *":$dir:"*) ;;
  *) say "add $dir to PATH to run tidetop" ;;
esac
