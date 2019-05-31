VERSION ?= snapshot
ifeq ($(VERSION), snapshot)
	TAG = latest
else
	TAG = $(VERSION)
endif

FIRSTGOPATH = $(firstword $(subst :, ,$(GOPATH)))

generated = gossip3/messages/external_gen.go gossip3/messages/external_gen_test.go gossip3/remote/messages_gen.go gossip3/remote/messages_gen_test.go
gosources = $(shell find . -path "./vendor/*" -prune -o -type f -name "*.go" -print)

all: build

$(generated): gossip3/messages/external.go gossip3/remote/messages.go $(FIRSTGOPATH)/bin/msgp
	cd gossip3/messages && go generate
	cd gossip3/remote && go generate

$(FIRSTGOPATH)/bin/modvendor:
	go get -u github.com/goware/modvendor

vendor: go.mod go.sum $(FIRSTGOPATH)/bin/modvendor
	mkdir -p $(FIRSTGOPATH)/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.0.3
	go mod vendor
	modvendor -copy="**/*.c **/*.h"

build: $(gosources) $(generated) go.mod go.sum
	go build ./...

lint: $(FIRSTGOPATH)/bin/golangci-lint build
	$(FIRSTGOPATH)/bin/golangci-lint run --build-tags integration

test: $(gosources) $(generated) go.mod go.sum $(FIRSTGOPATH)/bin/gotestsum
	gotestsum

ci-test: $(gosources) $(generated) go.mod go.sum $(FIRSTGOPATH)/bin/gotestsum
	mkdir -p test_results/unit_tests
	gotestsum --junitfile=test_results/unit_tests/results.xml -- -mod=readonly ./...

integration-test: docker-image
	docker run -e TUPELO_BOOTSTRAP_NODES=/ip4/172.16.238.10/tcp/34001/ipfs/\
16Uiu2HAm3TGSEKEjagcCojSJeaT5rypaeJMKejijvYSnAjviWwV5 --net tupelo_default \
quorumcontrol/tupelo-go-sdk go test -mod=vendor -tags=integration -timeout=2m ./...

docker-image: vendor $(gosources) $(generated) Dockerfile .dockerignore
	docker build -t quorumcontrol/tupelo-go-sdk:$(TAG) .

$(FIRSTGOPATH)/bin/golangci-lint:
	./scripts/download-golangci-lint.sh

$(FIRSTGOPATH)/bin/gotestsum:
	go get gotest.tools/gotestsum

$(FIRSTGOPATH)/bin/msgp:
	go get github.com/tinylib/msgp

clean:
	go clean ./...
	rm -rf vendor

.PHONY: all build test integration-test ci-test docker-image clean
