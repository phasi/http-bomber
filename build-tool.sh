#!/bin/bash

DOCKERIMAGE="phasi/http_bomber"

clean() {
rm -rf dist/
}

build() {

clean

VERSIONTAG=$(git describe --tags --abbrev=0)

cp -r src temp
mkdir dist

echo """
package main
var AppVersion string = \"${VERSIONTAG}\"
""" > temp/version.go

if [[ $1 == "darwin" ]]; then
    GOOS=darwin GOARCH=amd64 go build -o dist/http-bomber_darwin_amd64 temp/*.go
elif [[ $1 == "linux" ]]; then
    GOOS=linux GOARCH=amd64 go build -o dist/http-bomber_linux_amd64 temp/*.go
elif [[ $1 == "windows" ]]; then
    GOOS=windows GOARCH=amd64 go build -o dist/http-bomber_win_amd64.exe temp/*.go
elif [[ $1 == "docker" ]]; then
    GOOS=linux GOARCH=amd64 go build -o dist/http-bomber_linux_amd64 temp/*.go
    docker build -t $DOCKERIMAGE .
    clean
fi

rm -rf temp
}


"$@"
