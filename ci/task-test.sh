#!/bin/sh

GOPATH=$(pwd)/go
rootdir=$(dirname $(dirname $0))

cd $rootdir

go vet ./...
go test -v ./...

