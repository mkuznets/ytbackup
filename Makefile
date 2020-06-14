all: ytbackup

ytbackup:
	go generate ./...
	CGO_ENABLED=0 go build -ldflags="-s -w" mkuznets.com/go/ytbackup/cmd/ytbackup

.PHONY: ytbackup
