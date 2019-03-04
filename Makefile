VERSION ?= snapshot
ifeq ($(VERSION), snapshot)
	TAG = latest
else
	TAG = $(VERSION)
endif

FIRSTGOPATH = $(firstword $(subst :, ,$(GOPATH)))

generated = gossip3/messages/external_gen.go gossip3/messages/external_gen_test.go
gosources = $(shell find . -path "./vendor/*" -prune -o -type f -name "*.go" -print)

all: build

$(generated): gossip3/messages/external.go
	cd gossip3/messages && go generate

vendor: Gopkg.toml Gopkg.lock
	dep ensure

build: vendor $(gosources) $(generated)
	go build ./...

test: vendor $(gosources)
	go test ./...

integration-test: docker-image
	docker run -e TUPELO_BOOTSTRAP_NODES=/ip4/172.16.238.10/tcp/34001/ipfs/\
QmW2hgZqe6UcQ6kTaF8kS6CA3RDo7wbCvnGctCetbSt85n --net tupelo_default \
quorumcontrol/tupelo-go-client go test -tags=integration -timeout=2m ./...

docker-image: vendor Dockerfile .dockerignore
	docker build -t quorumcontrol/tupelo-go-client:$(TAG) .

$(FIRSTGOPATH)/bin/modvendor:
	go get -u github.com/goware/modvendor

clean:
	go clean
	rm -rf vendor

.PHONY: all vendor build test integration-test docker-image clean
