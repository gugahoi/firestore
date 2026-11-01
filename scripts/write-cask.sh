#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 3 ]]; then
    echo "usage: $0 <tag> <checksums-file> <output-file>" >&2
    exit 64
fi

TAG="$1"
CHECKSUMS_FILE="$2"
OUTPUT_FILE="$3"
VERSION="${TAG#v}"

if [[ ! "$TAG" =~ ^v[0-9] ]]; then
    echo "tag must start with v, for example v1.0.0" >&2
    exit 65
fi

# Look up an archive's sha256 in GoReleaser's checksums.txt ("<sha256>  <file>").
sha_for() {
    local sha
    sha="$(awk -v f="$1" '$2 == f { print $1 }' "$CHECKSUMS_FILE")"
    if [[ ! "$sha" =~ ^[0-9a-f]{64}$ ]]; then
        echo "no sha256 for $1 in $CHECKSUMS_FILE" >&2
        exit 66
    fi
    echo "$sha"
}

DARWIN_ARM64="$(sha_for firestore_Darwin_arm64.tar.gz)"
DARWIN_X86_64="$(sha_for firestore_Darwin_x86_64.tar.gz)"
LINUX_ARM64="$(sha_for firestore_Linux_arm64.tar.gz)"
LINUX_X86_64="$(sha_for firestore_Linux_x86_64.tar.gz)"

mkdir -p "$(dirname "$OUTPUT_FILE")"

cat >"$OUTPUT_FILE" <<CASK
cask "firestore" do
  version "${VERSION}"

  on_macos do
    on_arm do
      sha256 "${DARWIN_ARM64}"
      url "https://github.com/gugahoi/firestore/releases/download/v#{version}/firestore_Darwin_arm64.tar.gz"
    end
    on_intel do
      sha256 "${DARWIN_X86_64}"
      url "https://github.com/gugahoi/firestore/releases/download/v#{version}/firestore_Darwin_x86_64.tar.gz"
    end
  end
  on_linux do
    on_arm do
      sha256 "${LINUX_ARM64}"
      url "https://github.com/gugahoi/firestore/releases/download/v#{version}/firestore_Linux_arm64.tar.gz"
    end
    on_intel do
      sha256 "${LINUX_X86_64}"
      url "https://github.com/gugahoi/firestore/releases/download/v#{version}/firestore_Linux_x86_64.tar.gz"
    end
  end

  name "firestore"
  desc "Command-line utility to facilitate operations with Firestore"
  homepage "https://github.com/gugahoi/firestore"

  livecheck do
    skip "Auto-generated on release."
  end

  binary "firestore"

  postflight_steps do
    run "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "{{staged_path}}/firestore"], must_succeed: false
  end

  zap :not_required
end
CASK
