#!/bin/bash
REPO=${2:-amazeeio}
TAG=${1:-latest}
IMAGEUPSTREAM_REPO=amazeeiolagoon
echo "Creating image for $REPO/backup-handler:$TAG and pushing to docker hub"
docker build --build-arg IMAGE_REPO=$IMAGEUPSTREAM_REPO -t $REPO/backup-handler:${TAG} . && docker push $REPO/backup-handler:${TAG}
