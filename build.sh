#!/bin/bash -e

if [[ $1 = "-loc" ]]; then
    find . -name '*.go' -or -name '*.java' -or -name '*.js' -or -name '*.html' | xargs wc -l | sort -n
    exit
fi

VER=0.1.2rc
ID=$(git rev-parse HEAD | cut -c1-7)

cd daemon/pushd

if [[ $1 = "-mac" ]]; then
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-X github.com/nicholaskh/golib/server.VERSION=$VER -X github.com/nicholaskh/golib/server.BuildID=$ID -w"
    mv pushd ../../bin/pushd.mac
    ../../bin/pushd.mac -v
else
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X github.com/nicholaskh/golib/server.VERSION=$VER -X github.com/nicholaskh/golib/server.BuildID=$ID -w"
    #go build -race -v -ldflags "-X github.com/nicholaskh/golib/server.BuildID=$ID -w"
    mv pushd ../../bin/pushd.linux
fi

cd ../benchmark

if [[ $1 = "-mac" ]]; then
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-X github.com/nicholaskh/golib/server.VERSION=$VER -X github.com/nicholaskh/golib/server.BuildID=$ID -w"
    mv benchmark ../../bin/benchmark.mac
else
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X github.com/nicholaskh/golib/server.VERSION=$VER -X github.com/nicholaskh/golib/server.BuildID=$ID -w"
    #go build -race -v -ldflags "-X github.com/nicholaskh/golib/server.BuildID=$ID -w"
    mv benchmark ../../bin/benchmark.linux
fi

cd ../../cmd/pushd-cli

if [[ $1 = "-mac" ]]; then
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-X github.com/nicholaskh/golib/server.VERSION=$VER -X github.com/nicholaskh/golib/server.BuildID=$ID -w"
    #go build -race -v -ldflags "-X github.com/nicholaskh/golib/server.BuildID=$ID -w"
    mv pushd-cli ../../bin/pushd-cli.linux
else
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X github.com/nicholaskh/golib/server.VERSION=$VER -X github.com/nicholaskh/golib/server.BuildID=$ID -w"
    mv pushd-cli ../../bin/pushd-cli.linux
fi
