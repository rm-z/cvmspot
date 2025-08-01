#!/bin/bash
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -o main_static_armv7 main.go
