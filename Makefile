VERSION ?= snapshot
ifeq ($(VERSION), snapshot)
	TAG = latest
else
	TAG = $(VERSION)
endif

FIRSTGOPATH = $(firstword $(subst :, ,$(GOPATH)))

gosources = $(shell find . -path "./vendor/*" -prune -o -type f -name "*.go" -print)
TESTS_TO_RUN := .

all: build

$(FIRSTGOPATH)/bin/modvendor:
	go get -u github.com/goware/modvendor

vendor: go.mod go.sum $(FIRSTGOPATH)/bin/modvendor
	mkdir -p $(FIRSTGOPATH)/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.0.3
	go mod vendor
	modvendor -copy="**/*.c **/*.h"

build: $(gosources) go.mod go.sum
	go build ./...

lint: $(FIRSTGOPATH)/bin/golangci-lint build
	$(FIRSTGOPATH)/bin/golangci-lint run --build-tags integration

test: $(gosources) go.mod go.sum $(FIRSTGOPATH)/bin/gotestsum
	gotestsum -- -run $(TESTS_TO_RUN) ./...

ci-test: $(gosources) go.mod go.sum $(FIRSTGOPATH)/bin/gotestsum
	mkdir -p test_results/unit_tests
	gotestsum --junitfile=test_results/unit_tests/results.xml -- -mod=readonly -run $(TESTS_TO_RUN) ./...

TUPELO ?= master
ifeq ($(TUPELO), local)
    TUPELO_DIR ?= ../tupelo
    TUPELO_BUILD_PULL = build
else
    TUPELO_DIR = integration
    TUPELO_BUILD_PULL = pull
endif

integration-test: docker-image integration/* integration/configs/*
	cd $(TUPELO_DIR) && docker-compose -p tupelo down || true
ifeq ($(TUPELO_BUILD_PULL), build)
	cd $(TUPELO_DIR) && make vendor
endif
	cd $(TUPELO_DIR) && docker-compose -p tupelo $(TUPELO_BUILD_PULL) &&\
	docker-compose -p tupelo up --remove-orphans --force-recreate -d
	sleep 5
	docker run -e TUPELO_BOOTSTRAP_NODES=/ip4/172.16.246.10/tcp/34001/ipfs/\
16Uiu2HAm3TGSEKEjagcCojSJeaT5rypaeJMKejijvYSnAjviWwV5 --net tupelo_default \
quorumcontrol/tupelo-go-sdk go test -mod=vendor -tags=integration -timeout=2m -run $(TESTS_TO_RUN) ./... \
	|| (cd $(TUPELO_DIR); docker-compose -p tupelo down; exit 1)
	cd $(TUPELO_DIR) && docker-compose -p tupelo down || true

docker-image: vendor $(gosources) Dockerfile .dockerignore
	docker build -t quorumcontrol/tupelo-go-sdk:$(TAG) .

$(FIRSTGOPATH)/bin/golangci-lint:
	./scripts/download-golangci-lint.sh

$(FIRSTGOPATH)/bin/gotestsum:
	go get gotest.tools/gotestsum

clean:
	go clean ./...
	rm -rf vendor

.PHONY: all build test integration-test ci-test docker-image clean
