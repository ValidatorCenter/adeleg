GOARCH=amd64 go build -ldflags "-s" -o adlg_lin64 deleg.go
GOOS=darwin go build -ldflags "-s" -o adlg_mac deleg.go