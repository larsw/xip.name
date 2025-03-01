#!/bin/sh

set -e

DOCKER_BUILDKIT=1
IMAGE_NAME=${IMAGE_NAME:-"larsw/xip.name"}
DOCKER_TAG=${DOCKER_TAG:-"latest"}

build() {
  TARGET=$1
  TAG="${IMAGE_NAME}:$1-${DOCKER_TAG}"

  docker build \
    --build-arg CREATED=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
    --build-arg COMMIT=$(git rev-parse --short HEAD) \
    --build-arg VERSION=$(git describe --tags --always) \
    --target $TARGET \
    -t $TAG .

  docker tag $TAG "${IMAGE_NAME}:$1-latest"
}

build "minimal"
build "alpine"
build "alpine-web"

