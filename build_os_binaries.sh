#!/bin/bash

for OSNAME in linux darwin windows; do

  echo "Buidling v$1 for ${OSNAME}..."
  GOOS=${OSNAME} GOARCH=amd64 go build -ldflags "-s -w" -o bin/cloudflarebeat-$1-${OSNAME}-amd64
  tar -czvf bin/cloudflarebeat-$1-${OSNAME}-amd64.tar.gz bin/cloudflarebeat-$1-${OSNAME}-amd64
  zip bin/cloudflarebeat-$1-${OSNAME}-amd64.zip bin/cloudflarebeat-$1-${OSNAME}-amd64
  rm bin/cloudflarebeat-$1-${OSNAME}-amd64

done
