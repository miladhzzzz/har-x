#!/bin/bash

git clone https://github.com/miladhzzzz/har-x.git
cd har-x/capy
go mod download
cd cmd && go build -o ../../bin/capy
cp ../../bin/capy /usr/local/bin