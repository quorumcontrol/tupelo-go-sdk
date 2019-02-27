VERSION ?= snapshot
ifeq ($(VERSION), snapshot)
	TAG = latest
else
	TAG = $(VERSION)
endif

FIRSTGOPATH = $(firstword $(subst :, ,$(GOPATH)))

gosources = $(shell find . -path "./vendor/*" -prune -o -type f -name "*.go" -print)

all: build

vendor: go.mod go.sum $(FIRSTGOPATH)/bin/modvendor
	go mod vendor
	modvendor -copy="**/*.c **/*.h"

build: vendor $(gosources)
	go build ./...

docker-image: vendor Dockerfile .dockerignore
	docker build -t quorumcontrol/tupelo-go-client:$(TAG) .

$(FIRSTGOPATH)/bin/modvendor:
	go get -u github.com/goware/modvendor

clean:
	go clean
	rm -rf vendor

.PHONY: all build docker-image clean
