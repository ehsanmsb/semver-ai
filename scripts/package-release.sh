#!/bin/sh
set -eu
version="$1"
case "$version" in ''|*[!0-9.]*) echo 'Expected stable numeric SemVer' >&2; exit 1;; esac
: "${IMAGE_NAME:?IMAGE_NAME is required}"
mkdir -p dist
for os in linux darwin; do
  for arch in amd64 arm64; do
    CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath \
      -ldflags="-s -w -X main.buildVersion=$version" \
      -o "dist/release-manager_${os}_${arch}" ./cmd/release-manager
  done
done
(cd dist && sha256sum release-manager_* > SHA256SUMS)
# Publish the image before creating the Git tag so a build failure remains retryable.
docker buildx build --platform linux/amd64,linux/arm64 --push \
  --build-arg "VERSION=$version" \
  --label "org.opencontainers.image.source=https://github.com/$GITHUB_REPOSITORY" \
  --label "org.opencontainers.image.revision=$GITHUB_SHA" \
  --tag "$IMAGE_NAME:$version" --tag "$IMAGE_NAME:${version%.*}" \
  --tag "$IMAGE_NAME:${version%%.*}" --tag "$IMAGE_NAME:latest" .
