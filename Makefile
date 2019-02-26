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

test: $(gosources)
	go test ./...
	
integration-test: docker-image
	docker run -v /var/run/docker.sock:/var/run/docker.sock -v $(PWD):/src quorumcontrol/tupelo-integration-runner

docker-image: vendor Dockerfile .dockerignore
	docker build -t quorumcontrol/tupelo-go-client:$(TAG) .

$(FIRSTGOPATH)/bin/modvendor:
	go get -u github.com/goware/modvendor

clean:
	go clean
	rm -rf vendor

.PHONY: all build test integration-test docker-image clean
