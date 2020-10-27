#!/usr/bin/env bash

set -euxo pipefail

cd $WORKDIR
mv $WORKDIR/services/go-espeak-demo/demo.yml $WORKDIR/services/go-espeak-demo/docker-compose.yml

mkdir -p $WORKDIR/go-espeak-demo/static/audio

./compose.sh up --detach --remove-orphans --build --force-recreate
