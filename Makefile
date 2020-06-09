all: ytbackup

ytbackup:
	go generate ./...
	go build -ldflags="-s -w" mkuznets.com/go/ytbackup/cmd/ytbackup

.PHONY: ytbackup
