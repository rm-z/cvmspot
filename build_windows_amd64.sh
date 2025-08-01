#!/bin/bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o cvmspot_linux_amd64.exe main.go




