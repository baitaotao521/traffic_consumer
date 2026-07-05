#!/usr/bin/env bash
set -Eeuo pipefail

usage() {
  cat <<'USAGE'
Usage:
  scripts/package.sh [options]

Options:
  --all                    Build Linux, Windows, and macOS release packages.
  --dist DIR               Write packages to DIR. Defaults to ./dist.
  --skip-frontend          Reuse the current frontend/dist build.
  --skip-frontend-install  Do not run npm ci before the frontend build.
  -h, --help               Show this help.

Environment:
  TARGET_OS / GOOS         Target operating system for a single package.
  TARGET_ARCH / GOARCH     Target architecture for a single package.
  PLATFORM_LABEL           Release label, for example Linux or Windows.
  DIST_DIR                 Output directory when --dist is not provided.
  BINARY_BASE              Binary name inside the package. Defaults to traffic_consumer.
  CGO_ENABLED              CGO switch for go build. Defaults to 0.
  LDFLAGS                  Go linker flags. Defaults to "-s -w".
  SKIP_FRONTEND=1          Same as --skip-frontend.
  SKIP_FRONTEND_INSTALL=1  Same as --skip-frontend-install.
USAGE
}

log() {
  printf '==> %s\n' "$*"
}

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "missing required command: $1"
}

label_for() {
  local target_os="$1"
  local target_arch="$2"

  case "${target_os}/${target_arch}" in
    linux/amd64) printf 'Linux' ;;
    windows/amd64) printf 'Windows' ;;
    darwin/arm64) printf 'macOS' ;;
    *) printf '%s_%s' "${target_os}" "${target_arch}" ;;
  esac
}

archive_format_for() {
  local target_os="$1"

  if [[ "${target_os}" == "windows" ]]; then
    printf 'zip'
  else
    printf 'tar.gz'
  fi
}

build_frontend() {
  [[ "${run_frontend}" == "1" ]] || return 0

  require_cmd npm
  if [[ "${frontend_install}" == "1" ]]; then
    # CI uses a clean dependency install; local runs only install when node_modules is absent.
    if [[ "${CI:-}" == "true" || ! -d "${repo_root}/frontend/node_modules" ]]; then
      log "Installing frontend dependencies"
      npm --prefix "${repo_root}/frontend" ci
    fi
  fi

  log "Building Vue dashboard"
  npm --prefix "${repo_root}/frontend" run build
}

copy_runtime_assets() {
  local package_dir="$1"

  [[ -f "${repo_root}/frontend/dist/index.html" ]] || fail "frontend/dist/index.html is missing; run without --skip-frontend first"

  # The Go server reads all frontend runtime files from BaseDir/frontend/dist.
  mkdir -p "${package_dir}/frontend"
  cp -R "${repo_root}/frontend/dist" "${package_dir}/frontend/"
}

smoke_test_if_native() {
  local target_os="$1"
  local target_arch="$2"
  local binary_path="$3"

  if [[ "${target_os}" == "${host_os}" && "${target_arch}" == "${host_arch}" ]]; then
    log "Smoke testing ${binary_path}"
    "${binary_path}" --help >/dev/null 2>&1
  fi
}

create_archive() {
  local package_dir="$1"
  local archive_base="$2"
  local format="$3"

  [[ -d "${package_dir}" ]] || fail "package directory disappeared: ${package_dir}"

  case "${format}" in
    zip)
      require_cmd zip
      rm -f "${dist_dir}/${archive_base}.zip"
      (cd "${dist_dir}" && zip -qr "${archive_base}.zip" "${archive_base}")
      printf '%s/%s.zip' "${dist_dir}" "${archive_base}"
      ;;
    tar.gz)
      rm -f "${dist_dir}/${archive_base}.tar.gz"
      tar -C "${dist_dir}" -czf "${dist_dir}/${archive_base}.tar.gz" "${archive_base}"
      printf '%s/%s.tar.gz' "${dist_dir}" "${archive_base}"
      ;;
    *)
      fail "unsupported archive format: ${format}"
      ;;
  esac
}

build_package() {
  local target_os="$1"
  local target_arch="$2"
  local label="$3"
  local format
  local archive_base
  local package_dir
  local binary_name
  local binary_path
  local archive_path

  format="$(archive_format_for "${target_os}")"
  archive_base="traffic_consumer_${label}"
  package_dir="${dist_dir}/${archive_base}"
  binary_name="${binary_base}"

  if [[ "${target_os}" == "windows" && "${binary_name}" != *.exe ]]; then
    binary_name="${binary_name}.exe"
  fi

  rm -rf "${package_dir}"
  mkdir -p "${package_dir}"
  copy_runtime_assets "${package_dir}"

  binary_path="${package_dir}/${binary_name}"
  log "Building ${label} (${target_os}/${target_arch})"
  (
    cd "${repo_root}"
    env CGO_ENABLED="${CGO_ENABLED:-0}" GOOS="${target_os}" GOARCH="${target_arch}" \
      go build -trimpath -ldflags="${LDFLAGS:--s -w}" -o "${binary_path}" .
  )

  smoke_test_if_native "${target_os}" "${target_arch}" "${binary_path}"
  archive_path="$(create_archive "${package_dir}" "${archive_base}" "${format}")"
  log "Created ${archive_path}"

  if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
    {
      printf 'package_path=%s\n' "${archive_path}"
      printf 'package_name=%s\n' "$(basename "${archive_path}")"
      printf 'binary_path=%s\n' "${binary_path}"
    } >>"${GITHUB_OUTPUT}"
  fi
}

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
dist_dir="${DIST_DIR:-${repo_root}/dist}"
run_frontend=1
frontend_install=1
build_all=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --all)
      build_all=1
      ;;
    --dist)
      shift
      [[ $# -gt 0 ]] || fail "--dist requires a directory"
      dist_dir="$1"
      ;;
    --dist=*)
      dist_dir="${1#--dist=}"
      ;;
    --skip-frontend | --no-frontend)
      run_frontend=0
      ;;
    --skip-frontend-install)
      frontend_install=0
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      fail "unknown option: $1"
      ;;
  esac
  shift
done

if [[ "${SKIP_FRONTEND:-}" == "1" ]]; then
  run_frontend=0
fi
if [[ "${SKIP_FRONTEND_INSTALL:-}" == "1" ]]; then
  frontend_install=0
fi

require_cmd go
host_os="$(go env GOHOSTOS)"
host_arch="$(go env GOHOSTARCH)"
binary_base="${BINARY_BASE:-traffic_consumer}"

mkdir -p "${dist_dir}"
dist_dir="$(cd "${dist_dir}" && pwd)"

build_frontend

if [[ "${build_all}" == "1" ]]; then
  build_package linux amd64 Linux
  build_package windows amd64 Windows
  build_package darwin arm64 macOS
else
  target_os="${TARGET_OS:-${GOOS:-${host_os}}}"
  target_arch="${TARGET_ARCH:-${GOARCH:-${host_arch}}}"
  platform_label="${PLATFORM_LABEL:-$(label_for "${target_os}" "${target_arch}")}"
  build_package "${target_os}" "${target_arch}" "${platform_label}"
fi
