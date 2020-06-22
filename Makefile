VERSION := $(shell cat VERSION)
REVISION := $(shell scripts/git-rev.sh)
BUILD_TIME := $(shell TZ=Etc/UTC date +'%Y-%m-%dT%H:%M:%SZ')

VERSION_FLAG := -X mkuznets.com/go/ytbackup/internal/version.version=${VERSION}
REVISION_FLAG := -X mkuznets.com/go/ytbackup/internal/version.revision=${REVISION}
BUILD_TIME_FLAG := -X mkuznets.com/go/ytbackup/internal/version.buildTime=${BUILD_TIME}
LDFLAGS := "-s -w ${VERSION_FLAG} ${REVISION_FLAG} ${BUILD_TIME_FLAG}"

all: ytbackup

ytbackup:
	go generate ./...
	export CGO_ENABLED=0
	go build -ldflags=${LDFLAGS} mkuznets.com/go/ytbackup/cmd/ytbackup

.PHONY: ytbackup
