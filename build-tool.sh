#!/bin/bash

clean() {
rm -rf dist/
}

build() {

git checkout main

clean()


VERSIONTAG=$(git describe --tags --abbrev=0)

cp -r src temp
mkdir dist

echo """
package main
var AppVersion string = \"${VERSIONTAG}\"
""" > temp/version.go


go build -o dist/http-bomber temp/*.go

rm -rf temp
}



"$@"
