#!/bin/bash

cd capy
go mod download
cd cmd && go build -o ../../bin/capy