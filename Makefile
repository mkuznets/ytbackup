all: ytbackup

ytbackup:
	go generate ./...
	go build -ldflags="-s -w" --tags "json1" mkuznets.com/go/ytbackup/cmd/ytbackup

.PHONY: ytbackup
