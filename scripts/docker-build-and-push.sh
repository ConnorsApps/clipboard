#!/bin/bash
set -euo pipefail

IMAGE_TAG="${IMAGE_TAG:-latest}"
USE_PODMAN=false

for arg in "$@"; do
  case $arg in
    --podman) USE_PODMAN=true ;;
    *) echo "Unknown argument: $arg"; exit 1 ;;
  esac
done

PRIVATE_REGISTRY_IMAGE="registry.connorskees.com/lib/clipboard:${IMAGE_TAG}"
GITHUB_REGISTRY_IMAGE="ghcr.io/connorsapps/clipboard:${IMAGE_TAG}"

echo "Building and pushing to:"
echo "  - ${GITHUB_REGISTRY_IMAGE} (GitHub Container Registry)"
echo "  - ${PRIVATE_REGISTRY_IMAGE} (Private Registry)"

if $USE_PODMAN; then
  podman build \
    --platform linux/amd64 \
    -t "$GITHUB_REGISTRY_IMAGE" \
    -t "$PRIVATE_REGISTRY_IMAGE" \
    -f Dockerfile \
    .

  podman push "$GITHUB_REGISTRY_IMAGE"
  podman push "$PRIVATE_REGISTRY_IMAGE"
else
  if ! docker buildx version >/dev/null 2>&1; then
    echo "Docker buildx is not available. Please install Docker buildx."
    exit 1
  fi

  BUILDER=$(docker buildx ls | awk '/\*/ {gsub("\\*", "", $1); print $1; exit}')
  if [ -z "$BUILDER" ]; then
    BUILDER=$(docker buildx ls | awk 'NR==2 {print $1}')
  fi

  echo "Using buildx builder: $BUILDER"

  docker buildx build \
    --builder "$BUILDER" \
    --platform linux/amd64 \
    -t "$GITHUB_REGISTRY_IMAGE" \
    -t "$PRIVATE_REGISTRY_IMAGE" \
    -f Dockerfile \
    --push \
    .
fi

echo "Images built and pushed successfully:"
echo "  - ${GITHUB_REGISTRY_IMAGE}"
echo "  - ${PRIVATE_REGISTRY_IMAGE}"
