#!/bin/bash

VERSIONTAG=$(git describe --tags --abbrev=0)

build() {

cp -r src temp
mkdir dist

echo """
package main
var AppVersion string = \"${VERSIONTAG}\"
""" > temp/version.go


go build -o dist/http-bomber temp/*.go

rm -rf temp
}

clean() {
rm -rf dist/
}

"$@"
