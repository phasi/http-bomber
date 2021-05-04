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
    cd temp && GOOS=darwin GOARCH=amd64 go build -o ../dist/http-bomber_darwin_amd64 *.go && cd ..
elif [[ $1 == "linux" ]]; then
    cd temp && GOOS=linux GOARCH=amd64 go build -o ../dist/http-bomber_linux_amd64 *.go && cd ..
elif [[ $1 == "windows" ]]; then
    cd temp && GOOS=windows GOARCH=amd64 go build -o ../dist/http-bomber_win_amd64.exe *.go && cd ..
elif [[ $1 == "docker" ]]; then
    cd temp && GOOS=linux GOARCH=amd64 go build -o ../dist/http-bomber_linux_amd64 *.go && cd ..
    docker build -t $DOCKERIMAGE .
    clean
fi

rm -rf temp
}


"$@"
