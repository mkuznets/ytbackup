all: ytbackup

ytbackup:
	go generate ./...
	go build --tags "json1" mkuznets.com/go/ytbackup/cmd/ytbackup

.PHONY: ytbackup
