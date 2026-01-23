#!/bin/bash
set -euo pipefail

# Default to current user's GitHub username if not set
IMAGE_TAG="${IMAGE_TAG:-latest}"

# Define image names for both registries
PRIVATE_REGISTRY_IMAGE="registry.connorskees.com/lib/clipboard:${IMAGE_TAG}"
GITHUB_REGISTRY_IMAGE="ghcr.io/connorsapps/clipboard:${IMAGE_TAG}"

# Ensure buildx is available
if ! docker buildx version >/dev/null 2>&1; then
  echo "Docker buildx is not available. Please install Docker buildx."
  exit 1
fi

# Use the current builder, stripping asterisk if present
BUILDER=$(docker buildx ls | awk '/\*/ {gsub("\\*", "", $1); print $1; exit}')
if [ -z "$BUILDER" ]; then
  BUILDER=$(docker buildx ls | awk 'NR==2 {print $1}')
fi

echo "Using buildx builder: $BUILDER"
echo "Building and pushing to:"
echo "  - ${GITHUB_REGISTRY_IMAGE} (GitHub Container Registry)"
echo "  - ${PRIVATE_REGISTRY_IMAGE} (Private Registry)"

docker buildx build \
  --builder "$BUILDER" \
  --platform linux/amd64 \
  -t "$GITHUB_REGISTRY_IMAGE" \
  -t "$PRIVATE_REGISTRY_IMAGE" \
  -f Dockerfile \
  --push \
  .

echo "Images built and pushed successfully:"
echo "  - ${GITHUB_REGISTRY_IMAGE}"
echo "  - ${PRIVATE_REGISTRY_IMAGE}" 