#!/bin/sh

GOOS=windows GOARCH=amd64 go build -ldflags "-s" -o adlg_win64.exe deleg.go
GOOS=linux GOARCH=amd64 go build -ldflags "-s" -o adlg_lin64 deleg.go
GOOS=darwin GOARCH=amd64 go build -ldflags "-s" -o adlg_mac64 deleg.go