#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

detect_os() {
	case "$(uname -s)" in
		Linux*) echo "linux" ;;
		Darwin*) echo "darwin" ;;
		MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
		*) echo "unknown" ;;
	esac
}

detect_arch() {
	if command -v go >/dev/null 2>&1; then
		go env GOARCH
		return
	fi
	case "$(uname -m)" in
		x86_64|amd64) echo "amd64" ;;
		arm64|aarch64) echo "arm64" ;;
		*) uname -m ;;
	esac
}

detect_jobs() {
	if command -v nproc >/dev/null 2>&1; then
		nproc
	elif command -v sysctl >/dev/null 2>&1; then
		sysctl -n hw.ncpu
	else
		echo 2
	fi
}

TARGET_OS="${TARGET_OS:-$(detect_os)}"
TARGET_ARCH="${TARGET_ARCH:-$(detect_arch)}"
JOBS="${JOBS:-$(detect_jobs)}"
BUILD_DIR="${BUILD_DIR:-$ROOT_DIR/build/ffmpeg-$TARGET_OS-$TARGET_ARCH-src}"
PREFIX="${PREFIX:-$ROOT_DIR/build/ffmpeg-$TARGET_OS-$TARGET_ARCH}"

LAME_VERSION="${LAME_VERSION:-3.100}"
FFMPEG_REF="${FFMPEG_REF:-n7.1.1}"

case "$TARGET_OS/$TARGET_ARCH" in
	windows/amd64)
		HOST="${HOST:-x86_64-w64-mingw32}"
		FFMPEG_TARGET_ARGS=(--target-os=mingw32 --arch=x86_64 --cross-prefix="$HOST-" --enable-cross-compile)
		;;
	linux/*|darwin/*)
		HOST="${HOST:-}"
		FFMPEG_TARGET_ARGS=()
		;;
	*)
		echo "Unsupported target: $TARGET_OS/$TARGET_ARCH" >&2
		exit 1
		;;
esac

mkdir -p "$BUILD_DIR" "$PREFIX"

export PKG_CONFIG_ALLOW_CROSS=1
export PKG_CONFIG_PATH="$PREFIX/lib/pkgconfig"
export PKG_CONFIG_LIBDIR="$PREFIX/lib/pkgconfig"
export CFLAGS="${CFLAGS:-} -O2 -I$PREFIX/include"
export LDFLAGS="${LDFLAGS:-} -L$PREFIX/lib"

fetch_tarball() {
	local url="$1"
	local out="$2"
	if [ ! -f "$out" ]; then
		curl -L "$url" -o "$out"
	fi
}

extract_once() {
	local tarball="$1"
	local dir="$2"
	if [ ! -d "$dir" ]; then
		tar -xf "$tarball" -C "$BUILD_DIR"
	fi
}

clone_once() {
	local url="$1"
	local ref="$2"
	local dir="$3"
	if [ ! -d "$dir/.git" ]; then
		git clone --depth 1 --branch "$ref" "$url" "$dir"
	fi
}

build_autotools() {
	local dir="$1"
	shift
	cd "$dir"
	if [ ! -x ./configure ]; then
		if [ -x ./autogen.sh ]; then
			./autogen.sh
		else
			autoreconf -fiv
		fi
	fi
	local host_args=()
	if [ -n "$HOST" ]; then
		host_args=(--host="$HOST")
	fi
	./configure \
		"${host_args[@]}" \
		--prefix="$PREFIX" \
		--disable-shared \
		--enable-static \
		"$@"
	make -j"$JOBS"
	make install
}

cd "$BUILD_DIR"

fetch_tarball "https://downloads.sourceforge.net/project/lame/lame/$LAME_VERSION/lame-$LAME_VERSION.tar.gz" "$BUILD_DIR/lame-$LAME_VERSION.tar.gz"
extract_once "$BUILD_DIR/lame-$LAME_VERSION.tar.gz" "$BUILD_DIR/lame-$LAME_VERSION"
build_autotools "$BUILD_DIR/lame-$LAME_VERSION" --disable-frontend --disable-decoder

clone_once "https://github.com/FFmpeg/FFmpeg.git" "$FFMPEG_REF" "$BUILD_DIR/ffmpeg"
cd "$BUILD_DIR/ffmpeg"

./configure \
	--prefix="$PREFIX" \
	"${FFMPEG_TARGET_ARGS[@]}" \
	--pkg-config=pkg-config \
	--pkg-config-flags=--static \
	--disable-shared \
	--enable-static \
	--disable-debug \
	--disable-doc \
	--disable-programs \
	--disable-autodetect \
	--disable-everything \
	--enable-gpl \
	--enable-version3 \
	--enable-avcodec \
	--enable-avformat \
	--enable-avutil \
	--enable-swresample \
	--enable-protocol=file \
	--enable-parser=aac,mpegaudio,opus \
	--enable-demuxer=aac,mp3,mov,ogg,wav,s16le,s24le,s32le,f32le \
	--enable-decoder=aac,aac_fixed,mp3,mp3float,opus,pcm_f32be,pcm_f32le,pcm_f64be,pcm_f64le,pcm_s16be,pcm_s16le,pcm_s24be,pcm_s24le,pcm_s32be,pcm_s32le,pcm_s64be,pcm_s64le,pcm_s8 \
	--enable-encoder=libmp3lame \
	--enable-muxer=mp3 \
	--enable-libmp3lame

make -j"$JOBS"
make install
